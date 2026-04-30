// Package persistence handles reading and writing reading progress
// to ~/.config/readx/config.json.
package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"readx/internal/domain"
)

// Config holds reading progress for all books, keyed by book path.
type Config struct {
	Progress map[string]domain.ReadingProgress `json:"progress"`
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "readx", "config.json"), nil
}

// LoadConfig reads the progress config from disk. If the file does not
// exist, it returns a fresh empty config.
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{Progress: make(map[string]domain.ReadingProgress)}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config file corrupted, starting fresh: %v\n", err)
		return &Config{Progress: make(map[string]domain.ReadingProgress)}, nil
	}

	return cfg, nil
}

// SaveProgress persists reading progress for a single book.
func SaveProgress(cfg *Config, bookPath string, progress domain.ReadingProgress) error {
	configFilePath, err := configPath()
	if err != nil {
		return err
	}

	// Ensure the config directory exists.
	dir := filepath.Dir(configFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	progress.Timestamp = time.Now()
	progress.BookPath = bookPath

	if cfg.Progress == nil {
		cfg.Progress = make(map[string]domain.ReadingProgress)
	}
	cfg.Progress[bookPath] = progress

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configFilePath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// GetProgress retrieves saved progress for a specific book.
// Returns nil if no progress is saved.
func GetProgress(cfg *Config, bookPath string) *domain.ReadingProgress {
	if cfg == nil || cfg.Progress == nil {
		return nil
	}
	prog, ok := cfg.Progress[bookPath]
	if !ok {
		return nil
	}
	return &prog
}
