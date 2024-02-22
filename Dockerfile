FROM golang:1.21.5

# Base libraries
RUN apt-get update
RUN apt-get install -y protobuf-compiler make 

# Go proto dependencies
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
RUN go install github.com/envoyproxy/protoc-gen-validate@v0.6.3

# Common base proto dependencies.
RUN git clone https://github.com/googleapis/googleapis

RUN mkdir -p /proto-common/validate
RUN cp pkg/mod/github.com/envoyproxy/protoc-gen-validate@v0.6.3/validate/validate.proto /proto-common/validate/validate.proto
RUN cp -r googleapis/google /proto-common

# Install node and related packages.
ENV NODE_VERSION=20.10.0
RUN apt install -y curl
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
ENV NVM_DIR=/root/.nvm
RUN . "$NVM_DIR/nvm.sh" && nvm install ${NODE_VERSION}
RUN . "$NVM_DIR/nvm.sh" && nvm use v${NODE_VERSION}
RUN . "$NVM_DIR/nvm.sh" && nvm alias default v${NODE_VERSION}
ENV PATH="/root/.nvm/versions/node/v${NODE_VERSION}/bin/:${PATH}"
RUN node --version
RUN npm --version

# Setup working directory for the software.
RUN mkdir -p /opt/app
WORKDIR /opt/app

# Copy over the node files.
COPY package.json .
COPY package-lock.json .

# Install the node module.
RUN npm install 

# Copy over the source files.
COPY src/ src/
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY web/ web/
COPY Makefile . 
COPY go.mod .
COPY go.sum . 

RUN go mod tidy

RUN make build-all

EXPOSE 8000

CMD ["go", "run", "cmd/driver/main.go"]
