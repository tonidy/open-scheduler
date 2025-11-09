#!/bin/bash

set -e

echo "==> Building Centro server..."
go build -o centro_server ./centro

echo "==> Building Agent client..."
go build -o agent_client ./agent

echo "==> Starting etcd (make sure it's running elsewhere or in another terminal if not)..."
if ! pgrep -x etcd > /dev/null; then
    echo " [!] etcd not detected in process list. You must start etcd first (e.g. 'etcd &' or via Docker)."
fi

echo "==> Starting Centro server in background..."
./centro_server --etcd-endpoints=localhost:2379 > centro_server.log 2>&1 &
CENTRO_PID=$!
echo "Centro server PID: $CENTRO_PID"

sleep 2

echo "==> Starting Agent client in background..."
./agent_client --server=localhost:50051 --token=test-token > agent_client.log 2>&1 &
AGENT_PID=$!
echo "Agent client PID: $AGENT_PID"

echo ""
echo "==> Logs:"
echo "   Tail server logs: tail -f centro_server.log"
echo "   Tail agent logs:  tail -f agent_client.log"
echo ""
echo "==> To stop all, run: kill $CENTRO_PID $AGENT_PID"

wait
