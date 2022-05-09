package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"io"
	"os"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
)

func setKubeConfig(c *gin.Context) {
	cluster := c.Param("cluster")
	settings.KubeConfig = "/tmp/k8s-config/" + cluster + "/config"
}

func install(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")

	setKubeConfig(c)

	args := c.Request.FormValue("args")

	file, header, err := c.Request.FormFile("chart")
	if err != nil {
		respErr(c, err)
		return
	}

	filename := header.Filename
	t := strings.Split(filename, ".")
	if t[len(t)-1] != "tgz" {
		respErr(c, fmt.Errorf("chart file suffix must .tgz"))
		return
	}

	out, err := os.Create(helmConfig.UploadPath + "/" + filename)
	if err != nil {
		respErr(c, err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		respErr(c, err)
		return
	}

	aimChart := filename
	//kubeContext := c.Query("kube_context")
	if aimChart == "" {
		respErr(c, fmt.Errorf("chart name can not be empty"))
		return
	}

	// install with local uploaded charts, *.tgz
	splitChart := strings.Split(aimChart, ".")
	if splitChart[len(splitChart)-1] == "tgz" && !strings.Contains(aimChart, ":") {
		aimChart = helmConfig.UploadPath + "/" + aimChart
	}

	var options releaseOptions
	//err = c.ShouldBindJSON(&options)
	if args != "" {
		byteSets := []byte(args)
		err = json.Unmarshal(byteSets, &options)
		if err != nil && err != io.EOF {
			respErr(c, err)
			return
		}
	}

	vals, err := mergeValues(options)
	if err != nil {
		respErr(c, err)
		return
	}

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, ""))
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewInstall(actionConfig)
	client.ReleaseName = name
	client.Namespace = namespace

	// merge install options
	client.DryRun = options.DryRun
	client.DisableHooks = options.DisableHooks
	client.Wait = options.Wait
	client.Devel = options.Devel
	client.Description = options.Description
	client.Atomic = options.Atomic
	client.SkipCRDs = options.SkipCRDs
	client.SubNotes = options.SubNotes
	client.Timeout = options.Timeout
	client.CreateNamespace = options.CreateNamespace
	client.DependencyUpdate = options.DependencyUpdate

	// merge chart path options
	client.ChartPathOptions.CaFile = options.ChartPathOptions.CaFile
	client.ChartPathOptions.CertFile = options.ChartPathOptions.CertFile
	client.ChartPathOptions.KeyFile = options.ChartPathOptions.KeyFile
	client.ChartPathOptions.InsecureSkipTLSverify = options.ChartPathOptions.InsecureSkipTLSverify
	client.ChartPathOptions.Keyring = options.ChartPathOptions.Keyring
	client.ChartPathOptions.Password = options.ChartPathOptions.Password
	client.ChartPathOptions.RepoURL = options.ChartPathOptions.RepoURL
	client.ChartPathOptions.Username = options.ChartPathOptions.Username
	client.ChartPathOptions.Verify = options.ChartPathOptions.Verify
	client.ChartPathOptions.Version = options.ChartPathOptions.Version

	cp, err := client.ChartPathOptions.LocateChart(aimChart, settings)
	if err != nil {
		respErr(c, err)
		return
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		respErr(c, err)
		return
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		respErr(c, err)
		return
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getter.All(settings),
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					respErr(c, err)
					return
				}
			} else {
				respErr(c, err)
				return
			}
		}
	}

	info, err := client.Run(chartRequested, vals)
	if err != nil {
		respErr(c, err)
		return
	}

	respOK(c, info)
}

func upgrade(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")

	setKubeConfig(c)

	args := c.Request.FormValue("args")
	// 文件保存到缓存目录
	file, header, err := c.Request.FormFile("chart")
	if file == nil {
		respErr(c, fmt.Errorf("chart name can not be empty"))
		return
	}
	if err != nil {
		respErr(c, err)
		return
	}

	filename := header.Filename
	t := strings.Split(filename, ".")
	if t[len(t)-1] != "tgz" {
		respErr(c, fmt.Errorf("chart file suffix must .tgz"))
		return
	}

	out, err := os.Create(helmConfig.UploadPath + "/" + filename)
	if err != nil {
		respErr(c, err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		respErr(c, err)
		return
	}

	aimChart := filename

	// upgrade with local uploaded charts *.tgz
	splitChart := strings.Split(aimChart, ".")
	if splitChart[len(splitChart)-1] == "tgz" {
		aimChart = helmConfig.UploadPath + "/" + aimChart
	}

	var options releaseOptions
	//err := c.ShouldBindJSON(&options)
	byteSets := []byte(args)
	err = json.Unmarshal(byteSets, &options)
	if err != nil && err != io.EOF {
		respErr(c, err)
		return
	}
	vals, err := mergeValues(options)
	if err != nil {
		respErr(c, err)
		return
	}
	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, ""))
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace

	// merge upgrade options
	client.DryRun = options.DryRun
	client.DisableHooks = options.DisableHooks
	client.Wait = options.Wait
	client.Devel = options.Devel
	client.Description = options.Description
	client.Atomic = options.Atomic
	client.SkipCRDs = options.SkipCRDs
	client.SubNotes = options.SubNotes
	client.Timeout = options.Timeout
	client.Force = options.Force
	client.Install = options.Install
	client.Recreate = options.Recreate
	client.CleanupOnFail = options.CleanupOnFail

	// merge chart path options
	client.ChartPathOptions.CaFile = options.ChartPathOptions.CaFile
	client.ChartPathOptions.CertFile = options.ChartPathOptions.CertFile
	client.ChartPathOptions.KeyFile = options.ChartPathOptions.KeyFile
	client.ChartPathOptions.InsecureSkipTLSverify = options.ChartPathOptions.InsecureSkipTLSverify
	client.ChartPathOptions.Keyring = options.ChartPathOptions.Keyring
	client.ChartPathOptions.Password = options.ChartPathOptions.Password
	client.ChartPathOptions.RepoURL = options.ChartPathOptions.RepoURL
	client.ChartPathOptions.Username = options.ChartPathOptions.Username
	client.ChartPathOptions.Verify = options.ChartPathOptions.Verify
	client.ChartPathOptions.Version = options.ChartPathOptions.Version

	cp, err := client.ChartPathOptions.LocateChart(aimChart, settings)
	if err != nil {
		respErr(c, err)
		return
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		respErr(c, err)
		return
	}
	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			respErr(c, err)
			return
		}
	}

	info, err := client.Run(name, chartRequested, vals)
	if err != nil {
		respErr(c, err)
		return
	}

	respOK(c, info)
}

