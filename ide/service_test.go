package ide

import (
	"path/filepath"
	"testing"
)

func TestSettingsResponseMasksCloudAPIKeyButAgentCanUseRawSettings(t *testing.T) {
	t.Setenv("OLLAMA_IDE_SETTINGS", filepath.Join(t.TempDir(), "settings.json"))

	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	service.UpdateSettings(Settings{
		AI: AISettings{
			Provider:    "cloud",
			Model:       "gpt-test",
			CloudAPIKey: "secret-token",
		},
	})

	if got := service.Settings().AI.CloudAPIKey; got != "" {
		t.Fatalf("response API key = %q, want masked", got)
	}
	if got := service.settingsSnapshot(false).AI.CloudAPIKey; got != "secret-token" {
		t.Fatalf("raw API key = %q, want secret-token", got)
	}
}

func TestHealthReportsIDECapabilities(t *testing.T) {
	t.Setenv("OLLAMA_IDE_SETTINGS", filepath.Join(t.TempDir(), "settings.json"))

	service, err := NewService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	health := service.Health()
	if health.Status != "ok" {
		t.Fatalf("status = %q, want ok", health.Status)
	}
	if len(health.Features) == 0 {
		t.Fatal("expected features in health report")
	}
	if health.Limits.MaxFileSize <= 0 {
		t.Fatal("expected positive max file size")
	}
}
