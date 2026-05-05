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
│       ├── ping/
│       │   └── ping.go          # ping モジュール（スラッシュコマンド）
│       └── notify/
│           └── notify.go        # notify モジュール（HTTP API）
├── docs/                        # 設計ドキュメント
├── .github/
│   └── workflows/
│       ├── preview.yml          # main マージ時 preview イメージビルド
│       └── release.yml          # 手動 dispatch によるリリース
├── Dockerfile
├── compose.yaml
├── Makefile
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
│  4. Bot.Start() → READY 待機後にモジュール起動 │
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
         ┌────────────┴────────────┐
         ▼                         ▼
    ┌─────────┐              ┌──────────┐
    │  ping   │              │  notify  │
    │ /ping   │              │ POST     │
    │ → pong  │              │ /notify  │
    └─────────┘              └──────────┘
    Discord Gateway          HTTP :8080
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
`Register` は READY イベント受信後に呼び出される。
`Unregister` はシャットダウン時に呼び出され、リソースを解放する。

### `config.Config`

| フィールド | 環境変数 | 必須 | 説明 |
|---|---|---|---|
| `Token` | `DISCORD_TOKEN` | ✅ | Discord bot トークン |
| `GuildID` | `GUILD_ID` | — | ギルドコマンド登録先のサーバー ID |
| `NotificationChannelID` | `NOTIFICATION_CHANNEL_ID` | notify | notify モジュールの通知先チャンネル ID |
| `APIToken` | `API_TOKEN` | notify | notify モジュールの Bearer 認証トークン |
| `APIAddr` | `API_ADDR` | — | notify モジュールのリッスンアドレス（デフォルト `:8080`） |
| `EnabledModules` | `MODULES_ENABLED` | — | 有効化するモジュール名（カンマ区切り） |

### `bot.Bot`

```go
type Bot struct {
    session  *discordgo.Session
    registry *registry
    cfg      *config.Config
}
```

- `New(cfg)` でセッションを生成する（`IntentsNone`）。
- `RegisterModule(m)` で Registry にモジュールを登録する（起動前）。
- `Start()` でセッションを開き、READY イベントを待ってから `cfg.EnabledModules` に列挙されたモジュールを起動する。
- `Stop()` で全モジュールを `Unregister` してセッションを閉じる。

### `bot.registry`（非公開）

| メソッド | 説明 |
|---|---|
| `add(m Module)` | モジュールを登録 |
| `startEnabled(s, names)` | 指定モジュールの `Register` を呼び出す |
| `stopAll()` | 起動中モジュールを全て `Unregister` |

## モジュール実装パターン

### スラッシュコマンド型（ping を参照）

```go
package mymodule

import (
    "fmt"
    "github.com/bwmarrin/discordgo"
)

type MyModule struct {
    guildID       string
    session       *discordgo.Session
    removeHandler func()
    commandID     string
}

func New(guildID string) *MyModule { return &MyModule{guildID: guildID} }

func (m *MyModule) Name() string { return "mymodule" }

func (m *MyModule) Register(s *discordgo.Session) error {
    cmd, err := s.ApplicationCommandCreate(s.State.User.ID, m.guildID, &discordgo.ApplicationCommand{
        Name:        "mycommand",
        Description: "コマンドの説明",
    })
    if err != nil {
        return fmt.Errorf("mymodule: register: %w", err)
    }
    m.session = s
    m.commandID = cmd.ID

    m.removeHandler = s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        if i.Type != discordgo.InteractionApplicationCommand {
            return
        }
        if i.ApplicationCommandData().Name != "mycommand" {
            return
        }
        s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{Content: "応答"},
        })
    })
    return nil
}

func (m *MyModule) Unregister() error {
    if m.removeHandler != nil {
        m.removeHandler()
    }
    if m.commandID != "" {
        m.session.ApplicationCommandDelete(m.session.State.User.ID, m.guildID, m.commandID)
    }
    return nil
}
```

### HTTP サーバー型（notify を参照）

`Register` で `net.Listen` → goroutine で `Serve`、`Unregister` で `Shutdown` する。

## K8s 運用上の注意

- SIGTERM を受けると `Bot.Stop()` を呼び出してグレースフルシャットダウンする。
- コンテナは非 root ユーザー（distroless の `nonroot`）で動作する。
- 永続的なファイル書き込みは行わないため、読み取り専用ファイルシステムに対応可能。
- notify モジュールが有効な場合、ClusterIP Service 経由でクラスタ内の他 Pod から `POST /notify` で通知を送信できる。
- Discord bot は WebSocket 長接続クライアントのため `replicas: 1` 推奨。
