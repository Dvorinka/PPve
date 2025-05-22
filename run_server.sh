#!/bin/bash
# Run server in background
nohup go run main.go > server.log 2>&1 &
echo "Server started in background"
echo "PID: $!"
echo "Logs: server.log"
echo "For kontakt service, run in new terminal: cd kontakt && sudo make dev"
