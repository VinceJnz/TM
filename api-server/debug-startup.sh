#!/bin/sh
set -e

echo "Building application..."
cd /src/api-server
go mod download
go build -gcflags='all=-N -l' -o /apiserver .

echo "Starting debugger..."
dlv exec /apiserver --headless --listen=:40000 --api-version=2 --accept-multiclient &
DEBUGGER_PID=$!

# Function to restart debugger
restart_debugger() {
    echo "Restarting debugger..."
    kill $DEBUGGER_PID 2>/dev/null || true
    go build -gcflags='all=-N -l' -o /apiserver .
    dlv exec /apiserver --headless --listen=:40000 --api-version=2 --accept-multiclient &
    DEBUGGER_PID=$!
}

# Trap SIGUSR1 to restart debugger
trap restart_debugger USR1

echo "Debugger started with PID: $DEBUGGER_PID"
echo "Container ready. To restart debugger, run: docker kill -s USR1 apiserver-debug"
echo "To access shell, run: docker exec -it apiserver-debug /bin/sh"

# Keep container alive
while true; do
    if ! kill -0 $DEBUGGER_PID 2>/dev/null; then
        echo "Debugger process died, restarting..."
        restart_debugger
    fi
    sleep 10
done