func listReleasesV2(c *gin.Context) {
	namespace := c.Param("namespace")

	setKubeConfig(c)

	kubeContext := c.Query("kube_context")
	var options releaseListOptions
	err := c.ShouldBindJSON(&options)
	if err != nil && err != io.EOF {
		respErr(c, err)
		return
	}
	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext))
	if err != nil {
		respErr(c, err)
		return
	}

	client := action.NewList(actionConfig)

	// merge listReleasesV2 options
	client.All = options.All
	client.AllNamespaces = options.AllNamespaces
	if client.AllNamespaces {
		err = actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), glog.Infof)
		if err != nil {
			respErr(c, err)
			return
		}
	}
	client.ByDate = options.ByDate
	client.SortReverse = options.SortReverse
	client.Limit = options.Limit
	client.Offset = options.Offset
	client.Filter = options.Filter
	client.Uninstalled = options.Uninstalled
	client.Superseded = options.Superseded
	client.Uninstalling = options.Uninstalling
	client.Deployed = options.Deployed
	client.Failed = options.Failed
	client.Pending = options.Pending
	client.SetStateMask()

	results, err := client.Run()
	if err != nil {
		respErr(c, err)
		return
	}

	// Initialize the array so no results returns an empty array instead of null
	elements := make([]releaseElement, 0, len(results))
	for _, r := range results {
		elements = append(elements, constructReleaseElement(r, false))
	}

	respOK(c, elements)
}

func showReleaseInfoV2(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	setKubeConfig(c)
	info := c.Query("info")
	if info == "" {
		info = "values"
	}
	kubeContext := c.Query("kube_context")
	infos := []string{"hooks", "manifest", "notes", "values"}
	infoMap := map[string]bool{}
	for _, i := range infos {
		infoMap[i] = true
	}
	if _, ok := infoMap[info]; !ok {
		respErr(c, fmt.Errorf("bad info %s, release info only support hooks/manifest/notes/values", info))
		return
	}
	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext))
	if err != nil {
		respErr(c, err)
		return
	}

	if info == "values" {
		output := c.Query("output")
		// get values output format
		if output == "" {
			output = "json"
		}
		if output != "json" && output != "yaml" {
			respErr(c, fmt.Errorf("invalid format type %s, output only support json/yaml", output))
			return
		}

		client := action.NewGetValues(actionConfig)
		client.AllValues = true
		results, err := client.Run(name)
		if err != nil {
			respErr(c, err)
			return
		}
		if output == "yaml" {
			obj, err := yaml.Marshal(results)
			if err != nil {
				respErr(c, err)
				return
			}
			respOK(c, string(obj))
			return
		}
		respOK(c, results)
		return
	}

	client := action.NewGet(actionConfig)
	results, err := client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}
	// TODO: support all
	if info == "hooks" {
		if len(results.Hooks) < 1 {
			respOK(c, []*release.Hook{})
			return
		}
		respOK(c, results.Hooks)
		return
	} else if info == "manifest" {
		respOK(c, results.Manifest)
		return
	} else if info == "notes" {
		respOK(c, results.Info.Notes)
		return
	}
}

func uninstallReleaseV2(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	setKubeConfig(c)
	kubeContext := c.Query("kube_context")

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext))
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewUninstall(actionConfig)
	_, err = client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}

	respOK(c, nil)
}

func rollbackReleaseV2(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	reversionStr := c.Param("reversion")
	setKubeConfig(c)
	kubeContext := c.Query("kube_context")
	reversion, err := strconv.Atoi(reversionStr)
	if err != nil {
		respErr(c, err)
		return
	}

	var options releaseOptions
	err = c.ShouldBindJSON(&options)
	if err != nil && err != io.EOF {
		respErr(c, err)
		return
	}

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext))
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewRollback(actionConfig)
	client.Version = reversion

	// merge rollback options
	client.CleanupOnFail = options.CleanupOnFail
	client.Wait = options.Wait
	client.DryRun = options.DryRun
	client.DisableHooks = options.DisableHooks
	client.Force = options.Force
	client.Recreate = options.Recreate
	client.MaxHistory = options.MaxHistory
	client.Timeout = options.Timeout

	err = client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, nil)
}

func getReleaseStatusV2(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	setKubeConfig(c)
	kubeContext := c.Query("kube_context")

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext))
	if err != nil {
		respErr(c, err)
		return
	}

	client := action.NewStatus(actionConfig)
	results, err := client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}
	element := constructReleaseElement(results, true)

	respOK(c, &element)
}

func listReleaseHistoriesV2(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	setKubeConfig(c)
	kubeContext := c.Query("kube_context")

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext))
	if err != nil {
		respErr(c, err)
		return
	}

	client := action.NewHistory(actionConfig)
	results, err := client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}
	if len(results) == 0 {
		respOK(c, &releaseHistory{})
		return
	}

	respOK(c, getReleaseHistory(results))
}
