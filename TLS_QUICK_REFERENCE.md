# TLS Quick Reference Card

## 🚀 3-Minute TLS Setup

```bash
# Step 1: Generate certificates (first time only)
./scripts/generate-certs.sh

# Step 2: Start Centro with TLS (Terminal 1)
./scripts/start-with-tls.sh start-server

# Step 3: Start Agent with TLS (Terminal 2)
./scripts/start-with-tls.sh start-agent agent-1

# Done! gRPC is now encrypted 🔒
```

---

## 📋 Common Commands

### Generate Certificates

```bash
# Default (localhost, ./.certs)
./scripts/generate-certs.sh

# Custom domain and directory
./scripts/generate-certs.sh grpc.local ./certs
./scripts/generate-certs.sh staging.example.com /opt/certs
```

### Start Services

```bash
# Start Centro with TLS
./scripts/start-with-tls.sh start-server

# Start Agent (single node)
./scripts/start-with-tls.sh start-agent agent-1

# Start multiple agents
./scripts/start-with-tls.sh start-agent agent-2
./scripts/start-with-tls.sh start-agent agent-3
```

### Custom Configuration

```bash
# Change domain
DOMAIN=grpc.staging ./scripts/start-with-tls.sh start-server

# Change certificate directory
CERTS_DIR=/opt/certs ./scripts/start-with-tls.sh start-server

# Both
DOMAIN=example.com CERTS_DIR=/opt/certs ./scripts/start-with-tls.sh start-server
```

---

## 🔧 Manual Configuration

### Centro Environment Variables

```bash
export GRPC_SERVER_TLS_ENABLED=true
export GRPC_SERVER_TLS_CERT_FILE=./.certs/server-cert.pem
export GRPC_SERVER_TLS_KEY_FILE=./.certs/server-key.pem
export GRPC_SERVER_TLS_CA_FILE=./.certs/ca-cert.pem

./centro_server --etcd-endpoints localhost:2379
```

### Agent Environment Variables

```bash
export GRPC_TLS_ENABLED=true
export GRPC_TLS_CERT_FILE=./.certs/client-cert.pem
export GRPC_TLS_KEY_FILE=./.certs/client-key.pem
export GRPC_TLS_CA_FILE=./.certs/ca-cert.pem

./agent_client --server localhost:50051
```

---

## ✅ Verification

```bash
# Check certificate validity
openssl x509 -in ./.certs/server-cert.pem -text -noout

# Verify certificate chain
openssl verify -CAfile ./.certs/ca-cert.pem ./.certs/server-cert.pem

# Check expiry date
openssl x509 -in ./.certs/server-cert.pem -noout -dates

# List all certificate files
ls -lh ./.certs/
```

---

## 📂 Generated Files

```
.certs/
├── ca-cert.pem          # Root CA certificate
├── ca-key.pem           # Root CA private key
├── server-cert.pem      # Server certificate (Centro)
├── server-key.pem       # Server private key
├── client-cert.pem      # Client certificate (Agent)
└── client-key.pem       # Client private key
```

**File sizes:**
- CA key: ~1.7 KB
- CA cert: ~1.3 KB
- Server key: ~1.7 KB
- Server cert: ~1.4 KB
- Client key: ~1.7 KB
- Client cert: ~1.3 KB
- **Total: ~9.4 KB**

---

## 🔐 Security Checklist

- [ ] Certificates generated and distributed
- [ ] TLS enabled on Centro (GRPC_SERVER_TLS_ENABLED=true)
- [ ] TLS enabled on all Agents (GRPC_TLS_ENABLED=true)
- [ ] Agents can connect to Centro successfully
- [ ] Certificate expiry dates noted
- [ ] Renewal procedure documented
- [ ] Private keys backed up securely
- [ ] Private keys not in version control

---

## ⚠️ Common Issues

| Problem | Solution |
|---------|----------|
| Certificate not found | Run `./scripts/generate-certs.sh` |
| Permission denied | `chmod 644 .certs/*.pem` |
| Connection refused | Check if Centro is running on correct port |
| Certificate verification failed | Check CN/SAN matches domain; regenerate if needed |
| Expired certificate | `./scripts/generate-certs.sh` to regenerate |
| Domain mismatch error | Use correct domain in DOMAIN env var |

---

## 📖 Full Documentation

For detailed information, see:
- **[README/TLS_SETUP.md](README/TLS_SETUP.md)** - Complete setup guide
- **[README/GRPC_SETUP.md](README/GRPC_SETUP.md)** - gRPC configuration
- **[MVP_ENHANCEMENTS.md](MVP_ENHANCEMENTS.md)** - Security overview

---

## 🚀 Production Deployment

### With Let's Encrypt

```bash
# Install certbot
sudo apt-get install certbot

# Generate certificate
sudo certbot certonly --standalone -d grpc.example.com

# Configure Centro
export GRPC_SERVER_TLS_ENABLED=true
export GRPC_SERVER_TLS_CERT_FILE=/etc/letsencrypt/live/grpc.example.com/fullchain.pem
export GRPC_SERVER_TLS_KEY_FILE=/etc/letsencrypt/live/grpc.example.com/privkey.pem

./centro_server --etcd-endpoints etcd:2379
```

See [README/TLS_SETUP.md](README/TLS_SETUP.md) for full production setup.

---

## 💡 Pro Tips

**Tip 1: Check certificate details quickly**
```bash
openssl x509 -in ./.certs/server-cert.pem -noout -text | grep -E "CN=|DNS:|Not After"
```

**Tip 2: Test TLS connection with curl (if REST API)**
```bash
curl -k --cert ./.certs/client-cert.pem \
  --key ./.certs/client-key.pem \
  https://localhost:8080/api/jobs
```

**Tip 3: View all certificate info**
```bash
for cert in ./.certs/*.pem; do
  echo "=== $(basename $cert) ==="
  openssl x509 -in "$cert" -text -noout 2>/dev/null | head -15
done
```

**Tip 4: Monitor certificate expiry**
```bash
#!/bin/bash
for cert in ./.certs/*-cert.pem; do
  expiry=$(openssl x509 -in "$cert" -noout -enddate | cut -d= -f2)
  echo "$(basename $cert): Expires $expiry"
done
```

**Tip 5: Bulk copy certs to remote nodes**
```bash
scp -r ./.certs/* node1:/opt/certs/
scp -r ./.certs/* node2:/opt/certs/
scp -r ./.certs/* node3:/opt/certs/
```

---

Last updated: 2025-11-23
