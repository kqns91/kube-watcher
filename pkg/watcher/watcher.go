// Package watcher provides Kubernetes resource watching using informers.
package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/kqns91/kube-watcher/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// ContainerInfo represents container information
type ContainerInfo struct {
	Name  string
	Image string
}

// ReplicaInfo represents replica information
type ReplicaInfo struct {
	Desired int32
	Ready   int32
	Current int32
}

// Event represents a Kubernetes resource event
type Event struct {
	Kind      string
	Namespace string
	Name      string
	EventType string
	Timestamp time.Time
	Object    runtime.Object
	Labels    map[string]string

	// Additional information
	Reason      string
	Message     string
	Status      string
	Containers  []ContainerInfo
	Replicas    *ReplicaInfo
	ServiceType string

	// Image change detection
	ImageChanged bool
	OldImages    []ContainerInfo
	NewImages    []ContainerInfo
}

// EventHandler is a function that handles resource events
type EventHandler func(event *Event)

// Watcher watches Kubernetes resources and triggers events
type Watcher struct {
	clientset *kubernetes.Clientset
	config    *config.Config
	handler   EventHandler
	stopCh    chan struct{}
}

// NewWatcher creates a new Watcher instance
func NewWatcher(cfg *config.Config, handler EventHandler) (*Watcher, error) {
	// Try in-cluster config first, fall back to kubeconfig
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		// Try loading from kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		k8sConfig, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Watcher{
		clientset: clientset,
		config:    cfg,
		handler:   handler,
		stopCh:    make(chan struct{}),
	}, nil
}

// Start begins watching configured resources
func (w *Watcher) Start(ctx context.Context) error {
	factory := informers.NewSharedInformerFactoryWithOptions(
		w.clientset,
		time.Second*30,
		informers.WithNamespace(w.config.Namespace),
	)

	// Register informers for each watched resource kind
	for _, kind := range w.config.GetWatchedKinds() {
		if err := w.registerInformer(factory, kind); err != nil {
			return fmt.Errorf("failed to register informer for %s: %w", kind, err)
		}
	}

	// Start all informers
	factory.Start(w.stopCh)

	// Wait for cache sync
	factory.WaitForCacheSync(w.stopCh)

	// Block until context is cancelled
	<-ctx.Done()
	close(w.stopCh)

	return nil
}

// registerInformer registers an informer for a specific resource kind
func (w *Watcher) registerInformer(factory informers.SharedInformerFactory, kind string) error {
	switch kind {
	case "Pod":
		informer := factory.Core().V1().Pods().Informer()
		informer.AddEventHandler(w.createEventHandler("Pod"))
	case "Deployment":
		informer := factory.Apps().V1().Deployments().Informer()
		informer.AddEventHandler(w.createEventHandler("Deployment"))
	case "Service":
		informer := factory.Core().V1().Services().Informer()
		informer.AddEventHandler(w.createEventHandler("Service"))
	case "ConfigMap":
		informer := factory.Core().V1().ConfigMaps().Informer()
		informer.AddEventHandler(w.createEventHandler("ConfigMap"))
	case "Secret":
		informer := factory.Core().V1().Secrets().Informer()
		informer.AddEventHandler(w.createEventHandler("Secret"))
	case "ReplicaSet":
		informer := factory.Apps().V1().ReplicaSets().Informer()
		informer.AddEventHandler(w.createEventHandler("ReplicaSet"))
	case "StatefulSet":
		informer := factory.Apps().V1().StatefulSets().Informer()
		informer.AddEventHandler(w.createEventHandler("StatefulSet"))
	case "DaemonSet":
		informer := factory.Apps().V1().DaemonSets().Informer()
		informer.AddEventHandler(w.createEventHandler("DaemonSet"))
	case "CronJob":
		informer := factory.Batch().V1().CronJobs().Informer()
		informer.AddEventHandler(w.createEventHandler("CronJob"))
	default:
		return fmt.Errorf("unsupported resource kind: %s", kind)
	}

	return nil
}

// createEventHandler creates a ResourceEventHandler for a specific resource kind
func (w *Watcher) createEventHandler(kind string) cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			event := w.convertToEvent(obj, kind, "ADDED")
			if event != nil {
				w.handler(event)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Check for significant change and image change
			imageChanged, oldImages, newImages := w.detectImageChange(oldObj, newObj)
			if !w.hasSignificantChange(oldObj, newObj) {
				return
			}
			event := w.convertToEvent(newObj, kind, "UPDATED")
			if event != nil {
				event.ImageChanged = imageChanged
				event.OldImages = oldImages
				event.NewImages = newImages
				w.handler(event)
			}
		},
		DeleteFunc: func(obj interface{}) {
			event := w.convertToEvent(obj, kind, "DELETED")
			if event != nil {
				w.handler(event)
			}
		},
	}
}

