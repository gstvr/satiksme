package main

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Boards []BoardConfig `yaml:"boards"`
}

func (cfg Config) BoardBySlug(slug string) (BoardConfig, bool) {
	for _, b := range cfg.Boards {
		if b.Slug == slug {
			return b, true
		}
	}
	return BoardConfig{}, false
}

type StopConfig struct {
	Name       string   `yaml:"name"`
	StopIDs    []string `yaml:"stop_ids"`
	LineFilter []string `yaml:"line_filter"`
}

type BoardConfig struct {
	Name  string       `yaml:"name"`
	Slug  string       `yaml:"slug"`
	Stops []StopConfig `yaml:"stops"`
}

func (b BoardConfig) AllStopIDs() []string {
	var ids []string
	for _, stop := range b.Stops {
		for _, id := range stop.StopIDs {
			ids = append(ids, id)
		}
	}
	return ids
}

func loadConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	body, err := io.ReadAll(f)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err = yaml.Unmarshal(body, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	if len(cfg.Boards) == 0 {
		return Config{}, fmt.Errorf("no boards defined in config file")
	}
	return cfg, nil
}
