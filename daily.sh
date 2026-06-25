#!/bin/bash
# Run from digi-vault root
echo "🚀 Starting digi-vault daily automation..."
cd ~/desktop/git/digi-vault
git pull origin main  # sync latest
claude  # opens Claude Code in interactive mode, reads CLAUDE.md automatically