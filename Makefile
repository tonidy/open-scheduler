.PHONY: proto clean install-tools swagger

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/agent.proto
	@echo "Protobuf generation complete!"

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	swag init -g centro/main.go -o docs
	@echo "Swagger documentation generated!"

# Install required protobuf tools
install-tools:
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Tools installed!"

# Clean generated files
clean:
	rm -f proto/*.pb.go

# Run the agent
run-agent:
	cd agent && go run .

# Run the control plane (with REST API)
run-centro:
	cd centro && go run . --port 50051 --http-port 8080

# Build centro binary
build-centro:
	cd centro && go build -o centro_server .

# Build agent binary
build-agent:
	cd agent && go build -o agent_binary .

# Build all
build-all: clean proto swagger build-centro build-agent

# Test REST API (requires centro to be running)
test-api:
	@chmod +x test_rest_api.sh 2>/dev/null || true
	@./test_rest_api.sh

