package ide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const settingsFileName = "ide-settings.json"

func loadPersistedSettings(defaults Settings) Settings {
	path, err := settingsPath()
	if err != nil {
		return defaults
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaults
	}

	var persisted Settings
	if err := json.Unmarshal(data, &persisted); err != nil {
		return defaults
	}

	return mergeSettings(defaults, persisted)
}

func savePersistedSettings(settings Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settingsForDisk(settings), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func settingsPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("OLLAMA_IDE_SETTINGS")); path != "" {
		return filepath.Abs(path)
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Ali", settingsFileName), nil
}

func settingsForDisk(settings Settings) Settings {
	settings.AI.CloudAPIKey = ""
	return settings
}

func settingsForResponse(settings Settings) Settings {
	settings.AI.CloudAPIKey = ""
	return settings
}

func mergeSettings(base Settings, next Settings) Settings {
	if next.Theme.Mode != "" {
		base.Theme.Mode = next.Theme.Mode
	}
	if len(next.Theme.Colors) > 0 {
		base.Theme.Colors = next.Theme.Colors
	}
	if next.AI.Provider != "" {
		base.AI.Provider = next.AI.Provider
	}
	if next.AI.Model != "" {
		base.AI.Model = next.AI.Model
	}
	if next.AI.CloudBaseURL != "" {
		base.AI.CloudBaseURL = next.AI.CloudBaseURL
	}
	if next.AI.CloudAPIKey != "" {
		base.AI.CloudAPIKey = next.AI.CloudAPIKey
	}
	if next.AI.Temperature > 0 {
		base.AI.Temperature = next.AI.Temperature
	}
	if next.AI.MaxTokens > 0 {
		base.AI.MaxTokens = next.AI.MaxTokens
	}
	return base
}
