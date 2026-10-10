#!/bin/bash
set -e

echo "  AtonixCorp CI Pipeline Starting..."

ROOT_DIR=$(dirname "$0")/..
cd "$ROOT_DIR"

echo " Running Go formatting..."
go fmt ./...

echo " Running Go vet..."
go vet ./...

echo "Running tests..."
go test ./... -cover

echo " Building binaries..."
./scripts/build.sh

echo " Packaging artifacts..."
mkdir -p ci-artifacts
cp bin/* ci-artifacts/

echo "CI pipeline completed successfully."
