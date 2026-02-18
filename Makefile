PROTO_DIR := proto
PROTO_SRC := $(wildcard $(PROTO_DIR)/*.proto)
GO_OUT := .

.PHONY: generate-proto
generate-proto:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go-grpc_out=$(GO_OUT) \
		$(PROTO_SRC)




# Run web docker only from root dir
# docker build -f infra/development/docker/web.Dockerfile -t test-web .

# initial environment setup commands
# 01.
# minikube start \
#   --driver=docker \
#   --docker-opt="dns=192.168.0.1" \
#   --docker-opt="dns=8.8.8.8" \
#   --docker-opt="dns=1.1.1.1" \
#   --cpus=2 \
#   --memory=3072mb \
#   --kubernetes-version=v1.35.0

# 02.
# tilt up