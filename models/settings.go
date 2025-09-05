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
	setting, err := Settings.Find("WHERE Key = ?", key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// SetSetting creates or updates a setting
func SetSetting(key, value, settingType string) error {
	settings, err := Settings.Search("WHERE Key = ? LIMIT 1", key)
	if err != nil {
		return err
	}
	
	if len(settings) > 0 {
		// Update existing
		settings[0].Value = value
		return Settings.Update(settings[0])
	}
	
	// Create new
	_, err = Settings.Insert(&Setting{
		Key:   key,
		Value: value,
		Type:  settingType,
	})
	return err
}