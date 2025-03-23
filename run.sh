#!/bin/bash

# Function to cleanup background processes on script termination
cleanup() {
    echo "Shutting down..."
    kill $(jobs -p) 2>/dev/null
    exit
}

# Set up trap for cleanup on script termination
trap cleanup SIGINT SIGTERM

# Store the original directory
ORIGINAL_DIR=$(pwd)

# Start backend
cd backend && go run main.go &
BACKEND_PID=$!

# Return to original directory
cd "$ORIGINAL_DIR"

# Start frontend
cd frontend && npm run run &
FRONTEND_PID=$!

# Wait for both processes
wait $BACKEND_PID $FRONTEND_PID 