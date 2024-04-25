package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"io"
	"os"
	"strings"
	"time"
)

func setKubeConfig(c *gin.Context) {
	cluster := c.Param("cluster")
	settings.KubeConfig = helmConfig.UploadPath + "/k8s-config/" + cluster + "/config"
}

func install(c *gin.Context) {
	setKubeConfig(c)
	installReleaseV2(c)
}

func installReleaseV2(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")

	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

	args := c.Request.FormValue("args")

	// 新增
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
	//结束

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
	info, err := runInstallV2(name, namespace, kubeContext, aimChart, kubeConfig, options)
	if err != nil {
		respErr(c, err)
		return
	} else {
		respOK(c, info)
	}
}

func runInstallV2(name, namespace, kubeContext, aimChart, kubeConfig string, options releaseOptions) (*release.Release, error) {
	// runInstall方法，return增加了err的返回
	vals, err := mergeValues(options)
	if err != nil {
		return nil, err
	}

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
	if err != nil {
		return nil, err
	}
	client := action.NewInstall(actionConfig)
	client.ReleaseName = name
	client.Namespace = namespace

	// merge install options
	client.DryRun = options.DryRun
	client.DisableHooks = options.DisableHooks
	client.Wait = options.Wait
	if options.Timeout == "" {
		options.Timeout = defaultTimeout
	}
	client.Timeout, err = time.ParseDuration(options.Timeout)
	if err != nil {
		return nil, err
	}
	client.WaitForJobs = options.WaitForJobs
	client.Devel = options.Devel
	client.Description = options.Description
	client.Atomic = options.Atomic
	client.SkipCRDs = options.SkipCRDs
	client.SubNotes = options.SubNotes
	client.DisableOpenAPIValidation = options.DisableOpenAPIValidation
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
		return nil, err
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return nil, err
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
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}
	// 增加Info信息的接收
	info, err := client.Run(chartRequested, vals)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func upgrade(c *gin.Context) {
	setKubeConfig(c)
	upgradeReleaseV2(c)
}

func upgradeReleaseV2(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")

	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

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
	if aimChart == "" {
		respErr(c, fmt.Errorf("chart name can not be empty"))
		return
	}

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
	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
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
	client.Force = options.Force
	if options.Timeout == "" {
		options.Timeout = defaultTimeout
	}
	client.Timeout, err = time.ParseDuration(options.Timeout)
	if err != nil {
		return
	}
	client.Install = options.Install
	client.MaxHistory = options.MaxHistory
	client.Recreate = options.Recreate
	client.ReuseValues = options.ReuseValues
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

	if client.Install {
		hisClient := action.NewHistory(actionConfig)
		hisClient.Max = 1
		if _, err := hisClient.Run(name); err == driver.ErrReleaseNotFound {
			err = runInstall(name, namespace, kubeContext, aimChart, kubeConfig, options)
			if err != nil {
				respErr(c, err)
				return
			}

			respOK(c, err)
			return
		} else if err != nil {
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
	setKubeConfig(c)
	listReleases(c)
}

func showReleaseInfoV2(c *gin.Context) {
	setKubeConfig(c)
	showReleaseInfo(c)
}

func uninstallReleaseV2(c *gin.Context) {
	setKubeConfig(c)
	uninstallRelease(c)
}

func rollbackReleaseV2(c *gin.Context) {
	setKubeConfig(c)
	rollbackRelease(c)
}

func getReleaseStatusV2(c *gin.Context) {
	setKubeConfig(c)
	getReleaseStatus(c)
}

func listReleaseHistoriesV2(c *gin.Context) {
	setKubeConfig(c)
	listReleaseHistories(c)
}
