# Root Makefile for whatsapp-group4
.PHONY: gen-proto clean

# Directory containing your shared proto files
PROTO_DIR := proto
# Directory where you want the generated Go code
GEN_DIR := gen/go

# Find all .proto files
PROTO_FILES := $(shell find $(PROTO_DIR) -name "*.proto")

gen-proto:
	@echo "Generating Go code from protobufs..."
	@mkdir -p $(GEN_DIR)
	@for file in $(PROTO_FILES); do \
		protoc --proto_path=$(PROTO_DIR) \
		       --go_out=$(GEN_DIR) \
		       --go_opt=paths=source_relative \
		       --go-grpc_out=$(GEN_DIR) \
		       --go-grpc_opt=paths=source_relative \
		       $$file; \
	done
	@echo "Generation complete."

clean:
	@echo "Cleaning generated files..."
	@rm -rf $(GEN_DIR)