package main

import (
	"os"

	"github.com/golang/glog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
)

func actionConfigInit(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	clientConfig := kube.GetConfig(settings.KubeConfig, settings.KubeContext, namespace)
	if settings.KubeToken != "" {
		clientConfig.BearerToken = &settings.KubeToken
	}
	if settings.KubeAPIServer != "" {
		clientConfig.APIServer = &settings.KubeAPIServer
	}
	err := actionConfig.Init(clientConfig, namespace, os.Getenv("HELM_DRIVER"), glog.Infof)
	if err != nil {
		glog.Errorf("%+v", err)
		return nil, err
	}

	return actionConfig, nil
}
