# TLS/mTLS Setup Guide for Open Scheduler

This guide explains how to set up TLS (Transport Layer Security) certificates for secure gRPC communication between Centro and Agent nodes.

## Quick Start (Development)

```bash
# 1. Generate self-signed certificates
./scripts/generate-certs.sh

# 2. Start Centro with TLS (Terminal 1)
./scripts/start-with-tls.sh start-server

# 3. Start Agent with TLS (Terminal 2)
./scripts/start-with-tls.sh start-agent agent-1

# 4. Start another Agent (Terminal 3)
./scripts/start-with-tls.sh start-agent agent-2
```

Done! Your gRPC communication is now encrypted.

---

## Certificate Types

### Development Mode (Self-Signed)

Use self-signed certificates for local development and testing.

**Advantages:**
- ✅ No DNS or domain required
- ✅ Instant generation
- ✅ Works on localhost
- ✅ Perfect for dev/testing

**Disadvantages:**
- ❌ Not trusted by browsers/clients
- ❌ Gets warnings in TLS validation
- ❌ Only valid for 365 days (configurable)

**Generate:**
```bash
./scripts/generate-certs.sh localhost ./.certs
```

### Production Mode (Let's Encrypt)

Use Let's Encrypt certificates for production deployments.

**Advantages:**
- ✅ Trusted Certificate Authority
- ✅ Free
- ✅ Auto-renewal capability
- ✅ Industry standard

**Disadvantages:**
- ❌ Requires domain name and DNS
- ❌ Requires port 80/443 access
- ❌ Manual renewal setup

**Generate:**
```bash
# Install certbot
sudo apt-get install certbot

# Generate certificate
sudo certbot certonly --standalone -d grpc.example.com

# Certificates in:
# /etc/letsencrypt/live/grpc.example.com/fullchain.pem
# /etc/letsencrypt/live/grpc.example.com/privkey.pem
```

---

## Detailed Setup Instructions

### Step 1: Generate Certificates

#### Option A: Self-Signed (Development)

```bash
./scripts/generate-certs.sh [domain] [output_dir]

# Examples:
./scripts/generate-certs.sh                          # Uses localhost, ./.certs
./scripts/generate-certs.sh grpc.local ./certs       # Custom domain and dir
./scripts/generate-certs.sh example.com /opt/certs   # Production-like setup
```

**Output files:**
```
.certs/
├── ca-cert.pem           # CA certificate
├── ca-key.pem            # CA private key
├── server-cert.pem       # Server certificate (for Centro)
├── server-key.pem        # Server private key
├── client-cert.pem       # Client certificate (for Agent)
└── client-key.pem        # Client private key
```

#### Option B: Let's Encrypt (Production)

```bash
# Install certbot
sudo apt-get install certbot

# Generate
sudo certbot certonly --standalone -d grpc.example.com

# Use these files:
# Cert: /etc/letsencrypt/live/grpc.example.com/fullchain.pem
# Key:  /etc/letsencrypt/live/grpc.example.com/privkey.pem
```

---

### Step 2: Configure Centro (Server)

Export TLS environment variables before starting Centro:

```bash
# Enable TLS
export GRPC_SERVER_TLS_ENABLED=true

# Path to certificate files
export GRPC_SERVER_TLS_CERT_FILE=./.certs/server-cert.pem
export GRPC_SERVER_TLS_KEY_FILE=./.certs/server-key.pem

# CA certificate (for mTLS validation of agents)
export GRPC_SERVER_TLS_CA_FILE=./.certs/ca-cert.pem

# Start Centro
./centro_server --etcd-endpoints localhost:2379
```

**Or use the quick-start script:**
```bash
./scripts/start-with-tls.sh start-server
```

**Expected output:**
```
[Centro] TLS enabled for gRPC server
[Centro] Starting gRPC server on :50051
[Centro] REST API server is ready and listening on :8080
```

---

### Step 3: Configure Agent (Client)

Export TLS environment variables before starting Agent:

```bash
# Enable TLS
export GRPC_TLS_ENABLED=true

# Path to certificate files (client certs)
export GRPC_TLS_CERT_FILE=./.certs/client-cert.pem
export GRPC_TLS_KEY_FILE=./.certs/client-key.pem

# CA certificate (for validating server cert)
export GRPC_TLS_CA_FILE=./.certs/ca-cert.pem

# Start Agent
./agent_client --server localhost:50051 --token test-token
```

**Or use the quick-start script:**
```bash
./scripts/start-with-tls.sh start-agent agent-1
```

**Expected output:**
```
[GrpcClient] Using TLS for secure communication
[GrpcClient] Successfully connected to server
[Agent] Heartbeat sent successfully
```

