package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type BackendsConfig struct {
	OVN struct {
		Enabled   bool   `yaml:"enabled"`
		NBCTLPath string `yaml:"nbctl_path"`
		SBCTLPath string `yaml:"sbctl_path"`
		Bridge    string `yaml:"bridge"`
	} `yaml:"ovn"`

	FRR struct {
		Enabled       bool   `yaml:"enabled"`
		ConfigPath    string `yaml:"config_path"`
		ReloadCommand string `yaml:"reload_command"`
	} `yaml:"frr"`

	Dnsmasq struct {
		Enabled   bool   `yaml:"enabled"`
		ConfigDir string `yaml:"config_dir"`
	} `yaml:"dnsmasq"`

	Datacenter struct {
		Enabled   bool   `yaml:"enabled"`
		Mode      string `yaml:"mode"`
		VLANRange string `yaml:"vlan_range"`
		VNIRange  string `yaml:"vni_range"`
	} `yaml:"datacenter"`

	Kubernetes struct {
		Enabled    bool   `yaml:"enabled"`
		Kubeconfig string `yaml:"kubeconfig"`
		CNIPlugin  string `yaml:"cni_plugin"`
	} `yaml:"kubernetes"`

	Neutron struct {
		Enabled  bool   `yaml:"enabled"`
		Endpoint string `yaml:"endpoint"`
		Token    string `yaml:"token"`
	} `yaml:"neutron"`

	ATCloud struct {
		Enabled  bool   `yaml:"enabled"`
		Endpoint string `yaml:"endpoint"`
		Token    string `yaml:"token"`
	} `yaml:"atcloud"`

	Generic struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"generic"`
}

func LoadBackends(path string) (*BackendsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg BackendsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
