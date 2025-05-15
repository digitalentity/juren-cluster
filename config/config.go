package config

import (
	"encoding/json"
	"juren/oid"
	"os"

	log "github.com/sirupsen/logrus"
)

// Config represents the configuration for the ipfs-go-storage application
type Config struct {
	// Default config file location
	configFile string

	// Node
	Node struct {
		NodeID *oid.Oid `json:"nodeID"`
	} `json:"node"`

	// Publisher settings define the IPNS address which will be used to manage the VFS
	Network struct {
		RPCListenAddress       string `json:"rpcListenAddress"`
		PubSubMulticastAddress string `json:"pubSubMulticastAddress"`
	} `json:"network"`

	DataStore struct {
		BlockStorePath string `json:"metadata"`
		BlockIndexPath string `json:"blocks"`
		NodeIndexPath  string `json:"peers"`
	} `json:"datastore"`
}

// NewConfig generates a new configuration with default settings
func NewEmptyConfig(configFile string) *Config {
	cfg := &Config{}

	cfg.configFile = configFile

	// Generate a random Node ID
	nodeid, err := oid.Random(oid.OidTypeNode)
	if err != nil {
		log.Fatalf("Failed to generate Node ID: %v", err)
	}
	cfg.Node.NodeID = nodeid

	cfg.Network.RPCListenAddress = "0.0.0.0:5001"
	cfg.Network.PubSubMulticastAddress = "224.0.0.1:5002"

	cfg.DataStore.BlockStorePath = "/tmp/juren-cluster/blocks"
	cfg.DataStore.BlockIndexPath = "/tmp/juren-cluster/metadata"
	cfg.DataStore.NodeIndexPath = "/tmp/juren-cluster/peers"

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
