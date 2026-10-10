#!/bin/bash
set -e

echo " Starting AtonixCorp Container Runtime (local mode)..."

ROOT_DIR=$(dirname "$0")/..
cd "$ROOT_DIR"

# Build binaries if missing
if [ ! -f "bin/api" ] || [ ! -f "bin/scheduler" ] || [ ! -f "bin/node-agent" ]; then
    echo " Binaries missing — running build.sh first..."
    ./scripts/build.sh
fi

echo "➡ Starting Scheduler on :7070..."
./bin/scheduler &

echo "➡ Starting Node Agent on :9090..."
./bin/node-agent &

echo "➡ Starting API on :8080..."
./bin/api &

echo " Local runtime environment is running."
echo "   API:        http://localhost:8080"
echo "   Scheduler:  http://localhost:7070"
echo "   Node Agent: http://localhost:9090"
