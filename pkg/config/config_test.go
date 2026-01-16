package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	validConfig := `
namespace: production

watches:
  - selector:
      kind: Pod
    triggers:
      - imageChange: true
  - selector:
      kind: Deployment
      labels:
        app: web
    triggers:
      - imageChange: true

notifier:
  slack:
    webhookUrl: "https://hooks.slack.com/services/TEST/WEBHOOK/URL"
    template: "Test template"
`

	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if cfg.Namespace != "production" {
		t.Errorf("Namespace = %v, want production", cfg.Namespace)
	}

	if len(cfg.Watches) != 2 {
		t.Errorf("len(Watches) = %v, want 2", len(cfg.Watches))
	}

	if cfg.Notifier.Slack.WebhookURL != "https://hooks.slack.com/services/TEST/WEBHOOK/URL" {
		t.Errorf("WebhookURL = %v, want https://hooks.slack.com/services/TEST/WEBHOOK/URL", cfg.Notifier.Slack.WebhookURL)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadConfig() error = nil, want error for nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `
namespace: test
watches:
  - selector:
      kind: Pod
  invalid yaml here!!!
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() error = nil, want error for invalid YAML")
	}
}

func TestValidate_MissingNamespace(t *testing.T) {
	cfg := &Config{
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Pod"},
				Triggers: []Trigger{{ImageChange: true}},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for missing namespace")
	}
}

func TestValidate_MissingWatches(t *testing.T) {
	cfg := &Config{
		Namespace: "default",
		Watches:   []WatchConfig{},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for missing watches")
	}
}

func TestValidate_MissingWebhookURL(t *testing.T) {
	cfg := &Config{
		Namespace: "default",
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Pod"},
				Triggers: []Trigger{{ImageChange: true}},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for missing webhook URL")
	}
}

func TestValidate_MissingSelectorKind(t *testing.T) {
	cfg := &Config{
		Namespace: "default",
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: ""},
				Triggers: []Trigger{{ImageChange: true}},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for missing selector kind")
	}
}

func TestValidate_MissingTriggers(t *testing.T) {
	cfg := &Config{
		Namespace: "default",
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Pod"},
				Triggers: []Trigger{},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for missing triggers")
	}
}

func TestValidate_NoEnabledTrigger(t *testing.T) {
	cfg := &Config{
		Namespace: "default",
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Pod"},
				Triggers: []Trigger{{ImageChange: false}},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for no enabled trigger")
	}
}

func TestValidate_DefaultTemplate(t *testing.T) {
	cfg := &Config{
		Namespace: "default",
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Pod"},
				Triggers: []Trigger{{ImageChange: true}},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
				Template:   "",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}

	if cfg.Notifier.Slack.Template == "" {
		t.Error("Template is empty, expected default template to be set")
	}
}

func TestValidate_EnvChangeTrigger(t *testing.T) {
	// envChange: true should be valid
	cfg := &Config{
		Namespace: "default",
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Deployment"},
				Triggers: []Trigger{{EnvChange: true}},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for envChange: true", err)
	}
}

func TestValidate_BothTriggersEnabled(t *testing.T) {
	// Both imageChange and envChange should be valid
	cfg := &Config{
		Namespace: "default",
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Deployment"},
				Triggers: []Trigger{{ImageChange: true, EnvChange: true}},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil for both triggers enabled", err)
	}
}

func TestValidate_NoTriggersEnabled(t *testing.T) {
	// Neither imageChange nor envChange enabled should fail
	cfg := &Config{
		Namespace: "default",
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Deployment"},
				Triggers: []Trigger{{ImageChange: false, EnvChange: false}},
			},
		},
		Notifier: NotifierConfig{
			Slack: SlackConfig{
				WebhookURL: "https://example.com",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error when no triggers enabled")
	}
}

func TestGetWatchForResource(t *testing.T) {
	cfg := &Config{
		Watches: []WatchConfig{
			{
				Selector: WatchSelector{Kind: "Pod"},
				Triggers: []Trigger{{ImageChange: true}},
			},
			{
				Selector: WatchSelector{Kind: "Deployment"},
				Triggers: []Trigger{{ImageChange: true}},
			},
		},
	}

	tests := []struct {
		name     string
		kind     string
		wantNil  bool
		wantKind string
	}{
		{
			name:     "existing watch for Pod",
			kind:     "Pod",
			wantNil:  false,
			wantKind: "Pod",
		},
		{
			name:     "existing watch for Deployment",
			kind:     "Deployment",
			wantNil:  false,
			wantKind: "Deployment",
		},
		{
			name:    "non-existing watch for Service",
			kind:    "Service",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watch := cfg.GetWatchForResource(tt.kind)

			if tt.wantNil {
				if watch != nil {
					t.Errorf("GetWatchForResource() = %v, want nil", watch)
				}
			} else {
				if watch == nil {
					t.Fatal("GetWatchForResource() = nil, want non-nil")
					return
				}
				if watch.Selector.Kind != tt.wantKind {
					t.Errorf("watch.Selector.Kind = %v, want %v", watch.Selector.Kind, tt.wantKind)
				}
			}
		})
	}
}

func TestGetWatchedKinds(t *testing.T) {
	cfg := &Config{
		Watches: []WatchConfig{
			{Selector: WatchSelector{Kind: "Pod"}},
			{Selector: WatchSelector{Kind: "Deployment"}},
			{Selector: WatchSelector{Kind: "Pod"}}, // duplicate
		},
	}

	kinds := cfg.GetWatchedKinds()

	if len(kinds) != 2 {
		t.Errorf("len(GetWatchedKinds()) = %v, want 2", len(kinds))
	}

	kindSet := make(map[string]bool)
	for _, k := range kinds {
		kindSet[k] = true
	}

	if !kindSet["Pod"] {
		t.Error("GetWatchedKinds() should contain Pod")
	}
	if !kindSet["Deployment"] {
		t.Error("GetWatchedKinds() should contain Deployment")
	}
}

func TestLoadConfig_ComplexConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "complex.yaml")

	complexConfig := `
namespace: production

watches:
  - selector:
      kind: Pod
      labels:
        environment: production
        tier: frontend
    triggers:
      - imageChange: true
  - selector:
      kind: Deployment
      name: my-app
    triggers:
      - imageChange: true
  - selector:
      kind: CronJob
    triggers:
      - imageChange: true

notifier:
  slack:
    webhookUrl: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
    template: |
      :kubernetes: *[{{ .Kind }}]* {{ .Namespace }}/{{ .Name }}
      Action: {{ .EventType }}
`

	if err := os.WriteFile(configPath, []byte(complexConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	if len(cfg.Watches) != 3 {
		t.Errorf("len(Watches) = %v, want 3", len(cfg.Watches))
	}

	podWatch := cfg.GetWatchForResource("Pod")
	if podWatch == nil {
		t.Fatal("Pod watch is nil")
		return
	}

	if len(podWatch.Selector.Labels) != 2 {
		t.Errorf("len(PodWatch.Selector.Labels) = %v, want 2", len(podWatch.Selector.Labels))
	}

	if podWatch.Selector.Labels["environment"] != "production" {
		t.Errorf("PodWatch.Selector.Labels[environment] = %v, want production", podWatch.Selector.Labels["environment"])
	}

	deployWatch := cfg.GetWatchForResource("Deployment")
	if deployWatch == nil {
		t.Fatal("Deployment watch is nil")
		return
	}

	if deployWatch.Selector.Name != "my-app" {
		t.Errorf("DeploymentWatch.Selector.Name = %v, want my-app", deployWatch.Selector.Name)
	}
}
