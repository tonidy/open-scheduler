.PHONY: proto clean install-tools

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/agent.proto
	@echo "Protobuf generation complete!"

# Install required protobuf tools
install-tools:
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Tools installed!"

# Clean generated files
clean:
	rm -f proto/*.pb.go

# Run the agent
run-agent:
	cd agent && go run .

# Run the control plane
run-centro:
	cd centro && go run .

