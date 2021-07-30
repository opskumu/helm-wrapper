package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

type HelmConfig struct {
	UploadPath string        `yaml:"uploadPath"`
	HelmRepos  []*repo.Entry `yaml:"helmRepos"`
}

var (
	settings          = cli.New()
	defaultUploadPath = "/tmp/charts"
	helmConfig        = &HelmConfig{}
)

func main() {
	var (
		listenHost string
		listenPort string
		config     string
	)

	err := flag.Set("logtostderr", "true")
	if err != nil {
		glog.Fatalln(err)
	}
	pflag.CommandLine.StringVar(&listenHost, "addr", "0.0.0.0", "server listen addr")
	pflag.CommandLine.StringVar(&listenPort, "port", "8080", "server listen port")
	pflag.CommandLine.StringVar(&config, "config", "config.yaml", "helm wrapper config")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	settings.AddFlags(pflag.CommandLine)
	pflag.Parse()
	defer glog.Flush()

	configBody, err := ioutil.ReadFile(config)
	if err != nil {
		glog.Fatalln(err)
	}
	err = yaml.Unmarshal(configBody, helmConfig)
	if err != nil {
		glog.Fatalln(err)
	}

	// upload chart path
	if helmConfig.UploadPath == "" {
		helmConfig.UploadPath = defaultUploadPath
	} else {
		if !filepath.IsAbs(helmConfig.UploadPath) {
			glog.Fatalln("charts upload path is not absolute")
		}
	}
	_, err = os.Stat(helmConfig.UploadPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(helmConfig.UploadPath, 0755)
			if err != nil {
				glog.Fatalln(err)
			}
		} else {
			glog.Fatalln(err)
		}
	}

	// init repo
	for _, c := range helmConfig.HelmRepos {
		err = initRepos(c)
		if err != nil {
			glog.Fatalln(err)
		}
	}

	// router
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome helm wrapper server")
	})

	// register router
	RegisterRouter(router)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", listenHost, listenPort),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			glog.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	glog.Infoln("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
