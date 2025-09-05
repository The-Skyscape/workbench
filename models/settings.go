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
	setting, err := Settings.Find("WHERE Key = ?", key)
	
	if err == nil && setting != nil {
		// Update existing
		setting.Value = value
		return Settings.Update(setting)
	}
	
	// Create new
	_, err = Settings.Insert(&Setting{
		Key:   key,
		Value: value,
		Type:  settingType,
	})
	return err
}