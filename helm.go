package main

import (
	"os"

	"github.com/golang/glog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
)

type KubeInformation struct {
	AimNamespace string
	AimContext   string
}

func InitKubeInformation(namespace, context string) *KubeInformation {
	return &KubeInformation{
		AimNamespace: namespace,
		AimContext:   context,
	}
}

func actionConfigInit(kubeInfo *KubeInformation) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if kubeInfo.AimContext == "" {
		kubeInfo.AimContext = settings.KubeContext
	}
	clientConfig := kube.GetConfig(settings.KubeConfig, kubeInfo.AimContext, kubeInfo.AimNamespace)
	if settings.KubeToken != "" {
		clientConfig.BearerToken = &settings.KubeToken
	}
	if settings.KubeAPIServer != "" {
		clientConfig.APIServer = &settings.KubeAPIServer
	}
	err := actionConfig.Init(clientConfig, kubeInfo.AimNamespace, os.Getenv("HELM_DRIVER"), glog.Infof)
	if err != nil {
		glog.Errorf("%+v", err)
		return nil, err
	}

	return actionConfig, nil
}
