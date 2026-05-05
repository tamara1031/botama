# ADR 0002: プラグイン型モジュールシステムの設計

- **ステータス**: Accepted
- **決定日**: 2026-05-05

## コンテキスト

bot に機能を追加・削除・有効化・無効化できる拡張可能な仕組みが必要。

## 決定

`Module` インターフェースと `Registry` による疎結合なモジュールシステムを採用する。

## インターフェース定義

```go
type Module interface {
    Name() string
    Register(s *discordgo.Session) error
    Unregister() error
}
```

- `Name()` はモジュールの一意な識別子を返す（環境変数での有効化指定に使用）。
- `Register()` は Discord セッションにイベントハンドラを登録する。discordgo の `AddHandler` が返す削除関数を内部に保持することで `Unregister()` 時に確実に解除できる。
- `Unregister()` は登録済みハンドラを全て解除する。

## 有効化・無効化の方法

環境変数 `MODULES_ENABLED` にカンマ区切りでモジュール名を列挙する。

```
MODULES_ENABLED=ping,notifications
```

- 列挙されていないモジュールは `Registry` に登録されていてもハンドラが起動しない。
- 空の場合はどのモジュールも起動しない（bot は接続するが何も応答しない）。

### 代替案と不採用理由

| 案 | 不採用理由 |
|---|---|
| `MODULE_PING=true` などモジュールごとの個別フラグ | モジュール数が増えると env var が増殖する。CSV 一本の方がシンプル |
| ファイルベースの設定 (YAML/TOML) | ランタイム依存が増える。K8s ConfigMap との相性は良いが env var で十分 |
| Go plugin (.so) | クロスコンパイルが困難、ビルドが複雑 |

## モジュール追加手順

1. `internal/modules/<name>/` ディレクトリを作成する。
2. `Module` インターフェースを実装した struct を定義する。
3. `cmd/bot/main.go` の `b.RegisterModule(...)` に追加する。
4. `MODULES_ENABLED` に名前を追加する。

詳細は [architecture/overview.md](../architecture/overview.md) を参照。
