#!/bin/bash
# Generate self-signed TLS certificates for Open Scheduler
# Usage: ./scripts/generate-certs.sh [domain] [output_dir]
# Example: ./scripts/generate-certs.sh grpc.local ./certs

set -e

# Configuration
DOMAIN="${1:-localhost}"
OUTPUT_DIR="${2:-./.certs}"
DAYS_VALID=365
KEY_SIZE=2048

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Open Scheduler TLS Certificate Gen    ║${NC}"
echo -e "${BLUE}║  (Self-signed for Dev/Testing)         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"
echo -e "${GREEN}✓ Output directory: $OUTPUT_DIR${NC}"

# File paths
SERVER_KEY="$OUTPUT_DIR/server-key.pem"
SERVER_CERT="$OUTPUT_DIR/server-cert.pem"
CLIENT_KEY="$OUTPUT_DIR/client-key.pem"
CLIENT_CERT="$OUTPUT_DIR/client-cert.pem"
CA_KEY="$OUTPUT_DIR/ca-key.pem"
CA_CERT="$OUTPUT_DIR/ca-cert.pem"
CLIENT_CSR="$OUTPUT_DIR/client.csr"
SERVER_CSR="$OUTPUT_DIR/server.csr"
CA_CONFIG="$OUTPUT_DIR/ca.conf"

echo -e "${BLUE}Domain: $DOMAIN${NC}"
echo -e "${BLUE}Validity: $DAYS_VALID days${NC}"
echo ""

# Step 1: Generate CA private key
echo -e "${YELLOW}[1/6]${NC} Generating CA private key..."
openssl genrsa -out "$CA_KEY" $KEY_SIZE 2>/dev/null
echo -e "${GREEN}✓ CA private key created${NC}"

# Step 2: Generate CA certificate
echo -e "${YELLOW}[2/6]${NC} Generating CA certificate..."
openssl req -new -x509 -days $DAYS_VALID -key "$CA_KEY" -out "$CA_CERT" \
  -subj "/C=ID/ST=State/L=City/O=Open Scheduler/CN=Open Scheduler CA" \
  2>/dev/null
echo -e "${GREEN}✓ CA certificate created${NC}"

# Step 3: Generate Server private key
echo -e "${YELLOW}[3/6]${NC} Generating server private key..."
openssl genrsa -out "$SERVER_KEY" $KEY_SIZE 2>/dev/null
echo -e "${GREEN}✓ Server private key created${NC}"

# Step 4: Generate Server CSR and sign with CA
echo -e "${YELLOW}[4/6]${NC} Generating server certificate..."

# Create server config for SANs
cat > "$CA_CONFIG" <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = ID
ST = State
L = City
O = Open Scheduler
CN = $DOMAIN

[v3_req]
subjectAltName = DNS:$DOMAIN,DNS:localhost,DNS:*.localhost,IP:127.0.0.1
EOF

openssl req -new -key "$SERVER_KEY" -out "$SERVER_CSR" \
  -config "$CA_CONFIG" 2>/dev/null

openssl x509 -req -in "$SERVER_CSR" -CA "$CA_CERT" -CAkey "$CA_KEY" \
  -CAcreateserial -out "$SERVER_CERT" -days $DAYS_VALID \
  -extensions v3_req -extfile "$CA_CONFIG" 2>/dev/null

rm -f "$SERVER_CSR"
echo -e "${GREEN}✓ Server certificate created and signed${NC}"

# Step 5: Generate Client private key
echo -e "${YELLOW}[5/6]${NC} Generating client private key..."
openssl genrsa -out "$CLIENT_KEY" $KEY_SIZE 2>/dev/null
echo -e "${GREEN}✓ Client private key created${NC}"

# Step 6: Generate Client CSR and sign with CA
echo -e "${YELLOW}[6/6]${NC} Generating client certificate..."

cat > "$CA_CONFIG" <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = ID
ST = State
L = City
O = Open Scheduler
CN = agent

[v3_req]
subjectAltName = DNS:agent,DNS:localhost
EOF

