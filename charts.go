package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var readmeFileNames = []string{"readme.md", "readme.txt", "readme"}

type file struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func findReadme(files []*chart.File) (file *chart.File) {
	for _, file := range files {
		for _, n := range readmeFileNames {
			if strings.EqualFold(file.Name, n) {
				return file
			}
		}
	}
	return nil
}

func showChartInfo(c *gin.Context) {
	name := c.Query("chart")
	if name == "" {
		respErr(c, fmt.Errorf("chart name can not be empty"))
		return
	}
	// local charts with abs path *.tgz
	splitChart := strings.Split(name, ".")
	if splitChart[len(splitChart)-1] == "tgz" {
		name = helmConfig.UploadPath + "/" + name
	}

	info := c.Query("info") // readme, values, chart
	version := c.Query("version")

	client := action.NewShow(action.ShowAll)
	client.Version = version
	if info == string(action.ShowChart) {
		client.OutputFormat = action.ShowChart
	} else if info == string(action.ShowReadme) {
		client.OutputFormat = action.ShowReadme
	} else if info == string(action.ShowValues) {
		client.OutputFormat = action.ShowValues
	} else {
		respErr(c, fmt.Errorf("bad info %s, chart info only support readme/values/chart", info))
		return
	}

	cp, err := client.ChartPathOptions.LocateChart(name, settings)
	if err != nil {
		respErr(c, err)
		return
	}

	chrt, err := loader.Load(cp)
	if err != nil {
		respErr(c, err)
		return
	}

	if client.OutputFormat == action.ShowChart {
		respOK(c, chrt.Metadata)
		return
	}
	if client.OutputFormat == action.ShowValues {
		values := make([]*file, 0, len(chrt.Raw))
		for _, v := range chrt.Raw {
			values = append(values, &file{
				Name: v.Name,
				Data: string(v.Data),
			})
		}
		respOK(c, values)
		return
	}
	if client.OutputFormat == action.ShowReadme {
		respOK(c, string(findReadme(chrt.Files).Data))
		return
	}
}
