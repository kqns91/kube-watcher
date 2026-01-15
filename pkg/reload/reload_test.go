package reload

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kqns91/kube-watcher/pkg/config"
)

func TestNewConfigWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
namespace: default
watches:
  - selector:
      kind: Pod
    triggers:
      - imageChange: true
notifier:
  slack:
    webhookUrl: "https://example.com/webhook"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher() error = %v", err)
	}
	defer watcher.Stop()

	if watcher == nil {
		t.Fatal("NewConfigWatcher() returned nil")
	}

	if watcher.configPath != configPath {
		t.Errorf("Expected configPath %q, got %q", configPath, watcher.configPath)
	}
}

func TestConfigWatcher_AddCallback(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
namespace: default
watches:
  - selector:
      kind: Pod
    triggers:
      - imageChange: true
notifier:
  slack:
    webhookUrl: "https://example.com/webhook"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher() error = %v", err)
	}
	defer watcher.Stop()

	watcher.AddCallback(func(cfg *config.Config) error {
		return nil
	})

	if len(watcher.callbacks) != 1 {
		t.Errorf("Expected 1 callback, got %d", len(watcher.callbacks))
	}
}

func TestConfigWatcher_Reload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `
namespace: default
watches:
  - selector:
      kind: Pod
    triggers:
      - imageChange: true
notifier:
  slack:
    webhookUrl: "https://example.com/webhook"
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher() error = %v", err)
	}
	defer watcher.Stop()

	callbackCalled := make(chan bool, 1)
	var reloadedNamespace string

	watcher.AddCallback(func(cfg *config.Config) error {
		reloadedNamespace = cfg.Namespace
		callbackCalled <- true
		return nil
	})

	watcher.Start()

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Update config file
	updatedConfig := `
namespace: production
watches:
  - selector:
      kind: Pod
    triggers:
      - imageChange: true
  - selector:
      kind: Deployment
    triggers:
      - imageChange: true
notifier:
  slack:
    webhookUrl: "https://example.com/webhook"
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Wait for callback to be called
	select {
	case <-callbackCalled:
		if reloadedNamespace != "production" {
			t.Errorf("Expected namespace 'production', got %q", reloadedNamespace)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Callback was not called within timeout")
	}
}

func TestConfigWatcher_MultipleCallbacks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
namespace: default
watches:
  - selector:
      kind: Pod
    triggers:
      - imageChange: true
notifier:
  slack:
    webhookUrl: "https://example.com/webhook"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher() error = %v", err)
	}
	defer watcher.Stop()

	callback1Called := make(chan bool, 1)
	callback2Called := make(chan bool, 1)

	watcher.AddCallback(func(cfg *config.Config) error {
		callback1Called <- true
		return nil
	})

	watcher.AddCallback(func(cfg *config.Config) error {
		callback2Called <- true
		return nil
	})

	watcher.Start()
	time.Sleep(100 * time.Millisecond)

	// Update config
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Wait for both callbacks
	timeout := time.After(2 * time.Second)
	cb1Done := false
	cb2Done := false

	for !cb1Done || !cb2Done {
		select {
		case <-callback1Called:
			cb1Done = true
		case <-callback2Called:
			cb2Done = true
		case <-timeout:
			if !cb1Done || !cb2Done {
				t.Fatal("Not all callbacks were called within timeout")
			}
		}
	}
}
