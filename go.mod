module github.com/opskumu/helm-wrapper

go 1.14

require (
	github.com/Masterminds/semver v1.5.0
	github.com/gin-gonic/gin v1.6.3
	github.com/gofrs/flock v0.8.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	helm.sh/helm/v3 v3.5.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)
