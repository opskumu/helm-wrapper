# A [Helm3](https://github.com/helm/helm) HTTP Wrapper With [Go SDK](https://helm.sh/docs/topics/advanced/#go-sdk)

+ [中文文档](README_CN.md)

helm-wrapper is a helm3 HTTP wrapper with [helm Go SDK](https://helm.sh/docs/topics/advanced/#go-sdk). With helm-wrapper, you can use HTTP RESTFul API do something like helm commondline (install/uninstall/upgrade/get/list/rollback...).

## Support API


* If there are some APIs (`release` related) need to support multiple clusters,you can use the parameters below

| Params | Description |
|:---| :--- |
| kube_context | Support distinguish multiple clusters by the`kube_context`  |
| kube_config  | Support distinguish multiple clusters by the`kube_config`  |



+ helm install
    - `POST`
    - `/api/namespaces/:namespace/releases/:release?chart=<chartName>`

POST Body: 

``` json
{
    "dry_run": false,           // `--dry-run`
    "disable_hooks": false,     // `--no-hooks`
    "wait": false,              // `--wait`
    "timeout": "5m0s",          // `--timeout`
    "devel": false,             // `--false`
    "description": "",          // `--description`
    "atomic": false,            // `--atomic`
    "skip_crds": false,         // `--skip-crds`
    "sub_notes": false,         // `--render-subchart-notes`
    "create_namespace": false,  // `--create-namespace`
    "dependency_update": false, // `--dependency-update`
    "values": "",               // `--values`
    "set": [],                  // `--set`
    "set_string": [],           // `--set-string`
    "ca_file": "",              // `--ca-file`
    "cert_file": "",            // `--cert-file`
    "key_file": "",             // `--key-file`
    "insecure_skip_verify": false, // `--insecure-skip-verify`
    "keyring": "",              // `--keyring`
    "password": "",             // `--password`
    "repo": "",                 // `--repo`
    "username": "",             // `--username`
    "verify": false,            // `--verify`
    "version": ""               // `--version`
}
```

> `"values"` -> helm install `--values` option 

+ helm uninstall
    - `DELETE`
    - `/api/namespaces/:namespace/releases/:release`

Delete Body:
``` json 
{
    "dry_run": false,           // `--dry-run`
    "disable_hooks": false,     // `--no-hooks`
    "wait": false,              // `--wait`
    "timeout": "5m0s",          // `--timeout`
    "description": "",          // `--description`
    "ignore_not_found": false,  // `--ignore-not-found`
}

```
+ helm upgrade
    - `PUT`
    - `/api/namespaces/:namespace/releases/:release?chart=<chartName>`

PUT Body: 

``` json
{
    "dry_run": false,           // `--dry-run`
    "disable_hooks": false,     // `--no-hooks`
    "wait": false,              // `--wait`
    "timeout": "5m0s",          // `--timeout`
    "devel": false,             // `--false`
    "description": "",          // `--description`
    "atomic": false,            // `--atomic`
    "skip_crds": false,         // `--skip-crds`
    "sub_notes": false,         // `--render-subchart-notes`
    "force": false,             // `--force`
    "install": false,           // `--install`
    "recreate": false,          // `--recreate`
    "reuse_values": false,      // `--reuse-values`
    "cleanup_on_fail": false,   // `--cleanup-on-fail`
    "values": "",               // `--values`
    "set": [],                  // `--set`
    "set_string": [],           // `--set-string`
    "ca_file": "",              // `--ca-file`
    "cert_file": "",            // `--cert-file`
    "key_file": "",             // `--key-file`
    "insecure_skip_verify": false, // `--insecure-skip-verify`
    "keyring": "",              // `--keyring`
    "password": "",             // `--password`
    "repo": "",                 // `--repo`
    "username": "",             // `--username`
    "verify": false,            // `--verify`
    "version": ""               // `--version`
}
```

> `"values"` -> helm install `--values` option 

+ helm rollback
    - `PUT`
    - `/api/namespaces/:namespace/releases/:release/versions/:reversion`

PUT Body (optional):

``` json
{
    "dry_run": false,           // `--dry-run`
    "disable_hooks": false,     // `--no-hooks`
    "wait": false,              // `--wait`
    "timeout": "5m0s",          // `--timeout`
    "force": false,             // `--force`
    "recreate": false,          // `--recreate`
    "cleanup_on_fail": false,   // `--cleanup-on-fail`
    "history_max":              // `--history-max` int
}
```

+ helm list
    - `GET`
    - `/api/namespaces/:namespace/releases`

Body:

``` json
{
    "all": false,               // `--all`
    "all_namespaces": false,    // `--all-namespaces`
    "by_date": false,           // `--date`
    "sort_reverse": false,      // `--reverse`
    "limit":  ,                 // `--max`
    "offset": ,                 // `--offset`
    "filter": "",               // `--filter`
    "uninstalled": false,       // `--uninstalled`
    "uninstalling": false,      // `--uninstalling`
    "superseded": false,        // `--superseded`
    "failed": false,            // `--failed`
    "deployed": false,          // `--deployed`
    "pending": false            // `--pending`
}
```

+ helm get
    - `GET`
    - `/api/namespaces/:namespace/releases/:release`

| Params | Description |
| :--- | :--- |
| info | support hooks/manifest/notes/values, default values |
| output | get values output format (only info==values), support json/yaml, default json |

+ helm release history
    - `GET`
    - `/api/namespaces/:namespace/releases/:release/histories`


+ helm show
    - `GET`
    - `/api/charts`

| Params | Description |
| :--- | :--- |
| chart  | chart name, required|
| info   | support all/readme/values/chart, default all |
| version | --version |

+ helm search repo
    - `GET`
    - `/api/repositories/charts`

| Params | Description |
| :--- | :--- |
| keyword | search keyword，required |
| version | chart version |
| versions | if "true", all versions |

+ helm repo list
    - `GET`
    - `/api/repositories`

+ helm repo update
    - `PUT`
    - `/api/repositories`

+ helm env
    - `GET`
    - `/api/envs`

+ upload chart
    - `POST`
    - `/api/charts/upload`

| Params | Description |
| :--- | :--- |
| chart | upload chart file, with suffix .tgz |

+ list local charts
    - `GET`
    - `/api/charts/upload`

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
uploadPath: /tmp/charts
helmRepos:
  - name: bitnami
    url: https://charts.bitnami.com/bitnami
```

+ `--kubeconfig` default kubeconfig path is `~/.kube/config`.About `kubeconfig`, you can see [Configure Access to Multiple Clusters](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/).

### Run

```
$ ./helm-wrapper --config </path/to/config.yaml> --kubeconfig </path/to/kubeconfig>
```

#### Deploy in Kubernetes Cluster

replace deployment/deployment.yaml with helm-wrapper image, then:

```
kubectl create -f ./deployment 
```

> __Noets:__ with deployment/rbac.yaml, you not need `--kubeconfig`

## OCI Registries
For helm install, upgrade, and upgrade-install, replace the `chart` parameter with the OCI registry URL for the chart.

### Authentication
There are three ways to authenticate to an OCI registry: request body, config.yaml, and helm settings registry config file.  You only need to use one of these methods, but it also shouldn't cause any issues if you provide all three (not that it is advisable to do so).

#### Request body
You can include a `username` and `password` in the request body.

#### config.yaml
Similar to how you can add a helm repo under the `helmRepos` key in `config.yaml`, you can add a registry under the `helmRegistries` key.  The domain must match the chart registry URL in the upgrade or install request.  For example:
```
helmRegistries:
 - name: registry_name
   url: oci://registry.com
   username: username
   password: password
```

#### Helm settings registry config file
Both of the previous methods will also create/update the registry config file (the same as the helm CLI).  However, you can also put this file in the container and helm-wrapper will use that to authenticate.  By default, this file is located at `/home/helm/.config/helm/registry/config.json`.  Again, the domain must match the chart registry URL in the upgrade or install request.  Refer to helm documentation on how to configure this.  You can also use one of the other authentication methods and then look at the file that is created in the container.
