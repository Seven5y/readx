// Package persistence handles reading and writing reading progress
// to ~/.config/readx/config.json.
package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"readx/internal/domain"
)

// UserSettings holds user-configurable preferences persisted to disk.
type UserSettings struct {
	BgColor bool `json:"bg_color"` // false = terminal default bg, true = dark background
}

// Config holds reading progress for all books, keyed by book path,
// the library shelf, and user settings.
type Config struct {
	Progress map[string]domain.ReadingProgress `json:"progress"`
	Library  []domain.LibraryEntry             `json:"library"`
	Settings UserSettings                      `json:"settings"`
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

// AddBook adds or updates a book entry in the library and persists to disk.
func AddBook(cfg *Config, path string, book *domain.Book) error {
	entry := domain.LibraryEntry{
		Path:     path,
		Title:    book.Title,
		Author:   book.Author,
		Format:   book.Format,
		LastRead: time.Now(),
	}

	// Merge with existing if present.
	found := false
	for i, e := range cfg.Library {
		if e.Path == path {
			entry.Progress = e.Progress
			entry.LastPage = e.LastPage
			cfg.Library[i] = entry
			found = true
			break
		}
	}
	if !found {
		cfg.Library = append(cfg.Library, entry)
	}

	return writeConfig(cfg)
}

// UpdateBookProgress updates the reading progress for a library book entry
// and persists to disk.
func UpdateBookProgress(cfg *Config, path string, progress int, curPage int) error {
	for i, e := range cfg.Library {
		if e.Path == path {
			cfg.Library[i].Progress = progress
			cfg.Library[i].LastPage = curPage
			cfg.Library[i].LastRead = time.Now()
			return writeConfig(cfg)
		}
	}
	return nil
}

// RemoveBook deletes a book entry from the library (does not delete the file).
func RemoveBook(cfg *Config, path string) error {
	for i, e := range cfg.Library {
		if e.Path == path {
			cfg.Library = append(cfg.Library[:i], cfg.Library[i+1:]...)
			break
		}
	}
	delete(cfg.Progress, path)
	return writeConfig(cfg)
}

// ListBooks returns all library entries sorted by last-read time (newest first).
func ListBooks(cfg *Config) []domain.LibraryEntry {
	sorted := make([]domain.LibraryEntry, len(cfg.Library))
	copy(sorted, cfg.Library)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LastRead.After(sorted[j].LastRead)
	})
	return sorted
}

// writeConfig serializes and writes the config to disk.
func writeConfig(cfg *Config) error {
	configFilePath, err := configPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(configFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configFilePath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// SaveSettings persists user settings to disk.
func SaveSettings(cfg *Config, s UserSettings) error {
	cfg.Settings = s
	return writeConfig(cfg)
}
