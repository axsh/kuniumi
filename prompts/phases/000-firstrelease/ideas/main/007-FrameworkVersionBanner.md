# 007: CLI バナーにフレームワークバージョンとコミットIDを表示

## 背景 (Background)

現在、Kuniumi アプリケーションの CLI バナーはアプリケーション名とバージョンのみを表示する:

```
Calculator v1.0.0
```

開発者やユーザーがデバッグ時に、アプリケーションがどのバージョンの Kuniumi フレームワーク上で動作しているかを把握できると便利である。特にバグ報告時にフレームワークのバージョンとビルド元のコミットIDが分かれば、問題の再現・特定が容易になる。

### 現在の実装

`app.go` 88行目:

```go
Short: fmt.Sprintf("%s v%s", cfg.Name, cfg.Version),
```

## 要件 (Requirements)

### 必須要件

1. **バナーにフレームワーク情報を表示**: CLI のヘルプ出力に Kuniumi フレームワークのバージョンとコミットIDを表示すること
   - 表示形式: `Calculator v1.0.0 (based on kuniumi v0.1.1 <commit-hash>)`
   - コミットハッシュは先頭7文字（短縮形）を使用する
2. **ビルド情報の自動取得**: `runtime/debug.ReadBuildInfo()` を使用し、Go モジュールのバージョン情報と VCS コミットIDを自動的に取得すること（`ldflags` による手動注入は不要）
3. **フォールバック動作**: ビルド情報が取得できない場合（`go run` 等）でもパニックせず、`(based on kuniumi dev)` のようにフォールバックすること

### 任意要件

- VCS の dirty フラグ（未コミット変更あり）がある場合は `-dirty` サフィックスを付与する

## 実現方針 (Implementation Approach)

### 技術: `runtime/debug.ReadBuildInfo`

Go 1.18+ で利用可能な `runtime/debug.ReadBuildInfo()` を使用する。この API は以下の情報を自動的に提供する:

- **モジュールバージョン**: `info.Deps` から `github.com/axsh/kuniumi` の `Version` フィールドを参照
- **VCS コミットID**: `info.Settings` から `vcs.revision` を参照
- **VCS dirty フラグ**: `info.Settings` から `vcs.modified` を参照

> **注意**: `ReadBuildInfo` はアプリケーション（`main` パッケージ）のビルド情報を返す。Kuniumi がライブラリとして使われる場合、コミットIDはアプリケーション側のものになる。フレームワークのバージョンは `info.Deps` から取得する。

### 変更対象

- `app.go`: バナー文字列の生成ロジックを変更
- `version.go` (新規): バージョン情報取得関数を分離

### 表示フォーマット

```
{AppName} v{AppVersion} (based on kuniumi {FrameworkVersion} {CommitHash})
```

例:
```
Calculator v1.0.0 (based on kuniumi v0.1.1 abc1234)
Calculator v1.0.0 (based on kuniumi v0.1.1 abc1234-dirty)
Calculator v1.0.0 (based on kuniumi dev)       ← ビルド情報取得不可時
```

## 検証シナリオ (Verification Scenarios)

1. `examples/basic` アプリをビルドする
2. `./bin/basic.exe --help` を実行する
3. 出力に `Calculator v1.0.0 (based on kuniumi ...` が含まれることを確認する
4. `kuniumi` のバージョンまたは `dev` が表示されていることを確認する

## テスト項目 (Testing for the Requirements)

### 単体テスト

| テストケース | 内容 | 期待結果 |
|-------------|------|---------|
| バージョン文字列生成 | `frameworkVersionString()` の戻り値を確認 | `"based on kuniumi ..."` 形式の文字列 |

### 統合テスト

既存の `TestKuniumiIntegration/Help` テストを拡張し、バナー文字列にフレームワーク情報が含まれることを確認する。

| テストケース | 入力 | 確認事項 |
|-------------|------|---------|
| Help バナー | `--help` | 出力に `based on kuniumi` が含まれること |

### 検証コマンド

```bash
# 全体ビルド & 単体テスト
./scripts/process/build.sh

# 統合テスト
./scripts/process/integration_test.sh
```
