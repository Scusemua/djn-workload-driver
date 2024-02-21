STATIC_DIR ?= web

build-scss:
	npx sass -I . web/main.scss web/main.css

build: 
	GOARCH=wasm GOOS=js go build -o web/app.wasm .
	go build

depend:
	find node_modules/@patternfly/patternfly/ -name "*.css" -type f -delete
	rm -rf $(STATIC_DIR)/fonts
	mkdir -p $(STATIC_DIR)
	cp -r node_modules/@patternfly/patternfly/assets/fonts $(STATIC_DIR)

run: build
	./workload-driver.exe