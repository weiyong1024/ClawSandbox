package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/weiyong1024/clawsandbox/internal/version"
)

const (
	DefaultImageName    = "ghcr.io/weiyong1024/clawsandbox"
	DefaultNoVNCBase    = 6901
	DefaultGatewayBase  = 18789
	DefaultMemoryLimit  = "4g"
	DefaultCPULimit     = 2.0
	DefaultNamingPrefix = "claw"

	NetworkName  = "clawsandbox-net"
	LabelManaged = "clawsandbox.managed"
)

type Config struct {
	Image     ImageConfig    `yaml:"image"`
	Ports     PortsConfig    `yaml:"ports"`
	Resources ResourceConfig `yaml:"resources"`
	Naming    NamingConfig   `yaml:"naming"`
}

type ImageConfig struct {
	Name string `yaml:"name"`
	Tag  string `yaml:"tag"`
}

type PortsConfig struct {
	NoVNCBase   int `yaml:"novnc_start"`
	GatewayBase int `yaml:"gateway_start"`
}

type ResourceConfig struct {
	MemoryLimit string  `yaml:"memory_limit"`
	CPULimit    float64 `yaml:"cpu_limit"`
}

type NamingConfig struct {
	Prefix string `yaml:"prefix"`
}

func (c *Config) ImageRef() string {
	return fmt.Sprintf("%s:%s", c.Image.Name, c.Image.Tag)
}

func DefaultConfig() *Config {
	return &Config{
		Image: ImageConfig{Name: DefaultImageName, Tag: version.ImageTag()},
		Ports: PortsConfig{NoVNCBase: DefaultNoVNCBase, GatewayBase: DefaultGatewayBase},
		Resources: ResourceConfig{
			MemoryLimit: DefaultMemoryLimit,
			CPULimit:    DefaultCPULimit,
		},
		Naming: NamingConfig{Prefix: DefaultNamingPrefix},
	}
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".clawsandbox"), nil
}

func EnsureDataDir() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating data dir: %w", err)
	}
	return dir, nil
}

func configPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}
