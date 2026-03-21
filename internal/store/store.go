package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"darfin/internal/models"
)

// Store handles persistence of downloads and settings
type Store struct {
	mu        sync.RWMutex
	configDir string
}

// AppData holds all persisted data
type AppData struct {
	Downloads []models.DownloadItem `json:"downloads"`
	Settings  models.Settings       `json:"settings"`
}

// New creates a new Store instance
func New() (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(configDir, "DARFIN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &Store{configDir: dir}, nil
}

// GetConfigDir returns the config directory path
func (s *Store) GetConfigDir() string {
	return s.configDir
}

// LoadDownloads loads saved download items
func (s *Store) LoadDownloads() ([]models.DownloadItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.configDir, "downloads.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.DownloadItem{}, nil
		}
		return nil, err
	}

	var downloads []models.DownloadItem
	if err := json.Unmarshal(data, &downloads); err != nil {
		return []models.DownloadItem{}, nil
	}

	return downloads, nil
}

// SaveDownloads persists download items to disk
func (s *Store) SaveDownloads(downloads []models.DownloadItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(downloads, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.configDir, "downloads.json")
	return os.WriteFile(filePath, data, 0644)
}

// LoadSettings loads application settings
func (s *Store) LoadSettings() (models.Settings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.configDir, "settings.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return models.DefaultSettings(), nil
		}
		return models.Settings{}, err
	}

	var settings models.Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return models.DefaultSettings(), nil
	}

	return settings, nil
}

// SaveSettings persists application settings
func (s *Store) SaveSettings(settings models.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.configDir, "settings.json")
	return os.WriteFile(filePath, data, 0644)
}
