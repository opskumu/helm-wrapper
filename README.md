# A [Helm3](https://github.com/helm/helm) HTTP Wrapper With [Go SDK](https://helm.sh/docs/topics/advanced/#go-sdk)

+ [中文文档](README_CN.md)

helm-wrapper is a helm3 HTTP wrapper with [helm Go SDK](https://helm.sh/docs/topics/advanced/#go-sdk). With helm-wrapper, you can use HTTP RESTFul API do something like helm commondline (install/uninstall/upgrade/get/list/rollback...).

## Support API

+ helm install
    - `POST`
    - `/api/namespaces/:namespace/releases/:release?chart=<chartName>`

POST Body: 

``` json
{
    "dry_run": false,           // `--dry-run`
    "disable_hooks": false,     // `--no-hooks`
    "wait": false,              // `--wait`
    "devel": false,             // `--false`
    "description": "",          // `--description`
    "atomic": false,            // `--atomic`
    "skip_crds": false,         // `--skip-crds`
    "sub_notes": false,         // `--render-subchart-notes`
    "create_namespace": false,  // `--create-namespace`
    "dependency_update": false, // `--dependency-update`
    "values": "",               // `--values`
    "set": [],                  // `--set`
    "set_string": []            // `--set-string`
}
```

> `"values"` -> helm install `--values` option 

+ helm uninstall
    - `DELETE`
    - `/api/namespaces/:namespace/releases/:release`
+ helm upgrade
    - `PUT`
    - `/api/namespaces/:namespace/releases/:release?chart=<chartName>`

PUT Body: 

``` json
{
    "dry_run": false,           // `--dry-run`
    "disable_hooks": false,     // `--no-hooks`
    "wait": false,              // `--wait`
    "devel": false,             // `--false`
    "description": "",          // `--description`
    "atomic": false,            // `--atomic`
    "skip_crds": false,         // `--skip-crds`
    "sub_notes": false,         // `--render-subchart-notes`
    "force": false,             // `--force`
    "install": false,           // `--install`
    "recreate": false,          // `--recreate`
    "cleanup_on_fail": false,   // `--cleanup-on-fail`
    "values": "",               // `--values`
    "set": [],                  // `--set`
    "set_string": []            // `--set-string`
}
```

> `"values"` -> helm install `--values` option 

+ helm rollback
    - `PUT`
    - `/api/namespaces/:namespace/releases/:release/versions/:reversion`
+ helm list
    - `GET`
    - `/api/namespaces/:namespace/releases`
+ helm get
    - `GET`
    - `/api/namespaces/:namespace/releases/:release`

| Params | Name |
| :- | :- |
| info | support all/hooks/manifest/notes/values | 

+ helm release history
    - `GET`
    - `/api/namespaces/:namespace/releases/:release/histories`

+ helm show
    - `GET`
    - `/api/charts`

| Params | Name |
| :- | :- |
| chart  | chart name, required|
| info   | support readme/values/chart |
| version | --version |

+ helm search repo
    - `GET`
    - `/api/repositories/charts`

| Params | Name |
| :- | :- |
| keyword | search keyword，required |
| version | chart version |
| versions | if "true", all versions |

+ helm repo update
    - `PUT`
    - `/api/repositories`

+ helm env
    - `GET`
    - `/api/envs`

> __Notes:__ helm-wrapper is Alpha status, no more test

### Response 


``` go
type respBody struct {
    Code  int         `json:"code"` // 0 or 1, 0 is ok, 1 is error
    Data  interface{} `json:"data,omitempty"`
    Error string      `json:"error,omitempty"`
}
```


## Build & Run 

### Build

```
make build
make build-linux    // build helm-wrapper Linux binary
make build-docker   // build docker image with helm-wrapper
```

#### helm-wrapper help

```
$ helm-wrapper -h
Usage of helm-wrapper:
      --addr string                      server listen addr (default "0.0.0.0")
      --alsologtostderr                  log to standard error as well as files
      --config string                    helm wrapper config (default "config.yaml")
      --debug                            enable verbose output
      --kube-context string              name of the kubeconfig context to use
      --kubeconfig string                path to the kubeconfig file
      --log_backtrace_at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                   If non-empty, write log files in this directory
      --logtostderr                      log to standard error instead of files (default true)
  -n, --namespace string                 namespace scope for this request
      --port string                      server listen port (default "8080")
      --registry-config string           path to the registry config file (default "/root/.config/helm/registry.json")
      --repository-cache string          path to the file containing cached repository indexes (default "/root/.cache/helm/repository")
      --repository-config string         path to the file containing repository names and URLs (default "/root/.config/helm/repositories.yaml")
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
pflag: help requested
```

+ `--config` helm-wrapper configuration: 

```
$ cat config-example.yaml
helmRepos:
  - name: bitnami
    url: https://charts.bitnami.com/bitnami
```

+ `--kubeconfig` default kubeconfig path is `~/.kube/config`.About `kubeconfig`, you can see [Configure Access to Multiple Clusters](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/).

### Run

```
$ ./helm-wrapper --config config-example.yaml
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /                         --> main.main.func1 (3 handlers)
[GIN-debug] GET    /api/envs                 --> main.getHelmEnvs (3 handlers)
[GIN-debug] GET    /api/repositories/charts  --> main.listRepoCharts (3 handlers)
[GIN-debug] PUT    /api/repositories         --> main.updateRepositories (3 handlers)
[GIN-debug] GET    /api/charts               --> main.showChartInfo (3 handlers)
[GIN-debug] GET    /api/namespaces/:namespace/releases --> main.listReleases (3 handlers)
[GIN-debug] GET    /api/namespaces/:namespace/releases/:release --> main.showReleaseInfo (3 handlers)
[GIN-debug] POST   /api/namespaces/:namespace/releases/:release --> main.installRelease (3 handlers)
[GIN-debug] PUT    /api/namespaces/:namespace/releases/:release --> main.upgradeRelease (3 handlers)
[GIN-debug] DELETE /api/namespaces/:namespace/releases/:release --> main.uninstallRelease (3 handlers)
[GIN-debug] PUT    /api/namespaces/:namespace/releases/:release/versions/:reversion --> main.rollbackRelease (3 handlers)
[GIN-debug] GET    /api/namespaces/:namespace/releases/:release/status --> main.getReleaseStatus (3 handlers)
[GIN-debug] GET    /api/namespaces/:namespace/releases/:release/histories --> main.listReleaseHistories (3 handlers)
```
