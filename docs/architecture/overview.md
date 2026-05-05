# アーキテクチャ概要

## ディレクトリ構成

```
botama/
├── cmd/
│   └── bot/
│       └── main.go              # エントリポイント
├── internal/
│   ├── config/
│   │   └── config.go            # 環境変数からの設定読み込み
│   ├── bot/
│   │   ├── module.go            # Module インターフェース定義
│   │   ├── registry.go          # モジュール Registry
│   │   └── bot.go               # Bot 本体（セッション管理）
│   └── modules/
│       └── ping/
│           └── ping.go          # ping モジュール実装
├── docs/                        # 設計ドキュメント
├── .github/
│   └── workflows/
│       ├── preview.yml          # main マージ時 preview イメージビルド
│       └── release.yml          # 手動 dispatch によるリリース
├── Dockerfile
├── VERSION                      # リリースバージョン管理ファイル
└── go.mod
```

## コンポーネント図

```
┌──────────────────────────────────────────────┐
│                  main.go                      │
│  1. Config 読み込み                            │
│  2. Bot 生成                                   │
│  3. Module 登録 (RegisterModule)               │
│  4. Bot.Start()                               │
│  5. SIGTERM 待機 → Bot.Stop()                 │
└─────────────────────┬────────────────────────┘
                      │
          ┌───────────▼────────────┐
          │          Bot           │
          │  - discordgo.Session   │
          │  - Registry            │
          │  - Config              │
          └───────────┬────────────┘
                      │ startEnabled / stopAll
          ┌───────────▼────────────┐
          │        Registry        │
          │  modules map[name]→M   │
          │  active  map[name]→bool│
          └───────────┬────────────┘
                      │ Register / Unregister
         ┌────────────┼────────────┐
         ▼            ▼            ▼
    ┌─────────┐  ┌─────────┐  ┌─────────┐
    │  ping   │  │  (次の  │  │  (将来  │
    │ Module  │  │ モジュール│  │  追加)  │
    └─────────┘  └─────────┘  └─────────┘
```

## 主要インターフェース・型

### `bot.Module` インターフェース

```go
type Module interface {
    Name() string
    Register(s *discordgo.Session) error
    Unregister() error
}
```

すべてのモジュールはこのインターフェースを実装する。
`Register` は discordgo セッションにイベントハンドラを登録し、
`Unregister` は登録したハンドラを解除する。

### `config.Config`

| フィールド | 環境変数 | 必須 | 説明 |
|---|---|---|---|
| `Token` | `DISCORD_TOKEN` | ✅ | Discord bot トークン |
| `EnabledModules` | `MODULES_ENABLED` | — | 有効化するモジュール名（カンマ区切り） |
| `NotificationChannelID` | `NOTIFICATION_CHANNEL_ID` | — | 通知用チャンネル ID（将来の通知モジュール用に予約） |

### `bot.Bot`

```go
type Bot struct {
    session  *discordgo.Session
    registry *registry
    cfg      *config.Config
}
```

- `New(cfg)` でセッションを生成し、Intents を設定する。
- `RegisterModule(m)` で Registry にモジュールを登録する（起動前）。
- `Start()` でセッションを開き、`cfg.EnabledModules` に列挙されたモジュールを起動する。
- `Stop()` で全モジュールのハンドラを解除してセッションを閉じる。

### `bot.registry`（非公開）

| メソッド | 説明 |
|---|---|
| `add(m Module)` | モジュールを登録 |
| `startEnabled(s, names)` | 指定モジュールの `Register` を呼び出す |
| `stopAll()` | 起動中モジュールを全て `Unregister` |

## モジュール実装パターン

```go
package mymodule

import "github.com/bwmarrin/discordgo"

type MyModule struct {
    removeHandler func()
}

func New() *MyModule { return &MyModule{} }

func (m *MyModule) Name() string { return "mymodule" }

func (m *MyModule) Register(s *discordgo.Session) error {
    m.removeHandler = s.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageCreate) {
        // ロジックを実装
    })
    return nil
}

func (m *MyModule) Unregister() error {
    if m.removeHandler != nil {
        m.removeHandler()
    }
    return nil
}
```

## K8s 運用上の注意

- SIGTERM を受けると `Bot.Stop()` を呼び出してグレースフルシャットダウンする。
- コンテナは非 root ユーザー（distroless の `nonroot`）で動作する。
- 永続的なファイル書き込みは行わないため、読み取り専用ファイルシステムに対応可能。
- Discord bot は WebSocket 長接続クライアントのため、K8s の liveness/readiness probe は通常不要（必要な場合は別途 HTTP ヘルスエンドポイントをモジュールとして追加する）。
