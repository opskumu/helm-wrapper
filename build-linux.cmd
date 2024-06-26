set GOOS=linux
set GOARCH=amd64
go build -o bin/helm-wrapper



CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags -s -w -o helm-wrapper



set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -ldflags -s -w -o helm-wrapper


docker build -t registry.inspures.com/csp/heimdall/helm-wrapper:20240626 .

# windows ---> arm64

set CGO_ENABLED=0
set GOOS=linux
set GOARCH=arm64
go build -ldflags -s -o /home/huary/code/helm-wrapper/bin/arm64/helm-wrapper

## linux---> arm64
GOOS=linux GOARCH=arm64 go build -ldflags -s -o /home/huary/code/helm-wrapper/bin/arm64/helm-wrapper

## arm64 docker
docker buildx version
docker buildx create --use
docker buildx build --platform=linux/arm64 -t registry.inspures.com/csp/heimdall/helm-wrapper:20240425-arm64 --push -f Dockerfile .