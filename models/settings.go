package models

import (
	"github.com/The-Skyscape/devtools/pkg/application"
)

// Setting represents an application setting
type Setting struct {
	application.Model
	Key   string
	Value string
	Type  string // ssh_key, git_config, preference, etc
}

// Table returns the database table name
func (*Setting) Table() string {
	return "settings"
}

// GetSetting retrieves a setting by key
func GetSetting(key string) (string, error) {
	settings, err := Settings.Search("WHERE Key = ?", key)
	if err != nil || len(settings) == 0 {
		return "", err
	}
	return settings[0].Value, nil
}

// SetSetting creates or updates a setting
func SetSetting(key, value, settingType string) error {
	settings, _ := Settings.Search("WHERE Key = ?", key)
	
	if len(settings) > 0 {
		// Update existing
		settings[0].Value = value
		return Settings.Update(settings[0])
	}
	
	// Create new
	_, err := Settings.Insert(&Setting{
		Key:   key,
		Value: value,
		Type:  settingType,
	})
	return err
}