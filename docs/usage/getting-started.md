# Getting Started

## 前提条件

- Docker / Kubernetes クラスタ
- Discord Developer Portal でのアプリケーション・bot トークンの取得

## Discord アプリの準備

1. [Discord Developer Portal](https://discord.com/developers/applications) でアプリを作成する。
2. **Bot** タブで bot ユーザーを作成し、トークンをコピーする。
3. **Privileged Gateway Intents** は全て無効でよい（スラッシュコマンドは不要）。
4. **OAuth2 → URL Generator** で以下を選んで招待 URL を生成し、サーバーに招待する。
   - スコープ: `bot` + `applications.commands`（スラッシュコマンドの登録に必須）
   - 権限: `Send Messages`

## 環境変数

| 変数名 | 必須 | 説明 |
|---|---|---|
| `DISCORD_TOKEN` | ✅ | Discord bot トークン |
| `MODULES_ENABLED` | — | 有効化するモジュール名（カンマ区切り）。未設定の場合は何も起動しない |
| `GUILD_ID` | — | ギルドコマンドとして登録する場合のサーバー ID。未設定はグローバル（反映まで最大1時間） |
| `NOTIFICATION_CHANNEL_ID` | notify | notify モジュール使用時の通知先チャンネル ID |
| `API_TOKEN` | notify | notify モジュールの Bearer 認証トークン（`openssl rand -hex 32` 推奨） |
| `API_ADDR` | — | notify モジュールの HTTP リッスンアドレス（デフォルト `:8080`） |

## ローカル実行

`.env` を用意してから `make run`:

```bash
cp .env.example .env
# .env を編集して各値を設定

make run
```

## Docker Compose で実行

```bash
make up        # ビルド＆起動（ポート 8080 をホストに転送）
make logs      # ログ確認
make down      # 停止
```

### notify モジュールのテスト

Docker Compose 起動後、ホストから curl で疎通確認できる:

```bash
# 成功 → 204 No Content、Discord に通知が届く
curl -i -X POST http://localhost:8080/notify \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "テスト通知"}'

# 認証なし → 401
curl -i -X POST http://localhost:8080/notify \
  -H "Content-Type: application/json" \
  -d '{"content": "テスト"}'

# content 欠落 → 422
curl -i -X POST http://localhost:8080/notify \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

## Kubernetes へのデプロイ

`Secret` でトークンを管理し、`Deployment` と `Service` で bot を動かす例:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: botama-secret
type: Opaque
stringData:
  discord-token: "your_discord_token"
  api-token: "your_api_token"         # openssl rand -hex 32
  channel-id: "your_channel_id"
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
          image: ghcr.io/tamara1031/botama:latest
          env:
            - name: DISCORD_TOKEN
              valueFrom:
                secretKeyRef:
                  name: botama-secret
                  key: discord-token
            - name: API_TOKEN
              valueFrom:
                secretKeyRef:
                  name: botama-secret
                  key: api-token
            - name: NOTIFICATION_CHANNEL_ID
              valueFrom:
                secretKeyRef:
                  name: botama-secret
                  key: channel-id
            - name: MODULES_ENABLED
              value: "ping,notify"
          ports:
            - name: api
              containerPort: 8080
              protocol: TCP
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
---
apiVersion: v1
kind: Service
metadata:
  name: botama
spec:
  selector:
    app: botama
  ports:
    - port: 8080
      targetPort: api
  type: ClusterIP
```

> **注意**: Discord bot は WebSocket 接続のため `replicas: 1` 推奨。複数レプリカを立てると同じイベントを重複処理する。

クラスタ内の他 Pod からの呼び出し例:

```bash
curl -X POST http://botama.default.svc.cluster.local:8080/notify \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "デプロイ完了"}'
```

## 利用可能なモジュール

### ping

`MODULES_ENABLED` に `ping` を追加すると有効になる。Discord の `/ping` スラッシュコマンドに `pong` で応答する。

`GUILD_ID` を設定するとギルドコマンドとして即時反映される（テスト時推奨）。未設定の場合はグローバルコマンドとして登録され、反映まで最大1時間かかる。

### notify

`MODULES_ENABLED` に `notify` を追加すると有効になる。`NOTIFICATION_CHANNEL_ID` と `API_TOKEN` が必須。

```
POST /notify
Authorization: Bearer <API_TOKEN>
Content-Type: application/json

{"content": "送りたいメッセージ"}
```

レスポンス: 成功時 `204 No Content`

## 新しいモジュールの追加

[architecture/overview.md](../architecture/overview.md) のモジュール実装パターンを参照。

1. `internal/modules/<name>/` にパッケージを作成する。
2. `bot.Module` インターフェースを実装する。
3. `cmd/bot/main.go` に `b.RegisterModule(<name>.New())` を追加する。
4. `MODULES_ENABLED` に名前を加えてデプロイする。
