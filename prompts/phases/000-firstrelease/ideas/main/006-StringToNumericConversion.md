# 006: CallFunction の String → 数値型 変換サポート

## 背景 (Background)

Kuniumi フレームワークの `CallFunction` (`reflection.go`) は、登録された Go 関数を `map[string]interface{}` 形式の引数で呼び出す。現在の型変換ロジックは以下の2パターンのみ対応している:

1. **`reflect.ConvertibleTo` が true の場合**: そのまま `Convert()` で変換
2. **`float64` → 整数型**: JSON デコード時に数値が `float64` になるケースへの対応

しかし、**引数が `string` 型で渡された場合の数値型への変換が未実装**であり、以下のエラーが発生する:

```
Error: cannot convert string to int
```

### 発生条件

このエラーは、引数の値が `string` 型として `CallFunction` に渡されたときに発生する。現在のアダプター（HTTP, CGI, MCP）はすべて JSON ボディ経由で引数を受け取るため、JSON の数値リテラル (`{"x": 10}`) を使えば `float64` として渡され正常に動作する。

しかし、以下のケースでは `string` として渡される可能性がある:

- JSON で値を文字列として記述した場合 (`{"x": "10"}`)
- 外部システム（Docker コンテナの CGI）から文字列パラメータが渡される場合
- 将来的にクエリパラメータやフォームデータをサポートする場合

### 該当コード

`reflection.go` 108行目:

```go
return nil, fmt.Errorf("cannot convert %v to %v", targetVal.Type(), targetType)
```

## 要件 (Requirements)

### 必須要件

1. **`string` → 整数型の変換**: `string` 値を `int`, `int8`, `int16`, `int32`, `int64` へ変換できること
2. **`string` → 符号なし整数型の変換**: `string` 値を `uint`, `uint8`, `uint16`, `uint32`, `uint64` へ変換できること
3. **`string` → 浮動小数点型の変換**: `string` 値を `float32`, `float64` へ変換できること
4. **`string` → 真偽値の変換**: `string` 値 (`"true"`, `"false"`, `"1"`, `"0"` 等) を `bool` へ変換できること
5. **エラーハンドリング**: 変換不可能な文字列（例: `"abc"` → `int`）が渡された場合、明確なエラーメッセージを返すこと
6. **既存動作の維持**: 現在動作している `float64` → `int` の変換、および `ConvertibleTo` による変換は引き続き正常に動作すること

### 任意要件

- `string` → `string` は `ConvertibleTo` で対応済みのため、追加対応不要

## 実現方針 (Implementation Approach)

### 変更対象ファイル

- `reflection.go`: `CallFunction` 関数内の型変換ロジックを拡張

### 変更内容

`CallFunction` の型変換分岐（94〜109行目）に、`string` 型からの変換ハンドリングを追加する:

```go
// 既存のコード
if targetVal.Type().ConvertibleTo(targetType) {
    in = append(in, targetVal.Convert(targetType))
} else {
    if targetVal.Kind() == reflect.Float64 {
        // 既存: float64 → int 変換
    } else if targetVal.Kind() == reflect.String {
        // 新規: string → 数値型/bool 変換
        converted, err := convertStringToType(targetVal.String(), targetType)
        if err != nil {
            return nil, fmt.Errorf("cannot convert string %q to %v: %w", targetVal.String(), targetType, err)
        }
        in = append(in, converted)
    } else {
        return nil, fmt.Errorf("cannot convert %v to %v", targetVal.Type(), targetType)
    }
}
```

### 新規ヘルパー関数

`convertStringToType(s string, targetType reflect.Type) (reflect.Value, error)` を `reflection.go` 内に追加する。`strconv` パッケージを使って以下の変換を行う:

| 入力型 | ターゲット型 | 変換関数 |
|--------|-------------|----------|
| `string` | `int*` | `strconv.ParseInt` |
| `string` | `uint*` | `strconv.ParseUint` |
| `string` | `float*` | `strconv.ParseFloat` |
| `string` | `bool` | `strconv.ParseBool` |

## 検証シナリオ (Verification Scenarios)

### シナリオ 1: CGI モードで文字列数値の引数を渡す

1. `examples/basic` アプリをビルドする
2. `echo '{"x": "10", "y": "20"}' | PATH_INFO=/Add ./app cgi` を実行する
3. `{"result":30}` が返されることを確認する

### シナリオ 2: 変換不可能な文字列

1. `echo '{"x": "abc", "y": "20"}' | PATH_INFO=/Add ./app cgi` を実行する
2. `Status: 500` と変換エラーメッセージが返されることを確認する

### シナリオ 3: 既存の float64 入力が引き続き動作する

1. `echo '{"x": 10, "y": 20}' | PATH_INFO=/Add ./app cgi` を実行する
2. `{"result":30}` が返されることを確認する（既存テストと同等）

## テスト項目 (Testing for the Requirements)

### 単体テスト

`reflection.go` に対する単体テスト（新規ファイル `reflection_test.go`）を作成し、以下をカバーする:

| テストケース | 入力 | 期待結果 |
|-------------|------|---------|
| string → int | `"42"` → `int` | `42` |
| string → int64 | `"100"` → `int64` | `100` |
| string → uint | `"10"` → `uint` | `10` |
| string → float64 | `"3.14"` → `float64` | `3.14` |
| string → float32 | `"2.5"` → `float32` | `2.5` |
| string → bool (true) | `"true"` → `bool` | `true` |
| string → bool (false) | `"false"` → `bool` | `false` |
| string → bool (1) | `"1"` → `bool` | `true` |
| 不正な文字列 → int | `"abc"` → `int` | エラー |
| 不正な文字列 → float | `"xyz"` → `float64` | エラー |
| 既存: float64 → int | `float64(42)` → `int` | `42`（回帰なし） |

### 統合テスト

既存の統合テスト (`tests/kuniumi/kuniumi_test.go`) に `string` 入力のテストケースを追加する:

| テストケース | 入力 JSON | 期待結果 |
|-------------|----------|---------|
| CGI: 文字列数値 | `{"x": "10", "y": "20"}` | `{"result":30}` |

### 検証コマンド

```bash
# 全体ビルド & 単体テスト
./scripts/process/build.sh

# 統合テスト
./scripts/process/integration_test.sh
```