---

## Verification

### Check Certificate Details

```bash
# View server certificate
openssl x509 -in ./.certs/server-cert.pem -text -noout

# View client certificate
openssl x509 -in ./.certs/client-cert.pem -text -noout

# View CA certificate
openssl x509 -in ./.certs/ca-cert.pem -text -noout
```

### Verify Certificate Chain

```bash
# Verify server cert signed by CA
openssl verify -CAfile ./.certs/ca-cert.pem ./.certs/server-cert.pem

# Verify client cert signed by CA
openssl verify -CAfile ./.certs/ca-cert.pem ./.certs/client-cert.pem
```

### Test gRPC Connection

```bash
# With self-signed cert (insecure skip verify)
grpcurl -insecure grpc.localhost:50051 list

# With proper CA chain
grpcurl -cacert ./.certs/ca-cert.pem \
  -cert ./.certs/client-cert.pem \
  -key ./.certs/client-key.pem \
  grpc.localhost:50051 list
```

---

## Environment Variable Reference

### Centro (Server)

| Variable | Required | Example | Notes |
|----------|----------|---------|-------|
| `GRPC_SERVER_TLS_ENABLED` | Yes | `true` / `false` | Enable/disable TLS |
| `GRPC_SERVER_TLS_CERT_FILE` | If enabled | `./.certs/server-cert.pem` | Server certificate |
| `GRPC_SERVER_TLS_KEY_FILE` | If enabled | `./.certs/server-key.pem` | Server private key |
| `GRPC_SERVER_TLS_CA_FILE` | Optional | `./.certs/ca-cert.pem` | For mTLS validation |

### Agent (Client)

| Variable | Required | Example | Notes |
|----------|----------|---------|-------|
| `GRPC_TLS_ENABLED` | Yes | `true` / `false` | Enable/disable TLS |
| `GRPC_TLS_CERT_FILE` | If enabled | `./.certs/client-cert.pem` | Client certificate |
| `GRPC_TLS_KEY_FILE` | If enabled | `./.certs/client-key.pem` | Client private key |
| `GRPC_TLS_CA_FILE` | If enabled | `./.certs/ca-cert.pem` | CA certificate |
| `GRPC_TLS_INSECURE_SKIP_VERIFY` | Optional | `false` | Skip cert verification (insecure!) |

---

## Common Scenarios

### Scenario 1: Development on Localhost

```bash
# Generate self-signed certs
./scripts/generate-certs.sh localhost ./.certs

# Terminal 1: Start Centro
export GRPC_SERVER_TLS_ENABLED=true
export GRPC_SERVER_TLS_CERT_FILE=./.certs/server-cert.pem
export GRPC_SERVER_TLS_KEY_FILE=./.certs/server-key.pem
export GRPC_SERVER_TLS_CA_FILE=./.certs/ca-cert.pem
./centro_server

# Terminal 2: Start Agent
export GRPC_TLS_ENABLED=true
export GRPC_TLS_CERT_FILE=./.certs/client-cert.pem
export GRPC_TLS_KEY_FILE=./.certs/client-key.pem
export GRPC_TLS_CA_FILE=./.certs/ca-cert.pem
./agent_client --server localhost:50051
```

### Scenario 2: Multi-Node Cluster (Staging)

```bash
# Create shared certs directory
mkdir -p /opt/open-scheduler/certs

# Generate certs with proper domain
./scripts/generate-certs.sh grpc.staging /opt/open-scheduler/certs

# Copy certs to all nodes
scp -r /opt/open-scheduler/certs/* node1:/opt/open-scheduler/certs/
scp -r /opt/open-scheduler/certs/* node2:/opt/open-scheduler/certs/

# On Centro node
export GRPC_SERVER_TLS_ENABLED=true
export GRPC_SERVER_TLS_CERT_FILE=/opt/open-scheduler/certs/server-cert.pem
export GRPC_SERVER_TLS_KEY_FILE=/opt/open-scheduler/certs/server-key.pem
export GRPC_SERVER_TLS_CA_FILE=/opt/open-scheduler/certs/ca-cert.pem
./centro_server --etcd-endpoints etcd.staging:2379

# On each Agent node
export GRPC_TLS_ENABLED=true
export GRPC_TLS_CERT_FILE=/opt/open-scheduler/certs/client-cert.pem
export GRPC_TLS_KEY_FILE=/opt/open-scheduler/certs/client-key.pem
export GRPC_TLS_CA_FILE=/opt/open-scheduler/certs/ca-cert.pem
./agent_client --server grpc.staging:50051 --node-id agent-1
```

