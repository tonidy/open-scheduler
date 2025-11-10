#!/bin/bash
pkill etcd
nohup etcd --data-dir=default.etcd --listen-client-urls=http://localhost:2379 --advertise-client-urls=http://localhost:2379 > etcd.log 2>&1 &
podman stop $(podman ps -aq)
podman rm $(podman ps -aq)
set -e
# export XDG_RUNTIME_DIR="unix:///run/user/501"
echo "==> Cleaning etcd job and node data..."
etcdctl --endpoints=localhost:2379 del --prefix "/centro/" || {
    echo " [!] Failed to clean etcd. Is etcd running and etcdctl installed?"
}

echo "==> Building Centro server..."
go build -o centro_server ./centro

echo "==> Building Agent client..."
go build -o agent_client ./agent



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
