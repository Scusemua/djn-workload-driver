package config

import (
	"flag"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

const (
	OptionName    = "name"
	OptionDefault = "default"
	OptionDesc    = "description"
)

type Configuration struct {
	SpoofCluster        bool   `yaml:"spoof-gateway" description:"If true, use the fake cluster."`
	InCluster           bool   `yaml:"in-cluster" description:"Should be true if running from within the kubernetes cluster."`
	KernelQueryInterval string `yaml:"kernel-query-interval" default:"5s" description:"How frequently to query the Cluster for updated kernel information."`
	NodeQueryInterval   string `yaml:"node-query-interval" default:"10s" description:"How frequently to query the Cluster for updated Kubernetes node information."`
	KubeConfig          string `yaml:"kubeconfig" description:"Absolute path to the kubeconfig file."`
}

func (opts *Configuration) String() string {
	return fmt.Sprintf("Configuration[SpoofCluster=%v,InCluster=%v,KernelQueryInterval='%s',NodeQueryInterval='%s',KubeConfig='%s']", opts.SpoofCluster, opts.InCluster, opts.KernelQueryInterval, opts.NodeQueryInterval, opts.KubeConfig)
}

func GetConfiguration() *Configuration {
	var spoofFlag = flag.Bool("spoof-cluster", true, "Spoof the connection to the Cluster Gateway.")
	var inClusterFlag = flag.Bool("in-cluster", false, "Should be true if running from within the kubernetes cluster.")
	var kernelQueryIntervalFlag = flag.String("kernel-query-interval", "60s", "How often to refresh kernels from Cluster Gateway.")
	var nodeQueryIntervalFlag = flag.String("node-query-interval", "120s", "How often to refresh nodes from Cluster Gateway.")

	var kubeconfigFlag *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfigFlag = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfigFlag = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	return &Configuration{
		SpoofCluster:        *spoofFlag,
		InCluster:           *inClusterFlag,
		KernelQueryInterval: *kernelQueryIntervalFlag,
		NodeQueryInterval:   *nodeQueryIntervalFlag,
		KubeConfig:          *kubeconfigFlag,
	}
}

// func GetOptions() *Configuration {
// 	var yamlPath string
// 	flag.StringVar(&yamlPath, "config", "config.yaml", "Path to the YAML configuration file.")
// 	flag.Parse()

// 	yamlFile, err := os.ReadFile(yamlPath)
// 	if err != nil {
// 		log.Printf("[ERROR] Failed to read YAML config file \"%s\": %v\n.", yamlPath, err)
// 		log.Printf("Using default configuration file instead.")
// 		return GetDefaultConfiguration()
// 	}

// 	var conf Configuration
// 	err = yaml.Unmarshal(yamlFile, &conf)
// 	if err != nil {
// 		log.Fatalf("Unmarshal: %v", err)
// 	}

// 	return &conf
// }