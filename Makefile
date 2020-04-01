BINARY_NAME=helm-wrapper

LDFLAGS="-s -w"

build:
	go build -ldflags ${LDFLAGS} -o ${BINARY_NAME} 

# cross compilation
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags ${LDFLAGS} -o ${BINARY_NAME}

# build docker image
build-docker:
	GOOS=linux GOARCH=amd64 go build -ldflags ${LDFLAGS} -o ${BINARY_NAME}
	docker build -t helm-wrapper:`git rev-parse --short HEAD` .
