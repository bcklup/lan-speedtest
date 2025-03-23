#!/bin/bash

# Function to cleanup background processes on script termination
cleanup() {
    echo "Shutting down..."
    kill $(jobs -p) 2>/dev/null
    exit
}

# Set up trap for cleanup on script termination
trap cleanup SIGINT SIGTERM

# Get the local IP address
LOCAL_IP=$(ipconfig getifaddr en0 || ipconfig getifaddr en1 || echo "localhost")

# Default values
WS_PORT=":8080"
TEST_PORT=":3001"
CHUNK_SIZE="8388608"  # 8MB in bytes

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --ws-port)
      WS_PORT="$2"
      shift 2
      ;;
    --test-port)
      TEST_PORT="$2"
      shift 2
      ;;
    --chunk-size)
      CHUNK_SIZE="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Store the original directory
ORIGINAL_DIR=$(pwd)

# Start backend
echo "Starting speed test server..."
echo "WebSocket server: $LOCAL_IP$WS_PORT"
echo "Test server: $LOCAL_IP$TEST_PORT"
echo "Chunk size: $((CHUNK_SIZE/1024/1024))MB"

cd backend && go run main.go \
  -addr="$WS_PORT" \
  -test-addr="$TEST_PORT" \
  -chunk-size="$CHUNK_SIZE" &
BACKEND_PID=$!

# Return to original directory
cd "$ORIGINAL_DIR"

# Start frontend
cd frontend && npm run run &
FRONTEND_PID=$!

# Wait for both processes
wait $BACKEND_PID $FRONTEND_PID 