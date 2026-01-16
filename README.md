# kube-watcher

Namespace限定権限で動作する、軽量なKubernetesリソース監視Bot

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/kqns91/kube-watcher)](https://github.com/kqns91/kube-watcher/releases)

## 特徴

| 機能 | 説明 |
|------|------|
| **Namespace限定** | ClusterRole不要、マルチテナント環境でも安全 |
| **image変更検知** | Deployment/StatefulSet/DaemonSet/CronJobのイメージ変更を通知 |
| **env変更検知** | 環境変数・EnvFrom参照の変更を通知 |
| **ホットリロード** | ConfigMap変更を自動検知、Pod再起動不要 |
| **重複排除** | LRUキャッシュによる同一イベントの重複通知防止 |
| **バッチ処理** | 複数イベントをまとめて通知 |

## アーキテクチャ

```
K8s API → Watcher → Filter → Deduplicator → [Batcher] → Formatter → Slack
            │          │          │             │            │
        informer   watches    LRUキャッシュ   時間窓集約   Attachments
```

## クイックスタート

```bash
# Helmリポジトリの追加
helm repo add kube-watcher https://kqns91.github.io/kube-watcher/
helm repo update

# インストール
helm install kube-watcher kube-watcher/kube-watcher \
  --set slack.webhookUrl="https://hooks.slack.com/services/YOUR/WEBHOOK/URL" \
  --set namespace="monitoring" \
  --namespace monitoring \
  --create-namespace
```

## 設定例

```yaml
namespace: "production"

watches:
  # Deploymentのimage/env変更を監視
  - selector:
      kind: Deployment
      labels:
        app: my-app
    triggers:
      - imageChange: true
        envChange: true

  # CronJobのimage変更を監視
  - selector:
      kind: CronJob
    triggers:
      - imageChange: true

notifier:
  slack:
    webhookUrl: "${SLACK_WEBHOOK_URL}"

deduplication:
  enabled: true
  ttlSeconds: 300

batching:
  enabled: false
  windowSeconds: 300
  mode: smart
```

### watches構文

**selector** (監視対象)

| フィールド | 説明 |
|-----------|------|
| `kind` | リソース種類（必須）: Pod, Deployment, StatefulSet, DaemonSet, CronJob |
| `name` | リソース名（オプション） |
| `labels` | ラベルセレクター（オプション） |

**triggers** (通知条件)

| フィールド | 説明 |
|-----------|------|
| `imageChange` | コンテナイメージ変更時に通知 |
| `envChange` | 環境変数変更時に通知 |

### 検知動作

| リソース | imageChange | envChange |
|---------|-------------|-----------|
| Deployment | 新Podが起動時 | spec更新時 |
| StatefulSet | 新Podが起動時 | spec更新時 |
| DaemonSet | 新Podが起動時 | spec更新時 |
| CronJob | JobTemplate更新時 | JobTemplate更新時 |
| Pod | ADDED時 | ADDED時 |

**注意**: ConfigMap/Secretの中身の変更は検知不可（参照名の変更のみ）

## RBAC権限

Namespace限定のRoleのみ必要（ClusterRole不要）:

```yaml
rules:
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "secrets"]
    verbs: ["list", "watch", "get"]
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
    verbs: ["list", "watch", "get"]
  - apiGroups: ["batch"]
    resources: ["cronjobs"]
    verbs: ["list", "watch", "get"]
```

## 開発

```bash
make build          # バイナリビルド
make test           # テスト実行
make lint           # Lint実行
make docker-build   # Dockerイメージビルド
```

## ロードマップ

- [x] image変更検知 (v0.6.0)
- [x] env変更検知 (v0.7.0)
- [ ] Teams/Discord対応
- [ ] Prometheusメトリクス

## ライセンス

MIT License - [LICENSE](LICENSE)

## サポート

[GitHub Issues](https://github.com/kqns91/kube-watcher/issues)
