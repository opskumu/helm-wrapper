package main

import (
	"fmt"
	"os"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
)

type RegistryConfig struct {
	Host	 string
	Username string
	Password string
	CertFile string
	KeyFile  string
	CAFile   string
}

func repoEntryToRegistryConfig(repoEntry *repo.Entry) (*RegistryConfig, error) {
	if repoEntry.URL == "" || !strings.HasPrefix(repoEntry.URL, "oci://") {
		return nil, fmt.Errorf("Invalid OCI registry URL: %s", repoEntry.URL)
	}

	hostParts := strings.Split(repoEntry.URL, "oci://")
	if len(hostParts) != 2 {
		return nil, fmt.Errorf("Invalid OCI registry URL: %s", repoEntry.URL)
	}

	registryConfig := RegistryConfig{
		Host: hostParts[1],
		Username: repoEntry.Username,
		Password: repoEntry.Password,
		CertFile: repoEntry.CertFile,
		KeyFile: repoEntry.KeyFile,
		CAFile: repoEntry.CAFile,
	}

	return &registryConfig, nil
}

func chartPathOptionsToRegistryConfig(aimChart *string, chartPathOptions *action.ChartPathOptions) (*RegistryConfig, error) {
	if !strings.HasPrefix(*aimChart, "oci://") {
		return nil, fmt.Errorf("Invalid OCI chart url: %s", aimChart)
	}

	chartUrlParts := strings.Split(*aimChart, "oci://")
	if len(chartUrlParts) != 2 {
		return nil, fmt.Errorf("Invalid OCI chart url: %s", aimChart)
	}

	hostParts := strings.Split(chartUrlParts[1], "/")
	if len(hostParts) < 2 {
		return nil, fmt.Errorf("Invalid OCI chart url: %s", aimChart)
	}

	registryConfig := RegistryConfig{
		Host: hostParts[0],
		Username: chartPathOptions.Username,
		Password: chartPathOptions.Password,
		CertFile: chartPathOptions.CertFile,
		KeyFile: chartPathOptions.KeyFile,
		CAFile: chartPathOptions.CaFile,
	}

	return &registryConfig, nil
}

func createOCIRegistryClient(registryConfig *RegistryConfig) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(true),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stdout),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	}

	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to create registry client: %s", err)
	}

	if registryConfig.Username != "" && registryConfig.Password != "" {
		registryClient.Login(
			registryConfig.Host,
			registry.LoginOptBasicAuth(
				registryConfig.Username,
				registryConfig.Password,
			),
			registry.LoginOptInsecure(false),
			registry.LoginOptTLSClientConfig(
				registryConfig.CertFile,
				registryConfig.KeyFile,
				registryConfig.CAFile,
			),
		)
	}

	return registryClient, nil
}

func initRegistry(c *repo.Entry) (error) {
	registryConfig, err := repoEntryToRegistryConfig(c)
	if err != nil {
		return fmt.Errorf("Failed to convert repo entry to registry config: %s", err)
	}

	_, err = createOCIRegistryClient(registryConfig)
	return err
}

func createOCIRegistryClientForChartPathOptions(aimChart *string, chartPathOptions *action.ChartPathOptions) (*registry.Client, error) {
	if strings.HasPrefix(*aimChart, "oci://") {
		registryConfig, err := chartPathOptionsToRegistryConfig(aimChart, chartPathOptions)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert chart path options to registry config: %s", err)
		}

		return createOCIRegistryClient(registryConfig)
	}

	return nil, nil
}
