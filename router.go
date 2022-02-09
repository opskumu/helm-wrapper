package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

type respBody struct {
	Code  int         `json:"code"` // 0 or 1, 0 is ok, 1 is error
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func respErr(c *gin.Context, err error) {
	glog.Warningln(err)

	c.JSON(http.StatusOK, &respBody{
		Code:  500,
		Error: err.Error(),
	})
}

func respOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, &respBody{
		Code: 200,
		Data: data,
	})
}

func RegisterRouter(router *gin.Engine) {
	// helm env
	envs := router.Group("/api/envs")
	{
		envs.GET("", getHelmEnvs)
	}

	// helm repo
	repositories := router.Group("/api/repositories")
	{
		// helm repo listReleasesV2
		repositories.GET("", listRepos)
		// helm search repo
		repositories.GET("/charts", listRepoCharts)
		// helm repo update
		repositories.PUT("", updateRepos)
	}

	// helm chart
	charts := router.Group("/api/charts")
	{
		// helm show
		charts.GET("", showChartInfo)
		// upload chart
		charts.POST("/upload", uploadChart)
		// listReleasesV2 uploaded charts
		charts.GET("/upload", listUploadedCharts)
		// delete chart
		charts.DELETE("/upload/:chart", deleteChart)
	}

	// helm release
	releases := router.Group("/api/namespaces/:namespace/releases")
	{
		// helm list releases ->  helm list
		releases.GET("", listReleases)
		// helm get
		releases.GET("/:release", showReleaseInfo)
		// helm install
		releases.POST("/:release", installRelease)
		// helm upgrade
		releases.PUT("/:release", upgradeRelease)
		// helm uninstall
		releases.DELETE("/:release", uninstallRelease)
		// helm rollback
		releases.PUT("/:release/versions/:reversion", rollbackRelease)
		// helm status <RELEASE_NAME>
		releases.GET("/:release/status", getReleaseStatus)
		// helm release history
		releases.GET("/:release/histories", listReleaseHistories)
	}

	// Kubernetes Config File
	kubeConfig := router.Group("/api/k8s/config")
	{
		kubeConfig.POST("/upload", uploadKubeConfig)
	}

	fileRelease := router.Group("/api/namespaces/:namespace/releases/v2/:cluster")
	{
		// helm install
		fileRelease.POST("/install/:release", install)

		fileRelease.POST("/upgrade/:release", upgrade)

		// helm list releases ->  helm list
		fileRelease.GET("/list", listReleasesV2)
		// helm get
		fileRelease.GET("/:release", showReleaseInfoV2)

		// helm uninstall
		fileRelease.DELETE("/:release", uninstallReleaseV2)
		// helm rollback
		fileRelease.PUT("/:release/versions/:reversion", rollbackReleaseV2)
		// helm status <RELEASE_NAME>
		fileRelease.GET("/:release/status", getReleaseStatusV2)
		// helm release history
		fileRelease.GET("/:release/histories", listReleaseHistoriesV2)
	}
}
