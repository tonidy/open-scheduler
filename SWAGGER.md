# Swagger API Documentation

This project now includes Swagger/OpenAPI documentation for the REST API.

## Accessing the Swagger UI

Once the centro server is running, you can access the Swagger UI at:

```
http://localhost:8080/swagger/index.html
```

## Regenerating Documentation

If you modify the API endpoints or add new handlers, you need to regenerate the Swagger documentation:

```bash
make swagger
```

Or manually:

```bash
swag init -g centro/main.go -o docs
```

## Using the Swagger UI

1. **Start the centro server**:
   ```bash
   make run-centro
   ```

2. **Open your browser** and navigate to:
   ```
   http://localhost:8080/swagger/index.html
   ```

3. **Authenticate**:
   - First, use the `/api/v1/auth/login` endpoint to get a JWT token
   - Default credentials: username: `admin`, password: `admin123`
   - Copy the token from the response
   - Click the "Authorize" button (🔒) at the top right
   - **IMPORTANT**: Enter `Bearer <your-token>` (replace `<your-token>` with the actual token)
     - Example: `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`
     - **Do NOT forget the "Bearer " prefix!**
   - Click "Authorize" and then "Close"

4. **Try the API**:
   - All endpoints are now authenticated and you can test them directly from the Swagger UI
   - Expand any endpoint, click "Try it out", fill in parameters, and click "Execute"

## API Endpoints

The following endpoints are documented:

### Authentication
- `POST /api/v1/auth/login` - Login to get JWT token

### Jobs
- `GET /api/v1/jobs` - List all jobs (with optional status filter)
- `POST /api/v1/jobs` - Submit a new job
- `GET /api/v1/jobs/{id}` - Get job details
- `GET /api/v1/jobs/{id}/status` - Get job status
- `GET /api/v1/jobs/{id}/events` - Get job events
- `GET /api/v1/jobs/{id}/events/stream` - Stream job events (SSE)

### Nodes
- `GET /api/v1/nodes` - List all nodes
- `GET /api/v1/nodes/{id}` - Get node details
- `GET /api/v1/nodes/{id}/health` - Check node health

### Statistics
- `GET /api/v1/stats` - Get system statistics

## Development

When adding new endpoints:

1. Add Swagger annotations to your handler function:
   ```go
   // handleMyEndpoint godoc
   // @Summary Short description
   // @Description Detailed description
   // @Tags TagName
   // @Accept json
   // @Produce json
   // @Security BearerAuth
   // @Param id path string true "Parameter description"
   // @Success 200 {object} MyResponseType
   // @Failure 400 {object} map[string]string
   // @Router /my-endpoint/{id} [get]
   func (s *APIServer) handleMyEndpoint(w http.ResponseWriter, r *http.Request) {
       // handler code
   }
   ```

2. Regenerate the documentation:
   ```bash
   make swagger
   ```

3. Rebuild and restart the centro server:
   ```bash
   make build-centro
   make run-centro
   ```

## Troubleshooting

### Issue: Getting 401 Unauthorized errors

**Problem**: The generated curl command doesn't include "Bearer" prefix in the Authorization header.

**Solution**: When you click "Authorize" in Swagger UI, you MUST include the "Bearer " prefix:
- ✅ Correct: `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`
- ❌ Wrong: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`

The authorization dialog will show:
```
Available authorizations
BearerAuth (apiKey)
Enter your JWT token in the format: Bearer {token}

Value: [________________]
```

Make sure to paste: `Bearer YOUR_ACTUAL_TOKEN_HERE`

### Issue: Token expired

Tokens expire after 24 hours. If you get authentication errors:
1. Call `/api/v1/auth/login` again to get a new token
2. Re-authorize in Swagger UI with the new token (remember the "Bearer " prefix!)

## Notes

- The Swagger documentation is automatically served from the `/swagger/` path
- The documentation files are in the `docs/` directory
- Authentication is required for most endpoints except `/auth/login`
- All authenticated endpoints require a Bearer token in the Authorization header
- **Always include "Bearer " prefix when authorizing in Swagger UI**