openssl req -new -key "$CLIENT_KEY" -out "$CLIENT_CSR" \
  -config "$CA_CONFIG" 2>/dev/null

openssl x509 -req -in "$CLIENT_CSR" -CA "$CA_CERT" -CAkey "$CA_KEY" \
  -CAcreateserial -out "$CLIENT_CERT" -days $DAYS_VALID \
  -extensions v3_req -extfile "$CA_CONFIG" 2>/dev/null

rm -f "$CLIENT_CSR" "$CA_CONFIG"
echo -e "${GREEN}✓ Client certificate created and signed${NC}"

echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}✓ All certificates generated successfully!${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""

# Display certificate details
echo -e "${BLUE}Certificate Details:${NC}"
echo ""

echo -e "${YELLOW}CA Certificate:${NC}"
openssl x509 -in "$CA_CERT" -text -noout 2>/dev/null | grep -E "Subject:|Issuer:|Not Before|Not After|Public-Key" | sed 's/^/  /'

echo ""
echo -e "${YELLOW}Server Certificate:${NC}"
openssl x509 -in "$SERVER_CERT" -text -noout 2>/dev/null | grep -E "Subject:|CN =|DNS:|Issuer:|Not Before|Not After" | sed 's/^/  /'

echo ""
echo -e "${YELLOW}Client Certificate:${NC}"
openssl x509 -in "$CLIENT_CERT" -text -noout 2>/dev/null | grep -E "Subject:|CN =|DNS:|Issuer:|Not Before|Not After" | sed 's/^/  /'

echo ""
echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${BLUE}📁 Certificate Files:${NC}"
echo ""
ls -lh "$OUTPUT_DIR"/*.pem | awk '{print "  " $9 " (" $5 ")"}'
echo ""

echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${BLUE}🚀 Usage Instructions:${NC}"
echo ""
echo -e "${YELLOW}For Centro (Server):${NC}"
echo "  export GRPC_SERVER_TLS_ENABLED=true"
echo "  export GRPC_SERVER_TLS_CERT_FILE=$OUTPUT_DIR/server-cert.pem"
echo "  export GRPC_SERVER_TLS_KEY_FILE=$OUTPUT_DIR/server-key.pem"
echo "  export GRPC_SERVER_TLS_CA_FILE=$OUTPUT_DIR/ca-cert.pem"
echo "  ./centro_server"
echo ""

echo -e "${YELLOW}For Agent (Client):${NC}"
echo "  export GRPC_TLS_ENABLED=true"
echo "  export GRPC_TLS_CERT_FILE=$OUTPUT_DIR/client-cert.pem"
echo "  export GRPC_TLS_KEY_FILE=$OUTPUT_DIR/client-key.pem"
echo "  export GRPC_TLS_CA_FILE=$OUTPUT_DIR/ca-cert.pem"
echo "  ./agent_client --server $DOMAIN:50051"
echo ""

echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${YELLOW}⚠️  Warning:${NC}"
echo "  - These certificates are SELF-SIGNED and NOT trusted by browsers"
echo "  - Use ONLY for development and testing"
echo "  - For production, use Let's Encrypt or other CA"
echo "  - Certificates expire in $DAYS_VALID days"
echo ""

# Verify certificates work together
echo -e "${BLUE}Verifying certificate chain...${NC}"
if openssl verify -CAfile "$CA_CERT" "$SERVER_CERT" >/dev/null 2>&1; then
  echo -e "${GREEN}✓ Server certificate chain is valid${NC}"
else
  echo -e "${YELLOW}⚠ Warning: Could not verify server certificate chain${NC}"
fi

if openssl verify -CAfile "$CA_CERT" "$CLIENT_CERT" >/dev/null 2>&1; then
  echo -e "${GREEN}✓ Client certificate chain is valid${NC}"
else
  echo -e "${YELLOW}⚠ Warning: Could not verify client certificate chain${NC}"
fi

echo ""
echo -e "${GREEN}Done! 🎉${NC}"
