package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	// HTTP server
	ListenAddr string `json:"listen_addr"` // e.g. ":8080"

	// SQLite database file path
	DBPath string `json:"db_path"` // e.g. "./timeslot.db"

	// Admin UI credentials
	AdminUser     string `json:"admin_user"`
	AdminPassword string `json:"admin_password"`

	// Slot settings
	SlotDurationMin int `json:"slot_duration_min"` // default 30
	BufferBeforeMin int `json:"buffer_before_min"` // min minutes ahead of now to show slots
	MaxDaysAhead    int `json:"max_days_ahead"`    // how many days ahead to show slots

	// Timezone for availability rules (IANA tz name, e.g. "Asia/Shanghai")
	Timezone string `json:"timezone"`
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		ListenAddr:      ":8080",
		DBPath:          "./timeslot.db",
		AdminUser:       "admin",
		AdminPassword:   "changeme",
		SlotDurationMin: 30,
		BufferBeforeMin: 0,
		MaxDaysAhead:    30,
		Timezone:        "UTC",
	}
}

// Load reads config from a JSON file, falling back to defaults for missing fields.
func Load(path string) (*Config, error) {
	cfg := Default()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
