# Centro REST API Documentation

Centro provides a comprehensive REST API with JWT authentication for managing jobs, nodes, and monitoring the scheduler.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

All endpoints (except `/auth/login`) require JWT authentication. Include the JWT token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

### Default Credentials

- Username: `admin`
- Password: `admin123`

**Note:** Change these credentials in production by modifying `centro/rest/middleware.go`

---

## API Endpoints

### Authentication

#### POST /api/v1/auth/login

Login to get a JWT token.

**Request Body:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "username": "admin"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid request body or missing fields
- `401 Unauthorized` - Invalid credentials

---

### Jobs Management

#### GET /api/v1/jobs

List all jobs with optional status filtering.

**Query Parameters:**
- `status` (optional): Filter by status (`queued`, `active`, `completed`)

**Response (200 OK):**
```json
{
  "queued_count": 5,
  "active_jobs": [
    {
      "job_id": "abc-123",
      "node_id": "node-1",
      "status": "running",
      "detail": "Container started",
      "updated_at": "2025-11-09T10:30:00Z",
      "claimed_at": "2025-11-09T10:28:00Z",
      "job": {
        "job_id": "abc-123",
        "name": "example-job",
        "type": "service",
        "datacenters": "dc1",
        "tasks": [...]
      }
    }
  ],
  "completed_count": 42,
  "completed_jobs": [...]
}
```

#### POST /api/v1/jobs

Submit a new job to the scheduler.

**Request Body:**
```json
{
  "name": "my-job",
  "type": "service",
  "datacenters": "dc1,dc2",
  "tasks": [
    {
      "name": "web-server",
      "driver": "podman",
      "config": {
        "image": "nginx:latest",
        "options": {
          "port": "80:80"
        }
      },
      "env": {
        "NODE_ENV": "production"
      }
    }
  ],
  "meta": {
    "owner": "team-a",
    "priority": "high"
  }
}
```

**Response (201 Created):**
```json
{
  "job_id": "abc-123",
  "message": "Job submitted successfully",
  "job": {
    "job_id": "abc-123",
    "name": "my-job",
    ...
  }
}
```

**Error Responses:**
- `400 Bad Request` - Invalid request body or validation error
- `500 Internal Server Error` - Failed to submit job

#### GET /api/v1/jobs/:id

Get details of a specific job.

**Path Parameters:**
- `id`: Job ID

**Response (200 OK):**
```json
{
  "job_id": "abc-123",
  "status": "active",
  "node_id": "node-1",
  "detail": "Container running",
  "updated_at": "2025-11-09T10:30:00Z",
  "claimed_at": "2025-11-09T10:28:00Z",
  "job": {...}
}
```

**Error Responses:**
- `404 Not Found` - Job not found

#### GET /api/v1/jobs/:id/status

Get the status of a specific job.

**Path Parameters:**
- `id`: Job ID

**Response (200 OK):**
```json
{
  "job_id": "abc-123",
  "status": "running",
  "node_id": "node-1",
  "detail": "Container running",
  "updated_at": "2025-11-09T10:30:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Job not found

#### GET /api/v1/jobs/:id/events

Get events for a specific job.

**Path Parameters:**
- `id`: Job ID

**Response (200 OK):**
```json
{
  "job_id": "abc-123",
  "events": [
    "[2025-11-09T10:28:00Z] Job assigned to node node-1",
    "[2025-11-09T10:28:15Z] Status: claimed - Node claimed the job",
    "[2025-11-09T10:28:30Z] Status: running - Container started"
  ]
}
```

#### GET /api/v1/jobs/:id/events/stream

Stream events for a specific job in real-time using Server-Sent Events (SSE).

**Path Parameters:**
- `id`: Job ID

**Response (200 OK):**
```
Content-Type: text/event-stream

data: [2025-11-09T10:28:00Z] Job assigned to node node-1

data: [2025-11-09T10:28:15Z] Status: claimed - Node claimed the job

data: [2025-11-09T10:28:30Z] Status: running - Container started
```

---

### Node Management

#### GET /api/v1/nodes

List all registered nodes.

**Response (200 OK):**
```json
{
  "nodes": [
    {
      "node_id": "node-1",
      "last_heartbeat": "2025-11-09T10:29:45Z",
      "ram_mb": 8192.5,
      "cpu_percent": 45.2,
      "disk_mb": 102400.0,
      "metadata": {
        "datacenter": "dc1",
        "rack": "rack-1"
      }
    }
  ],
  "count": 1
}
```

#### GET /api/v1/nodes/:id

Get details of a specific node.

**Path Parameters:**
- `id`: Node ID

**Response (200 OK):**
```json
{
  "node_id": "node-1",
  "last_heartbeat": "2025-11-09T10:29:45Z",
  "ram_mb": 8192.5,
  "cpu_percent": 45.2,
  "disk_mb": 102400.0,
  "metadata": {
    "datacenter": "dc1"
  }
}
```

**Error Responses:**
- `404 Not Found` - Node not found

#### GET /api/v1/nodes/:id/health

Check the health status of a specific node.

**Path Parameters:**
- `id`: Node ID

**Response (200 OK):**
```json
{
  "node_id": "node-1",
  "healthy": true,
  "status": "healthy",
  "last_heartbeat": "2025-11-09T10:29:45Z"
}
```

A node is considered healthy if it has sent a heartbeat within the last 60 seconds.

**Error Responses:**
- `404 Not Found` - Node not found

---

### System Statistics

#### GET /api/v1/stats

Get overall system statistics.

**Response (200 OK):**
```json
{
  "nodes": {
    "total": 5,
    "healthy": 4
  },
  "jobs": {
    "queued": 12,
    "active": 8,
    "completed": 156
  }
}
```

---

## Usage Examples

### Using cURL

**1. Login and get token:**
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')
```

