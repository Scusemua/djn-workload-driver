package config

const (
	OptionName    = "name"
	OptionDefault = "default"
	OptionDesc    = "description"
)

type Configuration struct {
	SpoofCluster        bool   `yaml:"spoof-gateway" description:"If true, use the fake cluster."`
	KernelQueryInterval string `yaml:"kernel-query-interval" default:"5s" description:"How frequently to query the Cluster for updated kernel information."`
	NodeQueryInterval   string `yaml:"node-query-interval" default:"10s" description:"How frequently to query the Cluster for updated Kubernetes node information."`
}

func GetConfiguration() *Configuration {
	return &Configuration{
		SpoofCluster:        true,
		KernelQueryInterval: "5s",
		NodeQueryInterval:   "10s",
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
