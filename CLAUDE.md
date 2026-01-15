# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

kube-watcher is a lightweight Kubernetes resource monitoring bot that works with namespace-limited permissions only (no ClusterRole required). It watches Kubernetes resources for **image changes** and sends notifications to Slack.

## Core Feature (v0.6.0)

**Primary Purpose**: Notify when container images change in watched workloads.

| Resource | Detection Method |
|----------|-----------------|
| Pod | ADDED event (new Pod started with new image) |
| Deployment | UPDATED with image change |
| StatefulSet | UPDATED with image change |
| DaemonSet | UPDATED with image change |
| CronJob | UPDATED with image change (not on regular job runs) |

### Configuration Format

```yaml
watches:
  - selector:
      kind: Deployment
      labels:
        app: my-app
    triggers:
      - imageChange: true
```

## Build & Development Commands

```bash
make build          # Build binary to ./kube-watcher
make run            # Run locally (requires kubeconfig)
make test           # Run all tests
make lint           # Run golangci-lint
make lint-fix       # Auto-fix lint issues
make fmt            # Format code
make deps           # Download and tidy dependencies
make docker-build   # Build Docker image
```

## Architecture

Event processing pipeline:

```
Watcher (K8s informers) → Filter (watches) → Deduplicator (LRU) → Batcher (optional) → Formatter → Notifier (Slack)
```

### Key Components (pkg/)

- **watcher/**: Kubernetes informer-based resource watching with image change detection
- **filter/**: Event filtering based on watches configuration (kind, name, labels, triggers)
- **dedup/**: LRU cache-based duplicate event suppression
- **batcher/**: Time-window event batching (detailed/summary/smart modes)
- **formatter/**: Go template-based Slack message formatting
- **notifier/**: Slack webhook delivery
- **reload/**: fsnotify-based ConfigMap hot-reload
- **config/**: YAML configuration parsing

### Core Type

`watcher.Event` carries all event data through the pipeline: Kind, Namespace, Name, EventType, Labels, ImageChanged flag, OldImages, NewImages, and extracted metadata.

### Concurrency Model

- Background goroutines: dedup cleanup, config watching, batcher timer
- RWMutex protection on shared state for thread-safe hot-reload
- Shared informer factory to avoid duplicate K8s API watches

## Code Style

Uses golangci-lint with: errcheck, govet, staticcheck, misspell, revive, gosec, bodyclose. Key revive rules enforce context-as-argument, error-return, var-naming conventions.

## Supported Resources

Pod, Deployment, StatefulSet, DaemonSet, CronJob
