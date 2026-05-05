# botama

Kubernetes 上で動作するモジュール型 Discord bot。

## クイックスタート

```bash
cp .env.example .env
# .env を編集して DISCORD_TOKEN などを設定

make run
```

## モジュール

| 名前 | 種別 | 概要 |
|---|---|---|
| ping | スラッシュコマンド | `/ping` に `pong` で応答 |
| notify | HTTP API | `POST /notify/{channelID}` または `POST /notify`（デフォルトch）でDiscordに通知を送信 |

## ドキュメント

詳細は [docs/](docs/README.md) を参照。

- [アーキテクチャ](docs/architecture/overview.md)
- [Getting Started](docs/usage/getting-started.md)
- [ADR](docs/adr/)

## イメージ

| タグ | 用途 |
|---|---|
| `preview` / `preview-<sha>` | main マージ時に自動ビルド |
| `latest` / `X.Y.Z` | GitHub Actions の手動 dispatch でリリース |
