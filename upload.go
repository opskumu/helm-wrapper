package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func uploadChart(c *gin.Context) {
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

	respOK(c, nil)
}

func listUploadedCharts(c *gin.Context) {
	charts := []string{}
	files, err := ioutil.ReadDir(helmConfig.UploadPath)
	if err != nil {
		respErr(c, err)
		return
	}
	for _, f := range files {
		t := strings.Split(f.Name(), ".")
		if t[len(t)-1] == "tgz" {
			charts = append(charts, f.Name())
		}
	}

	respOK(c, charts)
}
