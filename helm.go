package main

import (
	"os"

	"github.com/golang/glog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func actionConfigInit(namespace string) (action.Configuration, error) {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), glog.Infof)
	if err != nil {
		glog.Errorf("%+v", err)
		return *actionConfig, err
	}

	return *actionConfig, nil
}
