package config

type MascConfig struct {
	Network NetworkConfig `yaml:"network"`
}

type NetworkConfig struct {
	Address string `yaml:"address"`
}
