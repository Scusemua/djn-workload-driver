STATIC_DIR ?= web

build-all: build-scss build-grpc build

build-scss:
	npx sass -I . web/main.scss web/main.css

build-grpc:
	@echo "Building gRPC now."
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative src/cluster/cluster.proto

build:  
	GOARCH=wasm GOOS=js go build -o web/app.wasm ./cmd/driver
	go build ./cmd/driver

depend:
	find node_modules/@patternfly/patternfly/ -name "*.css" -type f -delete
	rm -rf $(STATIC_DIR)/fonts
	mkdir -p $(STATIC_DIR)
	cp -r node_modules/@patternfly/patternfly/assets/fonts $(STATIC_DIR)

run: build
	go run cmd/driver/main.go