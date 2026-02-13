# 007-FrameworkVersionBanner

> **Source Specification**: [007-FrameworkVersionBanner.md](file:///c:/Users/yamya/myprog/kuniumi/prompts/phases/000-firstrelease/ideas/main/007-FrameworkVersionBanner.md)

## Goal Description

CLI バナーに Kuniumi フレームワークのバージョンとコミットIDを表示する。`runtime/debug.ReadBuildInfo()` を使用してビルド情報を自動取得し、`Calculator v1.0.0 (based on kuniumi v0.1.1 abc1234)` のような形式で表示する。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| バナーにフレームワーク情報を表示 | Proposed Changes > `app.go` > `New()` |
| ビルド情報の自動取得 (`ReadBuildInfo`) | Proposed Changes > `version.go` > `frameworkVersionString()` |
| フォールバック動作 (dev 表示) | Proposed Changes > `version.go` > `frameworkVersionString()` |
| VCS dirty フラグ | Proposed Changes > `version.go` > `frameworkVersionString()` |

## Proposed Changes

### kuniumi (ルートパッケージ)

#### [NEW] [version_test.go](file:///c:/Users/yamya/myprog/kuniumi/version_test.go)
*   **Description**: `frameworkVersionString()` の単体テストを作成する（TDD: テスト先行）。
*   **Technical Design**:
    *   パッケージ: `package kuniumi`（内部関数をテストするため同一パッケージ）
*   **Logic**:

    ```go
    func TestFrameworkVersionString(t *testing.T) {
        result := frameworkVersionString()
        // frameworkVersionString は常に文字列を返す
        // テスト実行時（go test）は ReadBuildInfo が利用可能なので
        // "based on kuniumi" を含む文字列が返る
        assert.Contains(t, result, "based on kuniumi")
        // "dev" またはバージョン番号が含まれる
        assert.Regexp(t, `based on kuniumi (v[\d.]+|dev)`, result)
    }
    ```

---

#### [NEW] [version.go](file:///c:/Users/yamya/myprog/kuniumi/version.go)
*   **Description**: フレームワークバージョン情報を取得するパッケージレベル関数を提供する。
*   **Technical Design**:
    ```go
    package kuniumi

    import (
        "fmt"
        "runtime/debug"
    )

    // frameworkVersionString returns a string describing the kuniumi framework version
    // and the application's VCS commit information.
    // Format: "based on kuniumi <version> <commit>"
    // Fallback: "based on kuniumi dev" if build info is unavailable.
    func frameworkVersionString() string {
        info, ok := debug.ReadBuildInfo()
        if !ok {
            return "based on kuniumi dev"
        }

        // Get kuniumi module version from dependencies
        fwVersion := "dev"
        for _, dep := range info.Deps {
            if dep.Path == "github.com/axsh/kuniumi" {
                fwVersion = dep.Version
                break
            }
        }

        // If this IS the kuniumi module itself (during development),
        // use the main module version
        if fwVersion == "dev" && info.Main.Path == "github.com/axsh/kuniumi" {
            if info.Main.Version != "" && info.Main.Version != "(devel)" {
                fwVersion = info.Main.Version
            }
        }

        // Get VCS revision and modified status from build settings
        var revision string
        var modified bool
        for _, s := range info.Settings {
            switch s.Key {
            case "vcs.revision":
                if len(s.Value) > 7 {
                    revision = s.Value[:7]
                } else {
                    revision = s.Value
                }
            case "vcs.modified":
                modified = s.Value == "true"
            }
        }

        // Build the version string
        result := fmt.Sprintf("based on kuniumi %s", fwVersion)
        if revision != "" {
            result += " " + revision
            if modified {
                result += "-dirty"
            }
        }

        return result
    }
    ```

*   **Logic**:
    1. `debug.ReadBuildInfo()` を呼び出し、ビルド情報を取得する
    2. `info.Deps` を走査して `github.com/axsh/kuniumi` モジュールの `Version` を取得する
    3. 見つからない場合（Kuniumi 自体の開発中など）、`info.Main` を確認する
    4. `info.Settings` から `vcs.revision`（先頭7文字に切り詰め）と `vcs.modified` を取得する
    5. `"based on kuniumi <version> <revision>[-dirty]"` 形式の文字列を組み立てて返す
    6. ビルド情報が取得できない場合は `"based on kuniumi dev"` を返す

---

#### [MODIFY] [app.go](file:///c:/Users/yamya/myprog/kuniumi/app.go)
*   **Description**: `New()` 関数のバナー文字列に `frameworkVersionString()` の出力を追加する。
*   **Technical Design**:

    変更箇所: 88行目

    変更前:
    ```go
    Short: fmt.Sprintf("%s v%s", cfg.Name, cfg.Version),
    ```

    変更後:
    ```go
    Short: fmt.Sprintf("%s v%s (%s)", cfg.Name, cfg.Version, frameworkVersionString()),
    ```

*   **Logic**: `frameworkVersionString()` は常に文字列を返すため（フォールバックで `"based on kuniumi dev"`）、nil チェックは不要

---

### 統合テスト

#### [MODIFY] [kuniumi_test.go](file:///c:/Users/yamya/myprog/kuniumi/tests/kuniumi/kuniumi_test.go)
*   **Description**: 既存の `Help` テストケースに、バナーにフレームワーク情報が含まれることのアサーションを追加する。
*   **Technical Design**:

    変更箇所: 97行目付近

    追加するアサーション:
    ```go
    assert.Contains(t, string(out), "based on kuniumi")
    ```

*   **Logic**: `--help` 出力に `"based on kuniumi"` が含まれることを確認する。バージョン番号やコミットIDの具体的な値はビルド環境に依存するため、プレフィックスのみチェックする。

## Step-by-Step Implementation Guide

1.  **単体テストの作成 (Red)**:
    *   `version_test.go` を新規作成する。
    *   `TestFrameworkVersionString` を記述する（上記 Proposed Changes の通り）。
    *   `./scripts/process/build.sh` を実行し、テストが **失敗する** ことを確認する（`frameworkVersionString` が未定義のためコンパイルエラー）。

2.  **`version.go` の実装 (Green)**:
    *   `version.go` を新規作成する。
    *   `frameworkVersionString()` 関数を実装する（上記 Proposed Changes の通り）。
    *   `./scripts/process/build.sh` を実行し、`TestFrameworkVersionString` が成功することを確認する。

3.  **`app.go` のバナー文字列を変更**:
    *   `app.go` 88行目の `Short` フィールドを変更する（上記 Proposed Changes の通り）。
    *   `./scripts/process/build.sh` を実行し、ビルドが成功することを確認する。

4.  **統合テストの更新**:
    *   `tests/kuniumi/kuniumi_test.go` の `Help` テストに `assert.Contains(t, string(out), "based on kuniumi")` を追加する。
    *   `./scripts/process/build.sh && ./scripts/process/integration_test.sh` を実行し、すべてのテストが成功することを確認する。

## Verification Plan

### Automated Verification

1.  **Build & Unit Tests**:
    ```bash
    ./scripts/process/build.sh
    ```
    *   **確認事項**: `TestFrameworkVersionString` が PASS すること。
    *   **回帰確認**: 既存の全テストが引き続き PASS すること。

2.  **Integration Tests**:
    ```bash
    ./scripts/process/build.sh && ./scripts/process/integration_test.sh
    ```
    *   **確認事項**: `Help` テストケースが PASS し、`"based on kuniumi"` がバナーに含まれること。
    *   **回帰確認**: 全統合テストが引き続き PASS すること。
    *   **Log Verification**: テスト出力に `FAIL` が含まれないこと。

## Documentation

影響を受ける既存ドキュメントはありません。`version.go` の関数コメントが唯一のドキュメント更新です。
