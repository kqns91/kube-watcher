# kube-watcher

Namespace限定権限で動作する、軽量なKubernetesリソース監視Bot

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/kqns91/kube-watcher)](https://github.com/kqns91/kube-watcher/releases)

## 概要

`kube-watcher`は、Kubernetesクラスタ内の特定のNamespace内でリソースの変更を監視し、Slackへ通知を送信する軽量な監視Botです。

BotKubeやRobustaなどの既存ツールは**ClusterRole**（クラスタ全体への権限）が必要ですが、`kube-watcher`は**Namespace限定のRole**のみで動作するため、厳格なRBACポリシーが適用されている環境でも安全にご利用いただけます。

## 主な特徴

- **🔒 セキュア**: ClusterRole不要、Namespace限定のRole権限のみで動作
- **🐳 image変更検知** (v0.6.0): 指定したワークロードのコンテナイメージが変更されたら通知
  - Deployment/StatefulSet/DaemonSet: 新imageのPodが起動した時に通知
  - CronJob: image定義が変わった時に通知（通常のJob実行では通知しない）
- **🌿 env変更検知** (v0.7.0): 指定したワークロードの環境変数が変更されたら通知
  - `container.Env` の追加/変更/削除を検知
  - `container.EnvFrom` の参照先変更（ConfigMap/Secret名）を検知
- **🔍 柔軟な監視**: Pod、Deployment、CronJobなど複数のリソースタイプに対応
- **⚙️ watches構文**: シンプルで直感的な監視設定
- **🎯 スマートな通知**: 不要な通知を削減する高度なフィルタリング
  - **変更差分フィルタリング**: 意味のある変更のみを通知（レプリカ数、イメージ、ステータス変化など）
  - **重複イベント抑止**: LRUキャッシュによる同一イベントの重複通知防止
- **🔄 ホットリロード**: ConfigMapの変更を自動検知してPod再起動不要で設定反映
- **📦 イベントバッチ処理**: 複数のイベントをまとめて通知し、通知頻度を最適化
  - 3つのモード（detailed/summary/smart）で柔軟な表示制御
  - スマートモードで重要イベント（削除など）は常に詳細表示
- **🎨 リッチな通知**: Slack Attachmentsによる色分けと詳細情報の表示
  - イベントタイプに応じた色分け（追加=緑、更新=黄、削除=赤）
  - コンテナイメージとタグ情報
  - レプリカ数の詳細（Desired/Ready/Current）
  - Podステータス、理由、メッセージなどの詳細情報
- **✨ カスタマイズ可能**: Goテンプレートを使用したSlackメッセージのカスタマイズ
- **🪶 軽量**: 最小限のリソースフットプリント、シンプルな依存関係

## アーキテクチャ

```
┌──────────────┐
│  ConfigMap   │  設定ファイル（ConfigMap）
└──────┬───────┘
       │
       │ (fsnotify watch)
       │
┌──────▼────────┐
│ ConfigWatcher │  設定変更の自動検知・ホットリロード
└───────────────┘
       ┃
       ┃ (reload components)
       ┃
       ▼
┌─────────────┐
│ Kubernetes  │
│   API       │
└──────┬──────┘
       │
       │ (informer/watch)
       │
┌──────▼──────┐
│   Watcher   │  リソース変更の検知 + image/env変更検知
└──────┬──────┘
       │
       │ (events)
       │
┌──────▼──────┐
│   Filter    │  watches設定に基づくフィルタリング
└──────┬──────┘
       │
       │ (filtered events)
       │
┌──────▼──────┐
│ Deduplicator│  重複イベント抑止（LRUキャッシュ）
└──────┬──────┘
       │
       │ (unique events)
       │
       ├──────────────────┐
       │                  │
       │ (batching=off)   │ (batching=on)
       │                  │
       │           ┌──────▼──────┐
       │           │   Batcher   │  イベント集約・バッチ処理
       │           └──────┬──────┘
       │                  │
       │                  │ (batch window)
       │                  │
       └──────────────────┤
                          │
                   ┌──────▼──────┐
                   │  Formatter  │  メッセージの整形（Slack Attachments）
                   └──────┬──────┘
                          │
                          │ (formatted message)
                          │
                   ┌──────▼──────┐
                   │  Notifier   │  通知の送信
                   └──────┬──────┘
       │
       │ (webhook)
       │
┌──────▼──────┐
│    Slack    │
└─────────────┘
```

## クイックスタート

kube-watcherは、以下の3つの方法でデプロイできます：

### 📦 方法1: Helm（推奨）

最も簡単で柔軟な方法です。

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

詳細は [Helm Chartドキュメント](charts/kube-watcher/README.md) をご覧ください。

### 📝 方法2: Helmfile（宣言的管理）

helmfileで宣言的に管理する場合：

```yaml
# helmfile.yaml
repositories:
  - name: kube-watcher
    url: https://kqns91.github.io/kube-watcher/

releases:
  - name: kube-watcher
    namespace: monitoring
    chart: kube-watcher/kube-watcher
    version: ~0.6.0
    values:
      - namespace: monitoring
        slack:
          webhookUrl: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
        config:
          # watches構文によるシンプルな監視設定
          watches:
            - selector:
                kind: Deployment
                labels:
                  app: my-app
              triggers:
                - imageChange: true
                  envChange: true
            - selector:
                kind: CronJob
              triggers:
                - imageChange: true
          # 重複排除設定（オプション）
          deduplication:
            enabled: true
            ttlSeconds: 300
            maxCacheSize: 1000
          # バッチ処理設定（オプション）
          batching:
            enabled: false
            windowSeconds: 300
            mode: smart
```

```bash
helmfile apply
```

### ⚙️ 方法3: kubectl（マニフェスト直接適用）

### 前提条件

- Kubernetesクラスタ（v1.20以降）
- kubectlの設定済み環境
- Slack Webhook URL（[こちら](https://api.slack.com/messaging/webhooks)から取得可能）

### 1. リポジトリのクローン

```bash
git clone https://github.com/kqns91/kube-watcher.git
cd kube-watcher
```

### 2. Slack Webhookの設定

`deployments/secret.yaml`を編集し、Slack Webhook URLを設定します。

```yaml
stringData:
  slack-webhook-url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

### 3. 設定のカスタマイズ（任意）

`deployments/configmap.yaml`を編集して、監視対象とトリガーを設定できます。

### 4. Kubernetesへのデプロイ

```bash
# 必要に応じてNamespaceを変更（デフォルトは "default"）
# sed -i 's/namespace: default/namespace: your-namespace/g' deployments/*.yaml

# RBACの適用
kubectl apply -f deployments/rbac.yaml

# Secretの適用
kubectl apply -f deployments/secret.yaml

# ConfigMapの適用
kubectl apply -f deployments/configmap.yaml

# Dockerイメージのビルドとプッシュ
docker build -t your-registry/kube-watcher:latest .
docker push your-registry/kube-watcher:latest

# deployment.yamlのイメージを更新
# sed -i 's|image: kube-watcher:latest|image: your-registry/kube-watcher:latest|' deployments/deployment.yaml

# アプリケーションのデプロイ
kubectl apply -f deployments/deployment.yaml
```

### 5. デプロイの確認

```bash
# Podの稼働状況を確認
kubectl get pods -l app=kube-watcher

# ログの確認
kubectl logs -l app=kube-watcher -f
```

## Slack通知の表示例

Slack Attachments API を使用したリッチな通知フォーマットに対応しています。

### 通知の色分け

イベントタイプに応じて、メッセージの左側に色が表示されます：

- 🟢 **ADDED（作成）**: 緑色 - 新しいリソースが作成されたとき
- 🟡 **UPDATED（更新）**: 黄色 - 既存のリソースが更新されたとき
- 🔴 **DELETED（削除）**: 赤色 - リソースが削除されたとき

### 表示される詳細情報

リソースタイプに応じて、以下の詳細情報が自動的に表示されます：

#### Deployment の場合
- コンテナ情報（名前とイメージタグ）
- レプリカ情報（Desired / Ready / Current）
- Deployment のステータスと理由
- **image変更時**: 変更前後のイメージ情報

#### Pod の場合
- Podのステータス（Running、Pending、Failed など）
- コンテナイメージ情報
- 理由とメッセージ（エラー時など）

#### CronJob の場合
- コンテナイメージ情報
- **image変更時**: 変更前後のイメージ情報（通常のJob実行では通知しない）

#### Service の場合
- サービスタイプ（ClusterIP、LoadBalancer など）

## 設定方法

### 監視可能なリソース

以下のKubernetesリソースの監視に対応しています。

- `Pod`
- `Deployment`
- `Service`
- `ConfigMap`
- `Secret`
- `ReplicaSet`
- `StatefulSet`
- `DaemonSet`
- `CronJob` (v0.6.0で追加)

### イベントタイプ

- `ADDED`: リソースが作成された
- `UPDATED`: リソースが更新された
- `DELETED`: リソースが削除された

### 設定例（watches構文）

v0.6.0 から、シンプルで直感的な `watches` 構文で監視設定を記述します。

```yaml
namespace: "production"

# watches構文による監視設定
watches:
  # Deploymentのimage変更とenv変更を監視
  - selector:
      kind: Deployment
      labels:
        app: my-app
    triggers:
      - imageChange: true
        envChange: true    # v0.7.0で追加

  # StatefulSetのimage変更を監視
  - selector:
      kind: StatefulSet
    triggers:
      - imageChange: true

  # DaemonSetのimage変更を監視
  - selector:
      kind: DaemonSet
    triggers:
      - imageChange: true

  # CronJobのimage変更を監視（通常のJob実行では通知しない）
  - selector:
      kind: CronJob
    triggers:
      - imageChange: true

  # 特定の名前のPodを監視
  - selector:
      kind: Pod
      name: important-pod
    triggers:
      - imageChange: true

  # 環境変数の変更のみを監視する例
  - selector:
      kind: Deployment
      name: config-sensitive-app
    triggers:
      - envChange: true    # envChangeのみ

notifier:
  slack:
    webhookUrl: "${SLACK_WEBHOOK_URL}"
    template: |
      :warning: *[{{ .Kind }}]* `{{ .Namespace }}/{{ .Name }}`
      アクション: *{{ .EventType }}*
      時刻: {{ .Timestamp }}
      {{- if .Labels }}
      ラベル: {{ range $k, $v := .Labels }}{{ $k }}={{ $v }} {{ end }}
      {{- end }}

# イベント重複排除設定（オプション）
deduplication:
  enabled: true        # 重複排除を有効化
  ttlSeconds: 300      # 5分間同じイベントは通知しない
  maxCacheSize: 1000   # 最大1000エントリをキャッシュ

# イベントバッチ処理設定（オプション）
batching:
  enabled: false       # バッチ処理を有効化
  windowSeconds: 300   # 5分間のイベントをまとめて通知
  mode: smart          # detailed/summary/smart
  smart:
    maxEventsPerGroup: 5    # グループごとに最大5件まで詳細表示
    maxTotalEvents: 20      # 合計20件を超えるとサマリーモード
    alwaysShowDetails:      # 常に詳細表示するイベントタイプ
      - DELETED
```

### watches構文の詳細

#### selector（監視対象の指定）

| フィールド | 説明 | 例 |
|-----------|------|-----|
| `kind` | リソース種類（必須） | `Deployment`, `Pod`, `CronJob` |
| `name` | リソース名（オプション） | `my-app` |
| `labels` | ラベルセレクター（オプション） | `app: web` |

#### triggers（通知トリガー）

| フィールド | 説明 | 対象リソース |
|-----------|------|--------------|
| `imageChange` | コンテナイメージの変更時に通知 | Pod, Deployment, StatefulSet, DaemonSet, CronJob |
| `envChange` | 環境変数の変更時に通知 (v0.7.0) | Pod, Deployment, StatefulSet, DaemonSet, CronJob |

### image変更検知の動作

| リソース | 検知方法 | 説明 |
|---------|---------|------|
| Deployment | 新imageのPodがADDED | ロールアウト時に新しいPodが起動したタイミングで通知 |
| StatefulSet | 新imageのPodがADDED | 同上 |
| DaemonSet | 新imageのPodがADDED | 同上 |
| CronJob | image定義がUPDATED | JobTemplate内のimage定義が変わった時のみ通知（通常のJob実行では通知しない） |

### env変更検知の動作 (v0.7.0)

| リソース | 検知方法 | 説明 |
|---------|---------|------|
| Deployment | env定義がUPDATED | `container.Env` や `container.EnvFrom` の変更時に通知 |
| StatefulSet | env定義がUPDATED | 同上 |
| DaemonSet | env定義がUPDATED | 同上 |
| CronJob | env定義がUPDATED | JobTemplate内のenv定義が変わった時のみ通知 |
| Pod | ADDED | 新しいPodが追加されたタイミングで通知 |

#### 検知対象

- `container.Env` の追加/変更/削除
  - 環境変数の名前や値の変更
  - `valueFrom` による ConfigMap/Secret 参照の変更
- `container.EnvFrom` の参照先変更
  - ConfigMap 名の変更
  - Secret 名の変更

#### 制限事項

- ConfigMap/Secret の**中身**の変更は検知不可（参照定義のみ）
- ConfigMap/Secret の内容変更を検知するには、参照名を変更する必要があります

### テンプレート変数

`template`フィールドで利用可能な変数は以下の通りです。

#### 基本情報

| 変数 | 説明 | 例 |
|------|------|-----|
| `.Kind` | リソースの種類 | `Pod`, `Deployment` |
| `.Namespace` | Namespace名 | `default`, `production` |
| `.Name` | リソース名 | `my-app-123` |
| `.EventType` | イベントタイプ | `ADDED`, `UPDATED`, `DELETED` |
| `.Timestamp` | イベント発生時刻 | `2025-10-28T12:34:56Z` |
| `.Labels` | リソースのラベル | `map[app:web env:prod]` |

#### 詳細情報

| 変数 | 説明 | 対象リソース |
|------|------|--------------|
| `.Status` | リソースのステータス | Pod |
| `.Reason` | イベントの理由 | Pod, Deployment |
| `.Message` | イベントメッセージ | Pod, Deployment |
| `.Containers` | コンテナ情報（名前、イメージ） | Pod, Deployment, CronJob |
| `.Replicas` | レプリカ情報（Desired/Ready/Current） | Deployment, ReplicaSet, StatefulSet |
| `.ServiceType` | サービスタイプ | Service |
| `.ImageChanged` | image変更フラグ | Deployment, StatefulSet, DaemonSet, CronJob |
| `.OldImages` | 変更前のイメージ情報 | 同上 |
| `.NewImages` | 変更後のイメージ情報 | 同上 |
| `.EnvChanged` | env変更フラグ (v0.7.0) | Deployment, StatefulSet, DaemonSet, CronJob |
| `.OldEnv` | 変更前の環境変数情報 (v0.7.0) | 同上 |
| `.NewEnv` | 変更後の環境変数情報 (v0.7.0) | 同上 |

**注意**: デフォルトでは Slack Attachments 形式で通知が送信されるため、これらの詳細情報は自動的に整形されて表示されます。カスタムテンプレートを使用する場合のみ、これらの変数を明示的に参照する必要があります。

## 開発

### ローカル開発環境

```bash
# 依存関係のインストール
go mod download

# ローカルでの実行（kubeconfigが必要）
go run cmd/main.go -config config/config.yaml
```

### ビルド

```bash
# バイナリのビルド
go build -o kube-watcher ./cmd

# Dockerイメージのビルド
docker build -t kube-watcher:latest .
```

### Makefileの利用

プロジェクトにはMakefileが含まれており、以下のコマンドが利用できます。

```bash
make build          # バイナリのビルド
make run            # ローカルでの実行
make test           # テストの実行
make lint           # コードのLint
make lint-fix       # Lintエラーの自動修正
make fmt            # コードフォーマット
make docker-build   # Dockerイメージのビルド
make deploy         # Kubernetesへのデプロイ
make logs           # ログの表示
make undeploy       # Kubernetesからのアンデプロイ
```

### コード品質

プロジェクトでは[golangci-lint](https://golangci-lint.run/)を使用してコード品質を維持しています。

```bash
# golangci-lintのインストール
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Lintの実行
make lint

# Lintエラーの自動修正
make lint-fix
```

### プロジェクト構成

```
.
├── cmd/
│   └── main.go                 # アプリケーションのエントリーポイント
├── pkg/
│   ├── config/                 # 設定管理
│   │   └── config.go
│   ├── watcher/                # Kubernetesリソース監視 + image/env変更検知
│   │   └── watcher.go
│   ├── filter/                 # イベントフィルタリング（watches構文）
│   │   └── filter.go
│   ├── dedup/                  # 重複イベント抑止
│   │   ├── dedup.go
│   │   └── dedup_test.go
│   ├── batcher/                # イベントバッチ処理
│   │   ├── batcher.go
│   │   └── batcher_test.go
│   ├── reload/                 # 設定ホットリロード
│   │   ├── reload.go
│   │   └── reload_test.go
│   ├── formatter/              # メッセージ整形
│   │   └── formatter.go
│   └── notifier/               # 通知送信
│       └── notifier.go
├── config/
│   └── config.yaml             # 設定ファイルのサンプル
├── deployments/
│   ├── rbac.yaml               # RBACマニフェスト
│   ├── secret.yaml             # Webhook URL用Secret
│   ├── configmap.yaml          # 設定用ConfigMap
│   └── deployment.yaml         # Deploymentマニフェスト
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```

## RBAC権限

本アプリケーションは**Namespace限定の権限**のみを必要とします。

```yaml
rules:
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "secrets", "events"]
    verbs: ["list", "watch", "get"]

  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
    verbs: ["list", "watch", "get"]

  - apiGroups: ["batch"]
    resources: ["cronjobs"]
    verbs: ["list", "watch", "get"]
```

**ClusterRoleは不要です！** そのため、マルチテナント環境でも安全にご利用いただけます。

## ロードマップ

### 完了 ✅
- [x] **基本機能の実装** - Kubernetesリソースの監視とSlack通知
- [x] **Helmチャート対応** - Helmによる簡単なデプロイ
- [x] **CI/CD構築** - GitHub Actionsによる自動ビルドとリリース
- [x] **リッチな通知フォーマット** - Slack Attachments APIによる色分け表示
- [x] **詳細情報の表示** - コンテナイメージ、レプリカ数、ステータスなど
- [x] **変更差分フィルタリング** - 意味のある変更のみを通知
- [x] **重複イベント抑止（LRUキャッシュ）** - 同一イベントの重複通知を防止
- [x] **ConfigMapのホットリロード** - Pod再起動なしで設定を自動反映
- [x] **イベント集約とバッチ処理** - 複数イベントをまとめて通知、3つのモード対応
- [x] **image変更検知 + watches構文** (v0.6.0) - 指定したワークロードのimage変更を検知
- [x] **env変更検知** (v0.7.0) - 指定したワークロードの環境変数変更を検知

### 将来
- [ ] 追加の通知先対応（Teams、Discord、汎用Webhook）
- [ ] リソースタイプごとのカスタムテンプレート
- [ ] メトリクスエンドポイント（Prometheus対応）
- [ ] Web UIダッシュボード

## トラブルシューティング

### Podが起動しない場合

```bash
# RBACの確認
kubectl get role,rolebinding -n your-namespace

# ログの確認
kubectl logs -l app=kube-watcher -n your-namespace
```

### 通知が届かない場合

1. Secretに設定されたSlack Webhook URLが正しいか確認してください
2. アプリケーションのログでエラーが発生していないか確認してください
3. Webhookを手動でテストしてください

   ```bash
   curl -X POST -H 'Content-type: application/json' \
     --data '{"text":"テストメッセージ"}' \
     YOUR_WEBHOOK_URL
   ```

### イベントが検知されない場合

1. リソースが監視対象のNamespace内に存在することを確認してください
2. watches設定を確認してください
   - `selector.kind` が正しいか
   - `selector.labels` が対象リソースに一致しているか
   - `triggers` が設定されているか
3. RBACのリソース権限を確認してください

### 通知が頻繁すぎる場合

kube-watcher には複数の通知削減機能があります：

1. **watches構文のlabelsセレクター**
   - 監視対象を特定のラベルを持つリソースに限定
   ```yaml
   watches:
     - selector:
         kind: Deployment
         labels:
           environment: production  # productionラベルのみ監視
       triggers:
         - imageChange: true
   ```

2. **重複イベント抑止**
   - 同じイベントが短時間に複数回発生しても1回だけ通知
   - `deduplication.enabled: true` で有効化
   - `ttlSeconds` で重複判定期間を調整（デフォルト: 300秒）

3. **イベントバッチ処理**
   - 複数のイベントをまとめて通知し、通知回数を削減
   - `batching.enabled: true` で有効化
   - `windowSeconds` でバッチウィンドウを調整（デフォルト: 300秒 = 5分）
   - スマートモードで自動的に詳細/サマリーを切り替え
   ```yaml
   batching:
     enabled: true
     windowSeconds: 300  # 5分ごとにまとめて通知
     mode: smart
   ```

### 設定変更が反映されない場合

ConfigMap の変更は自動的に検知され、Pod の再起動なしで反映されます。

1. **ホットリロードの動作確認**
   ```bash
   # ConfigMapを更新
   kubectl edit configmap kube-watcher-config -n your-namespace

   # ログで設定の再読み込みを確認
   kubectl logs -l app=kube-watcher -n your-namespace -f
   # 以下のようなログが出力されるはずです：
   # "Configuration file changed, reloading..."
   # "Configuration reloaded successfully"
   ```

2. **ホットリロードが動作しない場合**
   - ConfigMap がマウントされているか確認してください
   - ログにエラーが出ていないか確認してください
   - 最終手段として Pod を再起動してください：
     ```bash
     kubectl rollout restart deployment kube-watcher -n your-namespace
     ```

## コントリビューション

プルリクエストを歓迎いたします！以下の手順でご協力ください。

1. このリポジトリをForkしてください
2. フィーチャーブランチを作成してください（`git checkout -b feature/amazing-feature`）
3. 変更をコミットしてください（`git commit -m 'Add some amazing feature'`）
4. ブランチにPushしてください（`git push origin feature/amazing-feature`）
5. プルリクエストを作成してください

### 開発ガイドライン

- コードは`go fmt`でフォーマットしてください
- `make lint`でLintチェックを行い、エラーがないことを確認してください
- 新機能には適切なコメントを追加してください
- 可能な限りテストを追加してください
- コミット前に必ず`make lint`と`make test`を実行してください

## ライセンス

本プロジェクトはMITライセンスの下で公開されています。詳細は[LICENSE](LICENSE)ファイルをご覧ください。

## 参考資料

- [Kubernetes client-go](https://github.com/kubernetes/client-go) - 公式Kubernetes Goクライアント
- [Slack Incoming Webhooks](https://api.slack.com/messaging/webhooks) - Slack Webhook API
- [BotKube](https://github.com/kubeshop/botkube) - インスピレーション元
- [Robusta](https://github.com/robusta-dev/robusta) - インスピレーション元

## 設計思想

本プロジェクトは以下の原則に基づいて設計されています。

| 原則 | 内容 |
|------|------|
| セキュア | ClusterRole禁止・Namespace限定アクセスのみ |
| シンプル | 外部依存を最小化・Go標準ライブラリ中心 |
| 拡張性 | watcher/filter/formatter/notifierをinterface分離 |
| 管理容易 | Helm/ConfigMapで構成を完全外部化 |

## サポート

問題が発生した場合や機能のリクエストがある場合は、[GitHub Issues](https://github.com/kqns91/kube-watcher/issues)にてお気軽にお問い合わせください。

---

**kube-watcher**をご利用いただき、ありがとうございます。
