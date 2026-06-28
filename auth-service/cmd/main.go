package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/alokwritescode/digi-vault/auth-service/config"
	"github.com/alokwritescode/digi-vault/auth-service/internal/domain"
	"github.com/alokwritescode/digi-vault/auth-service/internal/handler"
	authgrpc "github.com/alokwritescode/digi-vault/auth-service/internal/grpc"
	"github.com/alokwritescode/digi-vault/auth-service/internal/repository"
	"github.com/alokwritescode/digi-vault/auth-service/internal/usecase"
	pb "github.com/alokwritescode/digi-vault/proto/auth"
)

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("failed to load config")
	}

	db, err := openDB(cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to database")
	}

	rdb := openRedis(cfg)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.WithError(err).Fatal("failed to connect to redis")
	}

	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(rdb)
	otpStore := repository.NewOTPStore(rdb)

	uc, err := usecase.NewAuthUsecase(userRepo, tokenRepo, otpStore, cfg, log)
	if err != nil {
		log.WithError(err).Fatal("failed to init usecase")
	}

	// ─── HTTP server ─────────────────────────────────────────────────────────
	h := handler.NewAuthHandler(uc)

	r := gin.New()
	r.Use(gin.Recovery())

	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/send-otp", h.SendOTP)
		auth.POST("/verify-otp", h.VerifyOTP)
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
		auth.POST("/refresh", h.Refresh)
	}

	httpSrv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	go func() {
		log.WithField("port", cfg.AppPort).Info("HTTP server starting")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("HTTP server error")
		}
	}()

	// ─── gRPC server ─────────────────────────────────────────────────────────
	grpcSrv := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcSrv, authgrpc.NewAuthServer(userRepo, cfg.JWTAccessSecret))

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.WithError(err).Fatal("failed to bind gRPC port")
	}

	go func() {
		log.WithField("port", cfg.GRPCPort).Info("gRPC server starting")
		if err := grpcSrv.Serve(lis); err != nil {
			log.WithError(err).Fatal("gRPC server error")
		}
	}()

	// ─── Graceful shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("HTTP forced shutdown")
	}

	grpcSrv.GracefulStop()
	log.Info("auth-service stopped")
}

func openDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}
	return db, nil
}

func openRedis(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost + ":" + cfg.RedisPort,
		Password: cfg.RedisPassword,
	})
}
