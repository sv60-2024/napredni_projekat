package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	MemtableSize   int `json:"memtable_size"`
	WALSegmentSize int `json:"wal_segment_size"`
	BlockSize      int `json:"block_size"`
	BlockCacheSize int `json:"block_cache_size"`
}

func DefaultConfig() Config {
	return Config{
		MemtableSize:   10,
		WALSegmentSize: 10,
		BlockSize:      4096,
		BlockCacheSize: 20,
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("greska pri citanju konfiguracije: %w", err)
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("greska pri parsiranju konfiguracije: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.MemtableSize <= 0 {
		return fmt.Errorf("memtable_size mora biti veci od 0")
	}

	if cfg.WALSegmentSize <= 0 {
		return fmt.Errorf("wal_segment_size mora biti veci od 0")
	}

	if cfg.BlockCacheSize <= 0 {
		return fmt.Errorf("block_cache_size mora biti veci od 0")
	}

	if cfg.BlockSize != 4096 &&
		cfg.BlockSize != 8192 &&
		cfg.BlockSize != 16384 {
		return fmt.Errorf("block_size mora biti 4096, 8192 ili 16384")
	}

	return nil
}