**2. List all jobs:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/jobs
```

**3. Submit a new job:**
```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx-service",
    "type": "service",
    "datacenters": "dc1",
    "tasks": [{
      "name": "web",
      "driver": "podman",
      "config": {
        "image": "nginx:latest"
      }
    }]
  }'
```

**4. Get job status:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/jobs/abc-123/status
```

**5. Stream job events:**
```bash
curl -N -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/jobs/abc-123/events/stream
```

**6. List all nodes:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/nodes
```

**7. Check node health:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/nodes/node-1/health
```

**8. Get system stats:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/stats
```

### Using JavaScript (fetch)

```javascript
// Login
const loginResponse = await fetch('http://localhost:8080/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: 'admin123' })
});
const { token } = await loginResponse.json();

// List jobs
const jobsResponse = await fetch('http://localhost:8080/api/v1/jobs', {
  headers: { 'Authorization': `Bearer ${token}` }
});
const jobs = await jobsResponse.json();

// Submit job
const newJobResponse = await fetch('http://localhost:8080/api/v1/jobs', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    name: 'my-job',
    type: 'service',
    datacenters: 'dc1',
    tasks: [...]
  })
});

// Stream events (EventSource)
const eventSource = new EventSource(
  `http://localhost:8080/api/v1/jobs/${jobId}/events/stream`,
  { headers: { 'Authorization': `Bearer ${token}` } }
);
eventSource.onmessage = (event) => {
  console.log('Event:', event.data);
};
```

---

## Running the Server

Start the Centro server with both gRPC and REST API:

```bash
cd centro && go run . --port 50051 --http-port 8080 --etcd-endpoints localhost:2379
```

Or using Make:

```bash
make run-centro
```

**Server Output:**
```
[Centro] Connecting to etcd endpoints: [localhost:2379]
[Centro] Successfully connected to etcd
[Centro] Starting gRPC server on :50051
[Centro] Starting REST API server on :8080
[Centro] gRPC server is ready and listening on :50051
[Centro] REST API server is ready and listening on :8080
[Centro] Press Ctrl+C to stop
```

---

## Security Notes

### JWT Secret

The JWT secret is currently hardcoded in `centro/rest/middleware.go`:

```go
var jwtSecret = []byte("your-secret-key-change-in-production")
```

**For production:**
1. Change this to a strong, randomly generated secret
2. Store it in an environment variable
3. Use a proper secrets management system

### Authentication

The current implementation uses a simple username/password check for demonstration. For production:

1. Implement proper user management with hashed passwords
2. Store credentials in a database
3. Add role-based access control (RBAC)
4. Consider OAuth2/OIDC integration
5. Implement rate limiting and brute-force protection

### CORS

CORS is currently configured to allow all origins (`*`). For production:

1. Restrict to specific trusted origins
2. Configure appropriate CORS policies based on your deployment

---

## Error Handling

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

Common HTTP status codes:
- `200 OK` - Successful request
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Missing or invalid authentication
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

---

## Future Enhancements

Planned features for the REST API:

1. **Job cancellation**: `DELETE /api/v1/jobs/:id`
2. **Job retry**: `POST /api/v1/jobs/:id/retry`
3. **Node filtering**: Query parameters for node listing
4. **Pagination**: Support for large result sets
5. **WebSocket support**: Alternative to SSE for log streaming
6. **Metrics endpoint**: Prometheus-compatible metrics
7. **API versioning**: Support for multiple API versions
8. **Rate limiting**: Protect against abuse
9. **Audit logging**: Track all API operations
10. **API key authentication**: Alternative to JWT for service-to-service

---

## Architecture

The REST API is implemented using:

- **HTTP Router**: gorilla/mux (already in dependencies)
- **JWT**: golang-jwt/jwt/v5 for token generation and validation
- **Storage**: Shared etcd storage with gRPC server
- **Middleware**: JWT authentication, CORS, logging
- **Streaming**: Server-Sent Events (SSE) for real-time log streaming

The REST API server runs alongside the gRPC server, sharing the same storage backend for consistency.