// detectImageChange detects if container images have changed between old and new objects
func (w *Watcher) detectImageChange(oldObj, newObj interface{}) (changed bool, oldImages, newImages []ContainerInfo) {
	getContainers := func(obj interface{}) []ContainerInfo {
		var containers []ContainerInfo
		switch o := obj.(type) {
		case *corev1.Pod:
			for _, c := range o.Spec.Containers {
				containers = append(containers, ContainerInfo{Name: c.Name, Image: c.Image})
			}
		case *appsv1.Deployment:
			for _, c := range o.Spec.Template.Spec.Containers {
				containers = append(containers, ContainerInfo{Name: c.Name, Image: c.Image})
			}
		case *appsv1.StatefulSet:
			for _, c := range o.Spec.Template.Spec.Containers {
				containers = append(containers, ContainerInfo{Name: c.Name, Image: c.Image})
			}
		case *appsv1.DaemonSet:
			for _, c := range o.Spec.Template.Spec.Containers {
				containers = append(containers, ContainerInfo{Name: c.Name, Image: c.Image})
			}
		case *batchv1.CronJob:
			for _, c := range o.Spec.JobTemplate.Spec.Template.Spec.Containers {
				containers = append(containers, ContainerInfo{Name: c.Name, Image: c.Image})
			}
		}
		return containers
	}

	oldImages = getContainers(oldObj)
	newImages = getContainers(newObj)

	// Check if images changed
	if len(oldImages) != len(newImages) {
		return true, oldImages, newImages
	}
	for i := range oldImages {
		if oldImages[i].Image != newImages[i].Image {
			return true, oldImages, newImages
		}
	}
	return false, oldImages, newImages
}

// hasSignificantChange checks if there's a significant change between old and new objects
func (w *Watcher) hasSignificantChange(oldObj, newObj interface{}) bool {
	oldMeta, ok1 := oldObj.(metav1.Object)
	newMeta, ok2 := newObj.(metav1.Object)
	if !ok1 || !ok2 {
		return true // If we can't get metadata, assume there's a change
	}

	// Skip if ResourceVersion is the same (no actual change)
	if oldMeta.GetResourceVersion() == newMeta.GetResourceVersion() {
		return false
	}

	// Check for significant changes based on resource type
	switch oldTyped := oldObj.(type) {
	case *corev1.Pod:
		newTyped := newObj.(*corev1.Pod)
		// Only notify on status phase changes or container image changes
		if oldTyped.Status.Phase != newTyped.Status.Phase {
			return true
		}
		// Check if any container image changed
		if len(oldTyped.Spec.Containers) != len(newTyped.Spec.Containers) {
			return true
		}
		for i := range oldTyped.Spec.Containers {
			if oldTyped.Spec.Containers[i].Image != newTyped.Spec.Containers[i].Image {
				return true
			}
		}
		return false

	case *appsv1.Deployment:
		newTyped := newObj.(*appsv1.Deployment)
		// Notify on replica count changes
		if oldTyped.Spec.Replicas != nil && newTyped.Spec.Replicas != nil &&
			*oldTyped.Spec.Replicas != *newTyped.Spec.Replicas {
			return true
		}
		// Notify on ready replica count changes
		if oldTyped.Status.ReadyReplicas != newTyped.Status.ReadyReplicas {
			return true
		}
		// Notify on container image changes
		if len(oldTyped.Spec.Template.Spec.Containers) != len(newTyped.Spec.Template.Spec.Containers) {
			return true
		}
		for i := range oldTyped.Spec.Template.Spec.Containers {
			if oldTyped.Spec.Template.Spec.Containers[i].Image != newTyped.Spec.Template.Spec.Containers[i].Image {
				return true
			}
		}
		return false

	case *corev1.Service:
		newTyped := newObj.(*corev1.Service)
		// Notify on service type changes
		if oldTyped.Spec.Type != newTyped.Spec.Type {
			return true
		}
		// Notify on port changes
		if len(oldTyped.Spec.Ports) != len(newTyped.Spec.Ports) {
			return true
		}
		return false

	case *appsv1.ReplicaSet:
		newTyped := newObj.(*appsv1.ReplicaSet)
		// Notify on replica count changes
		if oldTyped.Spec.Replicas != nil && newTyped.Spec.Replicas != nil &&
			*oldTyped.Spec.Replicas != *newTyped.Spec.Replicas {
			return true
		}
		if oldTyped.Status.ReadyReplicas != newTyped.Status.ReadyReplicas {
			return true
		}
		return false

	case *appsv1.StatefulSet:
		newTyped := newObj.(*appsv1.StatefulSet)
		// Notify on replica count changes
		if oldTyped.Spec.Replicas != nil && newTyped.Spec.Replicas != nil &&
			*oldTyped.Spec.Replicas != *newTyped.Spec.Replicas {
			return true
		}
		if oldTyped.Status.ReadyReplicas != newTyped.Status.ReadyReplicas {
			return true
		}
		// Check for image changes
		if len(oldTyped.Spec.Template.Spec.Containers) != len(newTyped.Spec.Template.Spec.Containers) {
			return true
		}
		for i := range oldTyped.Spec.Template.Spec.Containers {
			if oldTyped.Spec.Template.Spec.Containers[i].Image != newTyped.Spec.Template.Spec.Containers[i].Image {
				return true
			}
		}
		return false

	case *appsv1.DaemonSet:
		newTyped := newObj.(*appsv1.DaemonSet)
		// Only notify on container image changes
		if len(oldTyped.Spec.Template.Spec.Containers) != len(newTyped.Spec.Template.Spec.Containers) {
			return true
		}
		for i := range oldTyped.Spec.Template.Spec.Containers {
			if oldTyped.Spec.Template.Spec.Containers[i].Image != newTyped.Spec.Template.Spec.Containers[i].Image {
				return true
			}
		}
		return false

	case *batchv1.CronJob:
		newTyped := newObj.(*batchv1.CronJob)
		// Only notify on container image changes in job template
		oldContainers := oldTyped.Spec.JobTemplate.Spec.Template.Spec.Containers
		newContainers := newTyped.Spec.JobTemplate.Spec.Template.Spec.Containers
		if len(oldContainers) != len(newContainers) {
			return true
		}
		for i := range oldContainers {
			if oldContainers[i].Image != newContainers[i].Image {
				return true
			}
		}
		return false

	default:
		// For ConfigMap, Secret, etc., compare ResourceVersion only
		// This reduces noise significantly
		return false
	}
}

