#!/bin/bash
set -e

echo "🔧 Building AtonixCorp Container Runtime..."

ROOT_DIR=$(dirname "$0")/..
cd "$ROOT_DIR"

echo "➡ Building API service..."
go build -o bin/api ./cmd/api

echo "➡ Building Scheduler service..."
go build -o bin/scheduler ./cmd/scheduler

echo "➡ Building Node Agent..."
go build -o bin/node-agent ./cmd/node-agent

echo "✅ Build completed successfully."
