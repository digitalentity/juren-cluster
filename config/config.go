package config

import (
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Config represents the configuration for the ipfs-go-storage application
type Config struct {
	// Default config file location
	configFile string

	// Publisher settings define the IPNS address which will be used to manage the VFS
	Discovery struct {
		UseMDNS bool `json:"mdns"`
	} `json:"discovery"`

	DataStore struct {
		MetadataPath string `json:"metadata"`
		BlockPath    string `json:"blocks"`
	} `json:"datastore"`
}

// NewConfig generates a new configuration with default settings
func NewEmptyConfig(configFile string) *Config {
	cfg := &Config{}

	cfg.configFile = configFile

	cfg.Discovery.UseMDNS = false

	cfg.DataStore.MetadataPath = "/tmp/juren-cluster/metadata"
	cfg.DataStore.BlockPath = "/tmp/juren-cluster/blocks"

	return cfg
}

func NewConfigFromFile(configFile string) (*Config, error) {
	cfg := NewEmptyConfig(configFile)
	if err := cfg.Load(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save saves the configuration to a file
func (c *Config) Save() error {
	log.Infof("Saving config to %s", c.configFile)

	// We'll marshall our structure to JSON and write it into a file
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.configFile, data, 0644)
}

func (c *Config) Load() error {
	log.Infof("Loading config from %s", c.configFile)
	data, err := os.ReadFile(c.configFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	return nil
}
