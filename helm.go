package main

import (
	"os"

	"github.com/golang/glog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type KubeInformation struct {
	AimNamespace string
	AimContext   string
	AimConfig    string
}

func InitKubeInformation(namespace, context, config string) *KubeInformation {
	return &KubeInformation{
		AimNamespace: namespace,
		AimContext:   context,
		AimConfig:    config,
	}
}

func actionConfigInit(kubeInfo *KubeInformation) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if kubeInfo.AimContext == "" {
		kubeInfo.AimContext = settings.KubeContext
	}
	clientConfig := new(genericclioptions.ConfigFlags)
	if kubeInfo.AimConfig == "" {
		clientConfig = kube.GetConfig(settings.KubeConfig, kubeInfo.AimContext, kubeInfo.AimNamespace)
	} else {
		clientConfig = kube.GetConfig(kubeInfo.AimConfig, kubeInfo.AimContext, kubeInfo.AimNamespace)
	}
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
