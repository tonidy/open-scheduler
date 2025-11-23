#!/bin/bash
# Quick start Open Scheduler with TLS certificates
# Usage: ./scripts/start-with-tls.sh [mode]
# Modes: generate, start-server, start-agent

set -e

CERTS_DIR="${CERTS_DIR:-./.certs}"
DOMAIN="${DOMAIN:-localhost}"
MODE="${1:-help}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

function print_header() {
  echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
  echo -e "${BLUE}║  Open Scheduler TLS Quick Start        ║${NC}"
  echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
}

function check_certs() {
  if [ ! -f "$CERTS_DIR/server-cert.pem" ] || [ ! -f "$CERTS_DIR/ca-cert.pem" ]; then
    echo -e "${RED}✗ Certificates not found in $CERTS_DIR${NC}"
    echo -e "${YELLOW}Run: ./scripts/start-with-tls.sh generate${NC}"
    exit 1
  fi
}

function generate_certs() {
  print_header
  echo ""
  echo -e "${YELLOW}Generating self-signed certificates...${NC}"
  echo ""

  if [ ! -f "./scripts/generate-certs.sh" ]; then
    echo -e "${RED}✗ generate-certs.sh not found${NC}"
    exit 1
  fi

  chmod +x ./scripts/generate-certs.sh
  ./scripts/generate-certs.sh "$DOMAIN" "$CERTS_DIR"
}

function start_server() {
  print_header
  echo ""
  check_certs

  echo -e "${GREEN}Starting Centro with TLS...${NC}"
  echo ""
  echo -e "${BLUE}Configuration:${NC}"
  echo "  Domain: $DOMAIN"
  echo "  Certs Dir: $CERTS_DIR"
  echo "  gRPC Port: 50051"
  echo "  REST Port: 8080"
  echo ""
  echo -e "${YELLOW}Environment variables set:${NC}"
  echo "  GRPC_SERVER_TLS_ENABLED=true"
  echo "  GRPC_SERVER_TLS_CERT_FILE=$CERTS_DIR/server-cert.pem"
  echo "  GRPC_SERVER_TLS_KEY_FILE=$CERTS_DIR/server-key.pem"
  echo "  GRPC_SERVER_TLS_CA_FILE=$CERTS_DIR/ca-cert.pem"
  echo ""

  # Export for subprocess
  export GRPC_SERVER_TLS_ENABLED=true
  export GRPC_SERVER_TLS_CERT_FILE="$CERTS_DIR/server-cert.pem"
  export GRPC_SERVER_TLS_KEY_FILE="$CERTS_DIR/server-key.pem"
  export GRPC_SERVER_TLS_CA_FILE="$CERTS_DIR/ca-cert.pem"

  # Optional: set other security settings
  if [ -z "$JWT_SECRET" ]; then
    export JWT_SECRET="dev-secret-key-change-in-production"
    echo -e "${YELLOW}JWT_SECRET using dev default (not set)${NC}"
  fi

  if [ -z "$ADMIN_PASSWORD" ]; then
    export ADMIN_PASSWORD="admin123"
    echo -e "${YELLOW}ADMIN_PASSWORD using dev default (not set)${NC}"
  fi

  echo ""
  echo -e "${GREEN}Starting Centro...${NC}"
  ./centro_server --etcd-endpoints localhost:2379
}

function start_agent() {
  print_header
  echo ""
  check_certs

  NODE_ID="${2:-$(hostname)}"

  echo -e "${GREEN}Starting Agent with TLS...${NC}"
  echo ""
  echo -e "${BLUE}Configuration:${NC}"
  echo "  Node ID: $NODE_ID"
  echo "  Server: $DOMAIN:50051"
  echo "  Certs Dir: $CERTS_DIR"
  echo ""
  echo -e "${YELLOW}Environment variables set:${NC}"
  echo "  GRPC_TLS_ENABLED=true"
  echo "  GRPC_TLS_CERT_FILE=$CERTS_DIR/client-cert.pem"
  echo "  GRPC_TLS_KEY_FILE=$CERTS_DIR/client-key.pem"
  echo "  GRPC_TLS_CA_FILE=$CERTS_DIR/ca-cert.pem"
  echo ""

  # Export for subprocess
  export GRPC_TLS_ENABLED=true
  export GRPC_TLS_CERT_FILE="$CERTS_DIR/client-cert.pem"
  export GRPC_TLS_KEY_FILE="$CERTS_DIR/client-key.pem"
  export GRPC_TLS_CA_FILE="$CERTS_DIR/ca-cert.pem"

  echo -e "${GREEN}Starting Agent...${NC}"
  ./agent_client \
    --server "$DOMAIN:50051" \
    --token "test-token" \
    --node-id "$NODE_ID"
}

function print_help() {
  print_header
  echo ""
  echo -e "${BLUE}Usage:${NC}"
  echo "  ./scripts/start-with-tls.sh <command> [options]"
  echo ""
  echo -e "${BLUE}Commands:${NC}"
  echo "  generate          Generate self-signed certificates"
  echo "  start-server      Start Centro with TLS enabled"
  echo "  start-agent       Start Agent with TLS enabled [NODE_ID]"
  echo ""
  echo -e "${BLUE}Environment Variables:${NC}"
  echo "  CERTS_DIR         Directory for certificates (default: ./.certs)"
  echo "  DOMAIN            Domain name for certificates (default: localhost)"
  echo "  JWT_SECRET        JWT secret for tokens"
  echo "  ADMIN_PASSWORD    Admin password"
  echo ""
  echo -e "${BLUE}Examples:${NC}"
  echo "  # Generate certificates"
  echo "  ./scripts/start-with-tls.sh generate"
  echo ""
  echo "  # Start Centro with TLS"
  echo "  ./scripts/start-with-tls.sh start-server"
  echo ""
  echo "  # Start Agent with TLS (in another terminal)"
  echo "  ./scripts/start-with-tls.sh start-agent worker-1"
  echo ""
  echo "  # Custom domain and certs directory"
  echo "  DOMAIN=grpc.local CERTS_DIR=/etc/certs ./scripts/start-with-tls.sh start-server"
  echo ""
  echo -e "${BLUE}Quick Start (3 steps):${NC}"
  echo "  1. ./scripts/start-with-tls.sh generate"
  echo "  2. ./scripts/start-with-tls.sh start-server          # Terminal 1"
  echo "  3. ./scripts/start-with-tls.sh start-agent agent-1   # Terminal 2"
  echo ""
}

case "$MODE" in
  generate)
    generate_certs
    ;;
  start-server)
    start_server
    ;;
  start-agent)
    start_agent "$@"
    ;;
  help|"")
    print_help
    ;;
  *)
    echo -e "${RED}Unknown command: $MODE${NC}"
    print_help
    exit 1
    ;;
esac
