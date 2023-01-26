package main

import (
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type config struct {
	InputFiles []string

	UseK8sAPI     bool
	K8sKubeConfig string
	K8sNamespace  string
	K8sJobName    string
}

func initConfig() {
	pflag.StringSlice("file", []string{}, "List of local files to aggregate")
	pflag.String("kubeconfig", "", "Kubernetes kubeconfig location; if empty in-cluster auth will be used")
	pflag.String("namespace", "default", "Kubernetes Job namespace to grab logs from")
	pflag.String("job", "", "Kubernetes Job name to grab logs from")
	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("report_aggregator")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func loadConfig() *config {
	jobName := viper.GetString("job")

	return &config{
		InputFiles:    viper.GetStringSlice("file"),
		K8sKubeConfig: viper.GetString("kubeconfig"),
		K8sNamespace:  viper.GetString("namespace"),
		K8sJobName:    jobName,
		UseK8sAPI:     jobName != "",
	}
}
