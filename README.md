# botama

Kubernetes 上で動作するモジュール型 Discord bot。

## クイックスタート

```bash
export DISCORD_TOKEN=your_token
export MODULES_ENABLED=ping
go run ./cmd/bot
```

## モジュール

| 名前 | コマンド | 応答 |
|---|---|---|
| ping | `!ping` | `pong` |

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

> **注意**: `go.mod` のモジュールパス `github.com/tamara1031/botama` は実際のリポジトリパスに合わせて変更してください。
