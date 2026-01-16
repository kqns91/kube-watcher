package filter

import (
	"testing"

	"github.com/kqns91/kube-watcher/pkg/config"
	"github.com/kqns91/kube-watcher/pkg/watcher"
)

func TestFilter_ShouldProcess(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		event  *watcher.Event
		want   bool
	}{
		{
			name: "Pod ADDED event with imageChange trigger",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Pod"},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Pod", Name: "test-pod", EventType: "ADDED"},
			want:  true,
		},
		{
			name: "Pod UPDATED event should not trigger (not ADDED)",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Pod"},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Pod", Name: "test-pod", EventType: "UPDATED"},
			want:  false,
		},
		{
			name: "no watch configured for kind",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Deployment"},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Pod", Name: "test-pod", EventType: "ADDED"},
			want:  false,
		},
		{
			name: "matching watch with label selector",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{
							Kind:   "Pod",
							Labels: map[string]string{"app": "web"},
						},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{
				Kind:      "Pod",
				Name:      "test-pod",
				EventType: "ADDED",
				Labels:    map[string]string{"app": "web", "env": "prod"},
			},
			want: true,
		},
		{
			name: "non-matching label selector",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{
							Kind:   "Pod",
							Labels: map[string]string{"app": "api"},
						},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{
				Kind:      "Pod",
				Name:      "test-pod",
				EventType: "ADDED",
				Labels:    map[string]string{"app": "web"},
			},
			want: false,
		},
		{
			name: "Deployment UPDATED with image change",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{
							Kind: "Deployment",
							Name: "my-app",
						},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", ImageChanged: true},
			want:  true,
		},
		{
			name: "Deployment UPDATED without image change",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{
							Kind: "Deployment",
							Name: "my-app",
						},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", ImageChanged: false},
			want:  false,
		},
		{
			name: "CronJob UPDATED with image change",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "CronJob"},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "CronJob", Name: "my-cronjob", EventType: "UPDATED", ImageChanged: true},
			want:  true,
		},
		{
			name: "CronJob UPDATED without image change (regular job run)",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "CronJob"},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "CronJob", Name: "my-cronjob", EventType: "UPDATED", ImageChanged: false},
			want:  false,
		},
		{
			name: "non-matching name selector",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{
							Kind: "Deployment",
							Name: "my-app",
						},
						Triggers: []config.Trigger{{ImageChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "other-app", EventType: "UPDATED", ImageChanged: true},
			want:  false,
		},
		{
			name: "no enabled triggers",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Pod"},
						Triggers: []config.Trigger{{ImageChange: false}},
					},
				},
			},
			event: &watcher.Event{Kind: "Pod", Name: "test-pod", EventType: "ADDED"},
			want:  false,
		},
		// envChange trigger tests
		{
			name: "Pod ADDED event with envChange trigger",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Pod"},
						Triggers: []config.Trigger{{EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Pod", Name: "test-pod", EventType: "ADDED"},
			want:  true,
		},
		{
			name: "Deployment UPDATED with env change",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Deployment"},
						Triggers: []config.Trigger{{EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", EnvChanged: true},
			want:  true,
		},
		{
			name: "Deployment UPDATED without env change",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Deployment"},
						Triggers: []config.Trigger{{EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", EnvChanged: false},
			want:  false,
		},
		{
			name: "StatefulSet UPDATED with env change",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "StatefulSet"},
						Triggers: []config.Trigger{{EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "StatefulSet", Name: "my-sts", EventType: "UPDATED", EnvChanged: true},
			want:  true,
		},
		{
			name: "DaemonSet UPDATED with env change",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "DaemonSet"},
						Triggers: []config.Trigger{{EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "DaemonSet", Name: "my-ds", EventType: "UPDATED", EnvChanged: true},
			want:  true,
		},
		{
			name: "CronJob UPDATED with env change",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "CronJob"},
						Triggers: []config.Trigger{{EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "CronJob", Name: "my-cj", EventType: "UPDATED", EnvChanged: true},
			want:  true,
		},
		{
			name: "both imageChange and envChange triggers - image change detected",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Deployment"},
						Triggers: []config.Trigger{{ImageChange: true, EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", ImageChanged: true, EnvChanged: false},
			want:  true,
		},
		{
			name: "both imageChange and envChange triggers - env change detected",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Deployment"},
						Triggers: []config.Trigger{{ImageChange: true, EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", ImageChanged: false, EnvChanged: true},
			want:  true,
		},
		{
			name: "both imageChange and envChange triggers - both changed",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Deployment"},
						Triggers: []config.Trigger{{ImageChange: true, EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", ImageChanged: true, EnvChanged: true},
			want:  true,
		},
		{
			name: "both imageChange and envChange triggers - neither changed",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Deployment"},
						Triggers: []config.Trigger{{ImageChange: true, EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", ImageChanged: false, EnvChanged: false},
			want:  false,
		},
		{
			name: "envChange only trigger - image change should not trigger",
			config: &config.Config{
				Watches: []config.WatchConfig{
					{
						Selector: config.WatchSelector{Kind: "Deployment"},
						Triggers: []config.Trigger{{EnvChange: true}},
					},
				},
			},
			event: &watcher.Event{Kind: "Deployment", Name: "my-app", EventType: "UPDATED", ImageChanged: true, EnvChanged: false},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFilter(tt.config)
			got := f.ShouldProcess(tt.event)
			if got != tt.want {
				t.Errorf("ShouldProcess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_MatchesLabels(t *testing.T) {
	f := NewFilter(&config.Config{})

	tests := []struct {
		name           string
		eventLabels    map[string]string
		requiredLabels map[string]string
		want           bool
	}{
		{
			name:           "all labels match",
			eventLabels:    map[string]string{"app": "web", "env": "prod"},
			requiredLabels: map[string]string{"app": "web"},
			want:           true,
		},
		{
			name:           "multiple labels match",
			eventLabels:    map[string]string{"app": "web", "env": "prod", "tier": "frontend"},
			requiredLabels: map[string]string{"app": "web", "env": "prod"},
			want:           true,
		},
		{
			name:           "label missing",
			eventLabels:    map[string]string{"env": "prod"},
			requiredLabels: map[string]string{"app": "web"},
			want:           false,
		},
		{
			name:           "label value mismatch",
			eventLabels:    map[string]string{"app": "api"},
			requiredLabels: map[string]string{"app": "web"},
			want:           false,
		},
		{
			name:           "empty required labels",
			eventLabels:    map[string]string{"app": "web"},
			requiredLabels: map[string]string{},
			want:           true,
		},
		{
			name:           "nil event labels",
			eventLabels:    nil,
			requiredLabels: map[string]string{"app": "web"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.matchesLabels(tt.eventLabels, tt.requiredLabels)
			if got != tt.want {
				t.Errorf("matchesLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
