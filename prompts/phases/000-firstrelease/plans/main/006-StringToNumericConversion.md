# 006-StringToNumericConversion

> **Source Specification**: [006-StringToNumericConversion.md](file:///c:/Users/yamya/myprog/kuniumi/prompts/phases/000-firstrelease/ideas/main/006-StringToNumericConversion.md)

## Goal Description

`reflection.go` の `CallFunction` において、引数が `string` 型で渡された場合に、ターゲット型（`int`, `uint`, `float`, `bool`）への変換を `strconv` パッケージにより実現する。これにより、JSON 文字列値や外部システムからの文字列パラメータを正しく処理できるようになる。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| `string` → 整数型 (`int*`) の変換 | Proposed Changes > `reflection.go` > `convertStringToType` |
| `string` → 符号なし整数型 (`uint*`) の変換 | Proposed Changes > `reflection.go` > `convertStringToType` |
| `string` → 浮動小数点型 (`float*`) の変換 | Proposed Changes > `reflection.go` > `convertStringToType` |
| `string` → 真偽値 (`bool`) の変換 | Proposed Changes > `reflection.go` > `convertStringToType` |
| エラーハンドリング（変換不可能な文字列） | Proposed Changes > `reflection.go` > `convertStringToType` + テスト |
| 既存動作の維持 (`float64` → `int` 等) | Verification Plan > 回帰テスト |

## Proposed Changes

### kuniumi (ルートパッケージ)

#### [NEW] [reflection_test.go](file:///c:/Users/yamya/myprog/kuniumi/reflection_test.go)
*   **Description**: `CallFunction` および新規関数 `convertStringToType` の単体テストを作成する（TDD: テストを先に記述）。
*   **Technical Design**:
    *   パッケージ: `package kuniumi`（内部関数 `convertStringToType` をテストするため、同一パッケージ内テスト）
    *   テーブル駆動テスト (`[]struct{...}`) を使用
    *   テスト用のダミー関数を `reflect.ValueOf` で取得し、`AnalyzeFunction` + `CallFunction` を呼び出す
*   **Logic**:

    **Test 1: `TestConvertStringToType`** — ヘルパー関数の直接テスト

    ```go
    func TestConvertStringToType(t *testing.T) {
        tests := []struct {
            name       string
            input      string
            targetType reflect.Type
            wantValue  interface{}
            wantErr    bool
        }{
            // int variants
            {"string to int", "42", reflect.TypeOf(int(0)), int(42), false},
            {"string to int8", "127", reflect.TypeOf(int8(0)), int8(127), false},
            {"string to int16", "1000", reflect.TypeOf(int16(0)), int16(1000), false},
            {"string to int32", "100000", reflect.TypeOf(int32(0)), int32(100000), false},
            {"string to int64", "9999999", reflect.TypeOf(int64(0)), int64(9999999), false},
            // uint variants
            {"string to uint", "10", reflect.TypeOf(uint(0)), uint(10), false},
            {"string to uint8", "255", reflect.TypeOf(uint8(0)), uint8(255), false},
            {"string to uint16", "65535", reflect.TypeOf(uint16(0)), uint16(65535), false},
            {"string to uint32", "100000", reflect.TypeOf(uint32(0)), uint32(100000), false},
            {"string to uint64", "100000", reflect.TypeOf(uint64(0)), uint64(100000), false},
            // float variants
            {"string to float32", "2.5", reflect.TypeOf(float32(0)), float32(2.5), false},
            {"string to float64", "3.14", reflect.TypeOf(float64(0)), float64(3.14), false},
            // bool variants
            {"string to bool true", "true", reflect.TypeOf(false), true, false},
            {"string to bool false", "false", reflect.TypeOf(false), false, false},
            {"string to bool 1", "1", reflect.TypeOf(false), true, false},
            {"string to bool 0", "0", reflect.TypeOf(false), false, false},
            // error cases
            {"invalid string to int", "abc", reflect.TypeOf(int(0)), nil, true},
            {"invalid string to float", "xyz", reflect.TypeOf(float64(0)), nil, true},
            {"invalid string to bool", "maybe", reflect.TypeOf(false), nil, true},
            {"negative string to uint", "-1", reflect.TypeOf(uint(0)), nil, true},
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                got, err := convertStringToType(tt.input, tt.targetType)
                if tt.wantErr {
                    assert.Error(t, err)
                    return
                }
                require.NoError(t, err)
                assert.Equal(t, tt.wantValue, got.Interface())
            })
        }
    }
    ```

    **Test 2: `TestCallFunction_StringArgs`** — `CallFunction` 経由の結合動作テスト

    ```go
    // Test target function
    func addInts(ctx context.Context, x int, y int) (int, error) {
        return x + y, nil
    }

    func TestCallFunction_StringArgs(t *testing.T) {
        meta, err := AnalyzeFunction(addInts, "addInts", "test add")
        require.NoError(t, err)
        // Apply param names
        meta.Args[0].Name = "x"
        meta.Args[1].Name = "y"

        tests := []struct {
            name    string
            args    map[string]interface{}
            want    int
            wantErr bool
        }{
            {
                name: "string values",
                args: map[string]interface{}{"x": "10", "y": "20"},
                want: 30,
            },
            {
                name: "float64 values (existing behavior)",
                args: map[string]interface{}{"x": float64(5), "y": float64(3)},
                want: 8,
            },
            {
                name: "int values (direct ConvertibleTo)",
                args: map[string]interface{}{"x": 7, "y": 3},
                want: 10,
            },
            {
                name:    "invalid string value",
                args:    map[string]interface{}{"x": "abc", "y": "20"},
                wantErr: true,
            },
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                results, err := CallFunction(context.Background(), meta, tt.args)
                if tt.wantErr {
                    assert.Error(t, err)
                    return
                }
                require.NoError(t, err)
                require.Len(t, results, 1)
                assert.Equal(t, tt.want, results[0])
            })
        }
    }
    ```

---

#### [MODIFY] [reflection.go](file:///c:/Users/yamya/myprog/kuniumi/reflection.go)
*   **Description**: `CallFunction` の型変換ロジックに `string` → 数値型/bool 変換を追加し、新規ヘルパー関数 `convertStringToType` を実装する。
*   **Technical Design**:

    **1. import に `strconv` を追加**:
    ```go
    import (
        "context"
        "fmt"
        "reflect"
        "strconv"  // 追加
    )
    ```

    **2. `CallFunction` の型変換分岐を拡張 (96〜109行目)**:

    現在のコード:
    ```go
    } else {
        if targetVal.Kind() == reflect.Float64 {
            // ... float64 handling ...
        } else {
            return nil, fmt.Errorf("cannot convert %v to %v", targetVal.Type(), targetType)
        }
    }
    ```

    変更後:
    ```go
    } else {
        if targetVal.Kind() == reflect.Float64 {
            // Special handling for JSON numbers (float64) to integer types
            switch targetType.Kind() {
            case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
                in = append(in, reflect.ValueOf(int64(targetVal.Float())).Convert(targetType))
            case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
                in = append(in, reflect.ValueOf(uint64(targetVal.Float())).Convert(targetType))
            default:
                return nil, fmt.Errorf("cannot convert %v to %v", targetVal.Type(), targetType)
            }
        } else if targetVal.Kind() == reflect.String {
            // String to numeric/bool conversion using strconv
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

    **3. 新規ヘルパー関数 `convertStringToType` をファイル末尾に追加**:

    ```go
    // convertStringToType converts a string value to the specified reflect.Type using strconv.
    // Supported target types: int*, uint*, float*, bool.
    func convertStringToType(s string, targetType reflect.Type) (reflect.Value, error) {
        switch targetType.Kind() {
        case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
            n, err := strconv.ParseInt(s, 10, targetType.Bits())
            if err != nil {
                return reflect.Value{}, fmt.Errorf("parsing %q as integer: %w", s, err)
            }
            return reflect.ValueOf(n).Convert(targetType), nil

        case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
            n, err := strconv.ParseUint(s, 10, targetType.Bits())
            if err != nil {
                return reflect.Value{}, fmt.Errorf("parsing %q as unsigned integer: %w", s, err)
            }
            return reflect.ValueOf(n).Convert(targetType), nil

        case reflect.Float32, reflect.Float64:
            n, err := strconv.ParseFloat(s, targetType.Bits())
            if err != nil {
                return reflect.Value{}, fmt.Errorf("parsing %q as float: %w", s, err)
            }
            return reflect.ValueOf(n).Convert(targetType), nil

        case reflect.Bool:
            b, err := strconv.ParseBool(s)
            if err != nil {
                return reflect.Value{}, fmt.Errorf("parsing %q as bool: %w", s, err)
            }
            return reflect.ValueOf(b), nil

        default:
            return reflect.Value{}, fmt.Errorf("unsupported conversion from string to %v", targetType)
        }
    }
    ```

*   **Logic**:
    *   `strconv.ParseInt(s, 10, targetType.Bits())`: 基数10で解析し、ターゲット型のビット幅に合わせてオーバーフローチェックを行う
    *   `strconv.ParseUint(s, 10, targetType.Bits())`: 符号なし整数として解析
    *   `strconv.ParseFloat(s, targetType.Bits())`: ターゲット精度に合わせて浮動小数点を解析
    *   `strconv.ParseBool(s)`: Go 標準の真偽値解析（`"1"`, `"t"`, `"TRUE"`, `"true"`, `"0"`, `"f"`, `"FALSE"`, `"false"` 等を許容）
    *   各 Parse 関数の結果を `reflect.ValueOf(n).Convert(targetType)` でターゲット型に変換

---

### 統合テスト

#### [MODIFY] [kuniumi_test.go](file:///c:/Users/yamya/myprog/kuniumi/tests/kuniumi/kuniumi_test.go)
*   **Description**: CGI モードで文字列数値入力のテストケースを追加する。
*   **Technical Design**:
    *   既存の `t.Run("CGI", ...)` テストの後に新しいサブテストを追加
*   **Logic**:

    ```go
    // Case 2c: CGI Mode with string numeric values
    t.Run("CGI/StringArgs", func(t *testing.T) {
        input := `{"x": "10", "y": "20"}`
        cmd := exec.Command(binPath, "cgi")
        cmd.Env = append(os.Environ(), "PATH_INFO=/Add", "REQUEST_METHOD=POST")
        cmd.Stdin = strings.NewReader(input)

        var out bytes.Buffer
        cmd.Stdout = &out
        cmd.Stderr = os.Stderr

        require.NoError(t, cmd.Run())

        output := out.String()
        assert.Contains(t, output, "Status: 200 OK")
        assert.Contains(t, output, `{"result":30}`)
    })
    ```

## Step-by-Step Implementation Guide

1.  **単体テストの作成 (Red)**:
    *   `reflection_test.go` を新規作成する。
    *   `TestConvertStringToType` と `TestCallFunction_StringArgs` の2つのテスト関数を記述する（上記 Proposed Changes の通り）。
    *   `./scripts/process/build.sh` を実行し、テストが **失敗する** ことを確認する（`convertStringToType` が未定義のためコンパイルエラー）。

2.  **`convertStringToType` ヘルパー関数の実装 (Green)**:
    *   `reflection.go` の import に `"strconv"` を追加する。
    *   `reflection.go` のファイル末尾に `convertStringToType` 関数を追加する（上記 Proposed Changes の通り）。
    *   `./scripts/process/build.sh` を実行し、`TestConvertStringToType` が成功することを確認する。`TestCallFunction_StringArgs` はまだ一部失敗する可能性がある（`CallFunction` がまだ `convertStringToType` を呼んでいないため）。

3.  **`CallFunction` の型変換ロジック拡張 (Green)**:
    *   `reflection.go` の 96〜109 行目の `else` ブロックに、`targetVal.Kind() == reflect.String` の分岐を追加する（上記 Proposed Changes の通り）。
    *   `./scripts/process/build.sh` を実行し、すべての単体テストが成功することを確認する。

4.  **統合テストの追加**:
    *   `tests/kuniumi/kuniumi_test.go` の `t.Run("CGI", ...)` ブロックの後に `t.Run("CGI/StringArgs", ...)` テストケースを追加する（上記 Proposed Changes の通り）。
    *   `./scripts/process/build.sh && ./scripts/process/integration_test.sh` を実行し、既存テスト・新規テストがすべて成功することを確認する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認事項**: `TestConvertStringToType` と `TestCallFunction_StringArgs` の全ケースが PASS すること。
    *   **回帰確認**: 既存のビルドが成功し、他のテストに影響がないこと。

2.  **Integration Tests**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```
    *   **確認事項**: `CGI/StringArgs` テストケースが PASS すること。
    *   **回帰確認**: 既存の `CGI`, `Serve/FunctionCall`, `VirtualEnv` テストケースが引き続き PASS すること。
    *   **Log Verification**: テスト出力に `FAIL` が含まれないこと。

## Documentation

影響を受ける既存ドキュメントはありません。`reflection.go` に追加する関数コメントが唯一のドキュメント更新です。
