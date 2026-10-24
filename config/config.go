// Package config stores per-server (or per-user in DMs) bot settings in a JSON file.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/Skyfield1888/virtualflightwatcher/i18n"
)

const (
	NetworkVatsim = "VATSIM"
	NetworkIvao   = "IVAO"

	UnitsAviation = "aviation" // ft, kts, inHg + hPa, nm
	UnitsMetric   = "metric"   // m, km/h, hPa, km
	UnitsImperial = "imperial" // ft, mph, inHg, mi
)

type Settings struct {
	Language         i18n.Lang `json:"language"`
	Ephemeral        bool      `json:"ephemeral"`
	Units            string    `json:"units"`
	AllowedChannelID string    `json:"allowed_channel_id"`
}

func Defaults() Settings {
	return Settings{
		Language: i18n.Default,
		Units:    UnitsAviation,
	}
}

var (
	mu   sync.RWMutex
	path string
	data = map[string]Settings{}
)

// Load reads the settings file (a missing file is fine, it will be created on first save).
func Load(p string) error {
	mu.Lock()
	defer mu.Unlock()
	path = p

	raw, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &data)
}

// Get never fails: unknown keys and empty fields fall back to defaults.
func Get(key string) Settings {
	mu.RLock()
	defer mu.RUnlock()
	return withDefaults(data[key])
}

// Update applies fn to the current settings and persists the result.
func Update(key string, fn func(*Settings)) error {
	mu.Lock()
	defer mu.Unlock()
	s := withDefaults(data[key])
	fn(&s)
	data[key] = s
	return save()
}

// Reset removes a key so it goes back to defaults.
func Reset(key string) error {
	mu.Lock()
	defer mu.Unlock()
	delete(data, key)
	return save()
}

func withDefaults(s Settings) Settings {
	d := Defaults()
	if !i18n.Valid(s.Language) {
		s.Language = d.Language
	}
	switch s.Units {
	case UnitsAviation, UnitsMetric, UnitsImperial:
	default:
		s.Units = d.Units
	}
	return s
}

// save writes atomically (temp file + rename). Caller must hold the write lock.
func save() error {
	if path == "" {
		return errors.New("config: Load must be called before saving")
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
