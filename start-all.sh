#!/bin/bash

# WIM Service - Start All Services
# This script starts all three services in separate terminal windows

echo "========================================"
echo "  Starting WIM Services"
echo "========================================"
echo ""

# Check if running on macOS or Linux
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS - use Terminal
    echo "Starting API Server..."
    osascript -e 'tell app "Terminal" to do script "cd '"$(pwd)"' && go run cmd/api/main.go"'

    sleep 1

    echo "Starting ANPR Watcher..."
    osascript -e 'tell app "Terminal" to do script "cd '"$(pwd)"' && go run cmd/anpr-watcher/main.go"'

    sleep 1

    echo "Starting AXLE Watcher..."
    osascript -e 'tell app "Terminal" to do script "cd '"$(pwd)"' && go run cmd/axle-watcher/main.go"'

else
    # Linux - use gnome-terminal or xterm
    if command -v gnome-terminal &> /dev/null; then
        gnome-terminal -- bash -c "cd $(pwd) && go run cmd/api/main.go; exec bash"
        sleep 1
        gnome-terminal -- bash -c "cd $(pwd) && go run cmd/anpr-watcher/main.go; exec bash"
        sleep 1
        gnome-terminal -- bash -c "cd $(pwd) && go run cmd/axle-watcher/main.go; exec bash"
    else
        echo "Please install gnome-terminal or run services manually:"
        echo "  Terminal 1: go run cmd/api/main.go"
        echo "  Terminal 2: go run cmd/anpr-watcher/main.go"
        echo "  Terminal 3: go run cmd/axle-watcher/main.go"
        exit 1
    fi
fi

echo ""
echo "✓ All services started in separate terminals"
echo ""
echo "Services running:"
echo "  - API Server (Terminal 1)"
echo "  - ANPR Watcher (Terminal 2)"
echo "  - AXLE Watcher (Terminal 3)"
echo ""
echo "To stop: Close each terminal or press Ctrl+C in each"
echo "========================================"