// convertToEvent converts a Kubernetes object to an Event
func (w *Watcher) convertToEvent(obj interface{}, kind, eventType string) *Event {
	var meta metav1.Object
	var labels map[string]string
	event := &Event{
		Kind:      kind,
		EventType: eventType,
		Timestamp: time.Now(),
		Object:    obj.(runtime.Object),
	}

	// Extract metadata and additional information based on object type
	switch o := obj.(type) {
	case *corev1.Pod:
		meta = o
		labels = o.Labels
		event.Status = string(o.Status.Phase)
		event.Reason = o.Status.Reason
		event.Message = o.Status.Message
		// Extract container information
		for _, container := range o.Spec.Containers {
			event.Containers = append(event.Containers, ContainerInfo{
				Name:  container.Name,
				Image: container.Image,
			})
		}

	case *appsv1.Deployment:
		meta = o
		labels = o.Labels
		if o.Spec.Replicas != nil {
			event.Replicas = &ReplicaInfo{
				Desired: *o.Spec.Replicas,
				Ready:   o.Status.ReadyReplicas,
				Current: o.Status.Replicas,
			}
		}
		// Extract container information from template
		for _, container := range o.Spec.Template.Spec.Containers {
			event.Containers = append(event.Containers, ContainerInfo{
				Name:  container.Name,
				Image: container.Image,
			})
		}
		// Check deployment status
		for _, cond := range o.Status.Conditions {
			if cond.Type == appsv1.DeploymentProgressing {
				event.Status = string(cond.Status)
				event.Reason = cond.Reason
				event.Message = cond.Message
				break
			}
		}

	case *corev1.Service:
		meta = o
		labels = o.Labels
		event.ServiceType = string(o.Spec.Type)

	case *corev1.ConfigMap:
		meta = o
		labels = o.Labels

	case *corev1.Secret:
		meta = o
		labels = o.Labels

	case *appsv1.ReplicaSet:
		meta = o
		labels = o.Labels
		if o.Spec.Replicas != nil {
			event.Replicas = &ReplicaInfo{
				Desired: *o.Spec.Replicas,
				Ready:   o.Status.ReadyReplicas,
				Current: o.Status.Replicas,
			}
		}

	case *appsv1.StatefulSet:
		meta = o
		labels = o.Labels
		if o.Spec.Replicas != nil {
			event.Replicas = &ReplicaInfo{
				Desired: *o.Spec.Replicas,
				Ready:   o.Status.ReadyReplicas,
				Current: o.Status.Replicas,
			}
		}

	case *appsv1.DaemonSet:
		meta = o
		labels = o.Labels
		// Extract container information from template
		for _, container := range o.Spec.Template.Spec.Containers {
			event.Containers = append(event.Containers, ContainerInfo{
				Name:  container.Name,
				Image: container.Image,
			})
		}

	case *batchv1.CronJob:
		meta = o
		labels = o.Labels
		// Extract container information from job template
		for _, container := range o.Spec.JobTemplate.Spec.Template.Spec.Containers {
			event.Containers = append(event.Containers, ContainerInfo{
				Name:  container.Name,
				Image: container.Image,
			})
		}

	default:
		return nil
	}

	event.Namespace = meta.GetNamespace()
	event.Name = meta.GetName()
	event.Labels = labels

	return event
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.stopCh)
}
