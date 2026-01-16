// Package filter provides event filtering functionality based on configured watches.
package filter

import (
	"github.com/kqns91/kube-watcher/pkg/config"
	"github.com/kqns91/kube-watcher/pkg/watcher"
)

// Filter checks if an event should be processed based on configured watches
type Filter struct {
	config *config.Config
}

// NewFilter creates a new Filter instance
func NewFilter(cfg *config.Config) *Filter {
	return &Filter{
		config: cfg,
	}
}

// ShouldProcess determines if an event should be processed
func (f *Filter) ShouldProcess(event *watcher.Event) bool {
	// Get watch configuration for this resource kind
	watchConfig := f.config.GetWatchForResource(event.Kind)
	if watchConfig == nil {
		// No watch configured for this kind
		return false
	}

	// Check selector match
	if !f.matchesSelector(event, &watchConfig.Selector) {
		return false
	}

	// Check triggers
	for _, trigger := range watchConfig.Triggers {
		if trigger.ImageChange && f.isImageChangeEvent(event) {
			return true
		}
		if trigger.EnvChange && f.isEnvChangeEvent(event) {
			return true
		}
	}

	return false
}

// isImageChangeEvent checks if the event represents an image change
// - For Pod: ADDED event (new Pod with new image started)
// - For Deployment/StatefulSet/DaemonSet: UPDATED with ImageChanged flag
// - For CronJob: UPDATED with ImageChanged flag (not regular job runs)
func (f *Filter) isImageChangeEvent(event *watcher.Event) bool {
	switch event.Kind {
	case "Pod":
		// For Pod, notify when a new Pod is added (indicates new image deployed)
		// This covers Deployment/StatefulSet/DaemonSet rollouts
		return event.EventType == "ADDED"

	case "Deployment", "StatefulSet", "DaemonSet", "CronJob":
		// For workload resources, only notify on actual image changes
		return event.EventType == "UPDATED" && event.ImageChanged
	}

	return false
}

// isEnvChangeEvent checks if the event represents an environment variable change
// - For Pod: ADDED event (new Pod with new env started)
// - For Deployment/StatefulSet/DaemonSet/CronJob: UPDATED with EnvChanged flag
func (f *Filter) isEnvChangeEvent(event *watcher.Event) bool {
	switch event.Kind {
	case "Pod":
		// For Pod, notify when a new Pod is added
		return event.EventType == "ADDED"

	case "Deployment", "StatefulSet", "DaemonSet", "CronJob":
		// For workload resources, only notify on actual env changes
		return event.EventType == "UPDATED" && event.EnvChanged
	}

	return false
}

// matchesSelector checks if the event matches the watch selector
func (f *Filter) matchesSelector(event *watcher.Event, selector *config.WatchSelector) bool {
	// Kind is already matched by GetWatchForResource

	// Check name if specified
	if selector.Name != "" && event.Name != selector.Name {
		return false
	}

	// Check labels if specified
	if len(selector.Labels) > 0 && !f.matchesLabels(event.Labels, selector.Labels) {
		return false
	}

	return true
}

// matchesLabels checks if the event labels match all required labels
func (f *Filter) matchesLabels(eventLabels, requiredLabels map[string]string) bool {
	for key, value := range requiredLabels {
		eventValue, exists := eventLabels[key]
		if !exists || eventValue != value {
			return false
		}
	}
	return true
}
