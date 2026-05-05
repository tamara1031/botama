# Getting Started

## 前提条件

- Docker / Kubernetes クラスタ
- Discord Developer Portal でのアプリケーション・bot トークンの取得

## Discord アプリの準備

1. [Discord Developer Portal](https://discord.com/developers/applications) でアプリを作成する。
2. **Bot** タブで bot ユーザーを作成し、トークンをコピーする。
3. **Privileged Gateway Intents** で `Message Content Intent` を有効にする（メッセージ内容を読むために必要）。
4. **OAuth2 → URL Generator** で `bot` スコープと必要な権限（`Send Messages`, `Read Message History` など）を選んで招待 URL を生成し、サーバーに招待する。

## 環境変数

| 変数名 | 必須 | 説明 |
|---|---|---|
| `DISCORD_TOKEN` | ✅ | Discord bot トークン |
| `MODULES_ENABLED` | — | 有効化するモジュール名（カンマ区切り）。未設定の場合は何も起動しない |
| `NOTIFICATION_CHANNEL_ID` | — | 将来の通知モジュール用チャンネル ID（現時点では未使用） |

## ローカル実行

```bash
# Go 1.22+ が必要
go run ./cmd/bot \
  -e DISCORD_TOKEN=your_token \
  MODULES_ENABLED=ping
```

または `.env` ファイルを使う場合:

```bash
export DISCORD_TOKEN=your_token
export MODULES_ENABLED=ping
go run ./cmd/bot
```

## Docker で実行

```bash
docker build -t botama .

docker run --rm \
  -e DISCORD_TOKEN=your_token \
  -e MODULES_ENABLED=ping \
  botama
```

## Kubernetes へのデプロイ

`Secret` でトークンを管理し、`Deployment` で bot を動かす例:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: discord-bot-secret
type: Opaque
stringData:
  token: "your_discord_token"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: botama
spec:
  replicas: 1          # Discord bot は単一インスタンスで運用
  selector:
    matchLabels:
      app: botama
  template:
    metadata:
      labels:
        app: botama
    spec:
      containers:
        - name: bot
          image: ghcr.io/<owner>/botama:latest
          env:
            - name: DISCORD_TOKEN
              valueFrom:
                secretKeyRef:
                  name: discord-bot-secret
                  key: token
            - name: MODULES_ENABLED
              value: "ping"
          resources:
            requests:
              memory: "32Mi"
              cpu: "10m"
            limits:
              memory: "128Mi"
              cpu: "100m"
          securityContext:
            runAsNonRoot: true
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
```

> **注意**: Discord bot は WebSocket 接続のため `replicas: 1` 推奨。複数レプリカを立てると同じイベントを重複処理する。

## 利用可能なモジュール

### ping

`MODULES_ENABLED` に `ping` を追加すると有効になる。

| コマンド | 応答 |
|---|---|
| `!ping` | `pong` |

## 新しいモジュールの追加

[architecture/overview.md](../architecture/overview.md) のモジュール実装パターンを参照。

1. `internal/modules/<name>/` にパッケージを作成する。
2. `bot.Module` インターフェースを実装する。
3. `cmd/bot/main.go` に `b.RegisterModule(<name>.New())` を追加する。
4. `MODULES_ENABLED` に名前を加えてデプロイする。
