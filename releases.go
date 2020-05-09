package main

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"
	helmtime "helm.sh/helm/v3/pkg/time"
	"sigs.k8s.io/yaml"
)

type releaseInfo struct {
	Revision    int           `json:"revision"`
	Updated     helmtime.Time `json:"updated"`
	Status      string        `json:"status"`
	Chart       string        `json:"chart"`
	AppVersion  string        `json:"app_version"`
	Description string        `json:"description"`
}

type releaseHistory []releaseInfo

type releaseElement struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Revision     string `json:"revision"`
	Updated      string `json:"updated"`
	Status       string `json:"status"`
	Chart        string `json:"chart"`
	ChartVersion string `json:"chart_version"`
	AppVersion   string `json:"app_version"`

	Notes string `json:"notes,omitempty"`

	// TODO: Test Suite?
}

type releaseList []releaseElement

type releaseOptions struct {
	Values          string   `json:"values"`
	SetValues       []string `json:"set"`
	SetStringValues []string `json:"set_string"`
}

func formatChartname(c *chart.Chart) string {
	if c == nil || c.Metadata == nil {
		// This is an edge case that has happened in prod, though we don't
		// know how: https://github.com/helm/helm/issues/1347
		return "MISSING"
	}
	return fmt.Sprintf("%s-%s", c.Name(), c.Metadata.Version)
}

func formatAppVersion(c *chart.Chart) string {
	if c == nil || c.Metadata == nil {
		// This is an edge case that has happened in prod, though we don't
		// know how: https://github.com/helm/helm/issues/1347
		return "MISSING"
	}
	return c.AppVersion()
}

func mergeValues(options releaseOptions) (map[string]interface{}, error) {
	vals := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(options.Values), &vals)
	if err != nil {
		return vals, fmt.Errorf("failed parsing values")
	}

	for _, value := range options.SetValues {
		if err := strvals.ParseInto(value, vals); err != nil {
			return vals, fmt.Errorf("failed parsing set data")
		}
	}

	for _, value := range options.SetStringValues {
		if err := strvals.ParseIntoString(value, vals); err != nil {
			return vals, fmt.Errorf("failed parsing set_string data")
		}
	}

	return vals, nil
}

func getReleaseHistory(rls []*release.Release) (history releaseHistory) {
	for i := len(rls) - 1; i >= 0; i-- {
		r := rls[i]
		c := formatChartname(r.Chart)
		s := r.Info.Status.String()
		v := r.Version
		d := r.Info.Description
		a := formatAppVersion(r.Chart)

		rInfo := releaseInfo{
			Revision:    v,
			Status:      s,
			Chart:       c,
			AppVersion:  a,
			Description: d,
		}
		if !r.Info.LastDeployed.IsZero() {
			rInfo.Updated = r.Info.LastDeployed

		}
		history = append(history, rInfo)
	}

	return history
}

func constructReleaseElement(r *release.Release, showStatus bool) releaseElement {
	element := releaseElement{
		Name:         r.Name,
		Namespace:    r.Namespace,
		Revision:     strconv.Itoa(r.Version),
		Status:       r.Info.Status.String(),
		Chart:        r.Chart.Metadata.Name,
		ChartVersion: r.Chart.Metadata.Version,
		AppVersion:   r.Chart.Metadata.AppVersion,
	}
	if showStatus {
		element.Notes = r.Info.Notes
	}
	t := "-"
	if tspb := r.Info.LastDeployed; !tspb.IsZero() {
		t = tspb.String()
	}
	element.Updated = t

	return element
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}

	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func showReleaseInfo(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	info := c.Query("info")
	infos := []string{"all", "hooks", "manifest", "notes", "values"}
	infoMap := map[string]bool{}
	for _, i := range infos {
		infoMap[i] = true
	}
	if _, ok := infoMap[info]; !ok {
		respErr(c, fmt.Errorf("bad info %s, release info only support all/hooks/manifest/notes/values", info))
		return
	}

	actionConfig, err := actionConfigInit(namespace)
	if err != nil {
		respErr(c, err)
		return
	}
	if info == "values" {
		client := action.NewGetValues(&actionConfig)
		results, err := client.Run(name)
		if err != nil {
			respErr(c, err)
			return
		}
		respOK(c, results)
		return
	}

	client := action.NewGet(&actionConfig)
	results, err := client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}
	if info == "all" {
		results.Chart = nil
		respOK(c, results)
		return
	} else if info == "hooks" {
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

func installRelease(c *gin.Context) {
	name := c.Param("release")
	chart := c.Query("chart")
	namespace := c.Param("namespace")
	var options releaseOptions
	err := c.BindJSON(&options)
	if err != nil {
		respErr(c, err)
		return
	}

	vals, err := mergeValues(options)
	if err != nil {
		respErr(c, err)
		return
	}

	actionConfig, err := actionConfigInit(namespace)
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewInstall(&actionConfig)
	client.ReleaseName = name
	client.Namespace = namespace

	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
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

	_, err = client.Run(chartRequested, vals)
	if err != nil {
		respErr(c, err)
		return
	}

	respOK(c, nil)
}

func uninstallRelease(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	actionConfig, err := actionConfigInit(namespace)
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewUninstall(&actionConfig)
	_, err = client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}

	respOK(c, nil)
}

func rollbackRelease(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	reversionStr := c.Param("reversion")
	reversion, err := strconv.Atoi(reversionStr)
	if err != nil {
		respErr(c, err)
		return
	}

	actionConfig, err := actionConfigInit(namespace)
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewRollback(&actionConfig)
	client.Version = reversion
	err = client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}

	respOK(c, nil)
}

func upgradeRelease(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	chart := c.Query("chart")
	var options releaseOptions
	err := c.BindJSON(&options)
	if err != nil {
		respErr(c, err)
		return
	}

	vals, err := mergeValues(options)
	if err != nil {
		respErr(c, err)
		return
	}

	actionConfig, err := actionConfigInit(namespace)
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewUpgrade(&actionConfig)
	client.Namespace = namespace

	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
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

	_, err = client.Run(name, chartRequested, vals)
	if err != nil {
		respErr(c, err)
		return
	}

	respOK(c, nil)
}

func listReleases(c *gin.Context) {
	namespace := c.Param("namespace")
	actionConfig, err := actionConfigInit(namespace)
	if err != nil {
		respErr(c, err)
		return
	}

	client := action.NewList(&actionConfig)
	client.Deployed = true
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

func getReleaseStatus(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	actionConfig, err := actionConfigInit(namespace)
	if err != nil {
		respErr(c, err)
		return
	}

	client := action.NewStatus(&actionConfig)
	results, err := client.Run(name)
	if err != nil {
		respErr(c, err)
		return
	}
	element := constructReleaseElement(results, true)

	respOK(c, &element)
}

func listReleaseHistories(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	actionConfig, err := actionConfigInit(namespace)
	if err != nil {
		respErr(c, err)
		return
	}

	client := action.NewHistory(&actionConfig)
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
