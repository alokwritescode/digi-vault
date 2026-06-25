# DigiVault

> Production-grade digital wallet API in Go

**Status:** 🚧 In active development

## Architecture

[diagram coming after all services are scaffolded]

## Services

| Service | Port | Responsibility |
|---|---|---|
| api-gateway | 8080 | JWT auth, rate limiting, reverse proxy |
| auth-service | 8081 | Registration, login, JWT, OTP, RBAC |
| wallet-service | 8082 | Balance, credit, debit, transfer |
| txn-service | 8083 | Transaction history, Elasticsearch search |
| notification-service | 8084 | Pub/Sub consumer, event notifications |

## Tech Stack

Go · Gin · GORM · MySQL · Redis · gRPC · Google Cloud Pub/Sub · Elasticsearch · Docker Compose · Logrus · Viper

## Setup

[coming after auth-service is complete]