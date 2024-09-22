package main

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	helmtime "helm.sh/helm/v3/pkg/time"
	"sigs.k8s.io/yaml"
)

var defaultTimeout = "5m0s"

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

type releaseOptions struct {
	// common
	DryRun                   bool     `json:"dry_run"`
	DisableHooks             bool     `json:"disable_hooks"`
	Wait                     bool     `json:"wait"`
	Devel                    bool     `json:"devel"`
	Description              string   `json:"description"`
	Atomic                   bool     `json:"atomic"`
	SkipCRDs                 bool     `json:"skip_crds"`
	SubNotes                 bool     `json:"sub_notes"`
	Timeout                  string   `json:"timeout"`
	WaitForJobs              bool     `json:"wait_for_jobs"`
	DisableOpenAPIValidation bool     `json:"disable_open_api_validation"`
	Values                   string   `json:"values"`
	SetValues                []string `json:"set"`
	SetStringValues          []string `json:"set_string"`
	ChartPathOptions

	// only install
	CreateNamespace  bool `json:"create_namespace"`
	DependencyUpdate bool `json:"dependency_update"`

	// only upgrade
	Install bool `json:"install"`

	// only rollback
	MaxHistory int `json:"history_max"`

	// upgrade or rollback
	Force         bool `json:"force"`
	Recreate      bool `json:"recreate"`
	ReuseValues   bool `json:"reuse_values"`
	CleanupOnFail bool `json:"cleanup_on_fail"`
}

// ChartPathOptions captures common options used for controlling chart paths
type ChartPathOptions struct {
	CaFile                string `json:"ca_file"`              // --ca-file
	CertFile              string `json:"cert_file"`            // --cert-file
	KeyFile               string `json:"key_file"`             // --key-file
	InsecureSkipTLSverify bool   `json:"insecure_skip_verify"` // --insecure-skip-verify
	Keyring               string `json:"keyring"`              // --keyring
	Password              string `json:"password"`             // --password
	RepoURL               string `json:"repo"`                 // --repo
	Username              string `json:"username"`             // --username
	Verify                bool   `json:"verify"`               // --verify
	Version               string `json:"version"`              // --version
}

// helm List struct
type releaseListOptions struct {
	// All ignores the limit/offset
	All bool `json:"all"`
	// AllNamespaces searches across namespaces
	AllNamespaces bool `json:"all_namespaces"`
	// Overrides the default lexicographic sorting
	ByDate      bool `json:"by_date"`
	SortReverse bool `json:"sort_reverse"`
	// Limit is the number of items to return per Run()
	Limit int `json:"limit"`
	// Offset is the starting index for the Run() call
	Offset int `json:"offset"`
	// Filter is a filter that is applied to the results
	Filter       string `json:"filter"`
	Uninstalled  bool   `json:"uninstalled"`
	Superseded   bool   `json:"superseded"`
	Uninstalling bool   `json:"uninstalling"`
	Deployed     bool   `json:"deployed"`
	Failed       bool   `json:"failed"`
	Pending      bool   `json:"pending"`
}

// helm Uninstall struct
type releaseUninstallOptions struct {
	DisableHooks        bool          `json:"disable_hooks"`
	DryRun              bool          `json:"dry_run"`
	IgnoreNotFound      bool          `json:"ignore_not_found"`
	KeepHistory         bool          `json:"keep_history"`
	Wait                bool          `json:"wait"`
	DeletionPropagation string        `json:"delete_propagation"`
	Timeout             time.Duration `json:"timeout"`
	Description         string        `json:"description"`
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
	values, err := readValues(options.Values)
	if err != nil {
		return vals, err
	}
	err = yaml.Unmarshal(values, &vals)
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
	kubeConfig := c.Query("kube_config")
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
	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
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

func installRelease(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	aimChart := c.Query("chart")
	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

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
	err := c.ShouldBindJSON(&options)
	if err != nil && err != io.EOF {
		respErr(c, err)
		return
	}

	if err = runInstall(name, namespace, kubeContext, aimChart, kubeConfig, options); err != nil {
		respErr(c, err)
		return
	}

	respOK(c, err)
	return
}

func runInstall(name, namespace, kubeContext, aimChart, kubeConfig string, options releaseOptions) (err error) {
	vals, err := mergeValues(options)
	if err != nil {
		return
	}

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
	if err != nil {
		return
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
		return
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
		return
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err = action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getter.All(settings),
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
				}
				if err = man.Update(); err != nil {
					return
				}
			} else {
				return
			}
		}
	}

	_, err = client.Run(chartRequested, vals)
	if err != nil {
		return
	}

	return nil
}

func uninstallRelease(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

	var options releaseUninstallOptions

	err := c.ShouldBindJSON(&options)
	if err != nil && err != io.EOF {
		respErr(c, err)
		return
	}

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
	if err != nil {
		respErr(c, err)
		return
	}
	client := action.NewUninstall(actionConfig)
	client.DisableHooks = options.DisableHooks
	client.DryRun = options.DryRun
	client.IgnoreNotFound = options.IgnoreNotFound
	client.KeepHistory = options.KeepHistory
	client.Wait = options.Wait
	client.DeletionPropagation = options.DeletionPropagation
	client.Timeout = options.Timeout
	client.Description = options.Description

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
	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

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

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
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
	if options.Timeout == "" {
		options.Timeout = defaultTimeout
	}
	client.Timeout, err = time.ParseDuration(options.Timeout)
	if err != nil {
		return
	}

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
	aimChart := c.Query("chart")
	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

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
	err := c.ShouldBindJSON(&options)
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

	_, err = client.Run(name, chartRequested, vals)
	if err != nil {
		respErr(c, err)
		return
	}

	respOK(c, nil)
}

func listReleases(c *gin.Context) {
	namespace := c.Param("namespace")
	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

	var options releaseListOptions
	err := c.ShouldBindJSON(&options)
	if err != nil && err != io.EOF {
		respErr(c, err)
		return
	}
	if options.AllNamespaces {
		namespace = ""
	}
	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
	if err != nil {
		respErr(c, err)
		return
	}

	client := action.NewList(actionConfig)

	// merge list options
	client.All = options.All
	client.AllNamespaces = options.AllNamespaces
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

func getReleaseStatus(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
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

func listReleaseHistories(c *gin.Context) {
	name := c.Param("release")
	namespace := c.Param("namespace")
	kubeContext := c.Query("kube_context")
	kubeConfig := c.Query("kube_config")

	actionConfig, err := actionConfigInit(InitKubeInformation(namespace, kubeContext, kubeConfig))
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

func readValues(filePath string) ([]byte, error) {
	u, _ := url.Parse(filePath)
	if u == nil {
		return []byte(filePath), nil
	}

	p := getter.All(settings)
	g, err := p.ByScheme(u.Scheme)
	// if scheme not support, return self
	if err != nil {
		return []byte(filePath), nil
	}

	data, err := g.Get(filePath, getter.WithURL(filePath))
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}
