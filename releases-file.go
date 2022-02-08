package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"io"
	"os"
	"strings"
)

func newInstallRelease(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")

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

func newUpgradeRelease(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	//aimChart := c.Query("chart")
	//kubeContext := c.Query("kube_context")
	//if aimChart == "" {
	//	respErr(c, fmt.Errorf("chart name can not be empty"))
	//	return
	//}
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
