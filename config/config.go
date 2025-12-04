package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	path            string
	allowDuplicates bool
	directories     map[string]string
}

func NewConfig() Config {
	return Config{
		path:            "./config.json",
		allowDuplicates: false,
		directories:     make(map[string]string),
	}
}

func (c Config) AddDirectory(source, destination string) Config {
	c.directories[source] = destination
	return c
}

func (c Config) AllowDuplicates() Config {
	c.allowDuplicates = true
	return c
}

func (c Config) DisallowDuplicates() Config {
	c.allowDuplicates = false
	return c
}

func (c Config) IsAllowDuplicates() bool {
	return c.allowDuplicates
}

func (c Config) WriteToFile() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		// handle error
	}
	err = os.WriteFile(c.path, data, 0644)
	if err != nil {
		// handle error
	}
	return nil
}

func ReadConfigToFile() (Config, error) {
	c := NewConfig()
	data, err := os.ReadFile(c.path)
	if err != nil {
		// handle error
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		// handle error
	}
	return c, nil
}