### Scenario 3: Production with Let's Encrypt

```bash
# 1. Setup domain DNS (grpc.example.com -> server IP)
# 2. Generate certificate
sudo certbot certonly --standalone -d grpc.example.com

# 3. Setup auto-renewal hook
sudo nano /etc/letsencrypt/renewal-hooks/post/restart-centro.sh

#!/bin/bash
systemctl restart open-scheduler-centro
chmod +x /etc/letsencrypt/renewal-hooks/post/restart-centro.sh

# 4. Configure Centro
export GRPC_SERVER_TLS_ENABLED=true
export GRPC_SERVER_TLS_CERT_FILE=/etc/letsencrypt/live/grpc.example.com/fullchain.pem
export GRPC_SERVER_TLS_KEY_FILE=/etc/letsencrypt/live/grpc.example.com/privkey.pem
./centro_server --etcd-endpoints etcd.prod:2379

# 5. Configure Agents
export GRPC_TLS_ENABLED=true
export GRPC_TLS_CA_FILE=/etc/letsencrypt/live/grpc.example.com/chain.pem
./agent_client --server grpc.example.com:50051
```

---

## Troubleshooting

### Certificate Not Found Error

```
Error: failed to create TLS credentials: open ./.certs/server-cert.pem: no such file or directory
```

**Solution:**
```bash
# Generate certificates
./scripts/generate-certs.sh

# Or specify correct path
export GRPC_SERVER_TLS_CERT_FILE=/path/to/cert.pem
```

### Certificate Verification Failed

```
Error: certificate verification failed
```

**Solution:**
```bash
# Verify certificate chain
openssl verify -CAfile ./.certs/ca-cert.pem ./.certs/server-cert.pem

# Check certificate expiry
openssl x509 -in ./.certs/server-cert.pem -noout -dates

# If expired, regenerate
./scripts/generate-certs.sh
```

### Certificate Doesn't Have Required Domain

```
Error: certificate is not valid for any of the requested names
```

**Solution:**
```bash
# Regenerate with correct domain
./scripts/generate-certs.sh grpc.yourdomain.com ./.certs

# Or use INSECURE_SKIP_VERIFY (dev only)
export GRPC_TLS_INSECURE_SKIP_VERIFY=true
```

### Permission Denied on Certificate Files

```
Error: permission denied (os error 13)
```

**Solution:**
```bash
# Make certificates readable
chmod 644 ./.certs/*.pem
chmod 755 ./.certs

# For production, set proper owner
sudo chown centro_user:centro_user /opt/certs/*.pem
```

---

## Security Best Practices

### ✅ DO

- ✅ Use Let's Encrypt for production
- ✅ Enable auto-renewal for Let's Encrypt certs
- ✅ Use mTLS (mutual TLS) with CA validation
- ✅ Rotate certificates regularly
- ✅ Use strong private key sizes (2048+ bits)
- ✅ Restrict file permissions on private keys (600)
- ✅ Monitor certificate expiry dates
- ✅ Keep private keys secure and backed up

### ❌ DON'T

- ❌ Use self-signed certificates in production
- ❌ Skip certificate verification (`INSECURE_SKIP_VERIFY`)
- ❌ Commit private keys to version control
- ❌ Share certificates across multiple services
- ❌ Use expired certificates
- ❌ Make private keys world-readable
- ❌ Use weak key sizes (< 2048 bits)
- ❌ Trust self-signed certs without validation

---

## Certificate Renewal

### Self-Signed Certificates

Self-signed certificates are valid for 365 days. Regenerate before expiry:

```bash
# Check expiry date
openssl x509 -in ./.certs/server-cert.pem -noout -dates

# Regenerate if expiring soon
./scripts/generate-certs.sh localhost ./.certs
```

### Let's Encrypt Certificates

Auto-renewal is handled by certbot:

```bash
# Test renewal
sudo certbot renew --dry-run

# Manual renewal
sudo certbot renew

# Check renewal status
sudo systemctl status certbot.timer
```

---

## Files Reference

| File | Purpose |
|------|---------|
| `scripts/generate-certs.sh` | Generate self-signed certificates |
| `scripts/start-with-tls.sh` | Quick start Centro/Agent with TLS |
| `centro/grpc/tls.go` | Server TLS utilities |
| `agent/grpc/tls.go` | Client TLS utilities |

---

## Related Documentation

- [MVP_ENHANCEMENTS.md](./MVP_ENHANCEMENTS.md) - Security hardening overview
- [QUICKSTART.md](./QUICKSTART.md) - Getting started guide
- [GRPC_SETUP.md](./GRPC_SETUP.md) - gRPC configuration details
