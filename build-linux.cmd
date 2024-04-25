set GOOS=linux
set GOARCH=amd64
go build -o bin/helm-wrapper



CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags -s -w -o helm-wrapper



set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -ldflags -s -w -o helm-wrapper


docker build -t registry.inspures.com/csp/heimdall/helm-wrapper:20240425 .