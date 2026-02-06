# Kuniumi Framework リファレンスマニュアル

## 1. 概要 (Overview)

**Kuniumi** (国生み) は、Go言語で記述された関数を、様々なインターフェース（HTTP, CGI, MCP, Dockerコンテナ）を通じてポータブルなWebサービスとして公開するためのフレームワークです。

開発者はビジネスロジック（関数）の実装に集中し、Kuniumiが実行環境への適応（Adapter）やファイルシステム/環境変数の抽象化（Virtual Environment）を担当します。

### 主な特徴
- **Write Once, Run Anywhere**: ひとつのGoコードから、HTTPサーバー、MCPサーバー、CGIスクリプト、Dockerコンテナを生成可能。
- **Virtual Environment**: ホストOSのファイルシステムや環境変数を直接操作せず、サンドボックス化された仮想環境を通じて安全にアクセス。
- **Type Safety**: Goの静的型付けを活かし、Reflectionを用いて関数のメタデータを自動抽出。

## 2. クイックスタート (Getting Started)

### インストール

```bash
go get github.com/axsh/kuniumi
```

### 基本的な実装 (`main.go`)

```go
package main

import (
    "context"
    "github.com/axsh/kuniumi"
)

// 公開したい関数
// context.Context を第一引数に取ることで VirtualEnvironment にアクセス可能
func Add(ctx context.Context, x int, y int) (int, error) {
    return x + y, nil
}

func main() {
    // アプリケーションの初期化
    app := kuniumi.New(kuniumi.Config{
        Name:    "Calculator",
        Version: "1.0.0",
    })

    // 関数の登録
    // WithArgs で引数名を指定することで、JSON入力などのキーとして使用される
    app.RegisterFunc(Add, "2つの整数を加算します", kuniumi.WithArgs("x", "y"))

    // 実行（CLI引数に基づいて適切なアダプターが起動）
    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

### ビルドと実行

```bash
go build -o calculator main.go

# ヘルプ表示
./calculator --help

# HTTPサーバーとして起動
./calculator serve --port 8080

# MCPサーバーとして起動 (Stdio通信)
./calculator mcp

# CGIモードで実行 (環境変数や標準入力が必要)
export PATH_INFO="/Add"
echo '{"x": 10, "y": 20}' | ./calculator cgi

# Dockerfileの生成
./calculator containerize
```

## 3. コアコンセプト (Core Concepts)

### 3.1 アプリケーション (App)

`kuniumi.App` はフレームワークの中心となる構造体で、設定 (`Config`) と 登録された関数 (`RegisteredFunc`) を管理します。`cobra` を内部で使用しており、強力なCLI機能を提供します。

### 3.2 関数登録 (Function Registration)

`app.RegisterFunc` を使用して関数を登録します。

- **自動名前解決**: `runtime.FuncForPC` を使用して関数名（例: `Add`）を自動的に抽出します。
- **引数名の指定**: Goのリフレクションでは引数名（変数名）を取得できないため、`kuniumi.WithArgs("arg1", "arg2")` オプションを使用して明示的に指定することを推奨します。指定しない場合、自動生成された名前が使用される可能性がありますが、APIの可読性が低下します。
- **メタデータ**: 関数のシグネチャ（引数の型、戻り値）とDocstringがメタデータとして解析され、MCPのツール定義やOpenAPIスペックの生成に使用されます。

### 3.3 仮想環境 (Virtual Environment)

Kuniumiは、ポータビリティとセキュリティを高めるために「仮想環境 (`VirtualEnvironment`)」を提供します。関数内からは、ホストOSのファイルシステムを直接触るのではなく、この仮想環境を通じて操作を行います。

#### アクセス方法

```go
func MyFunc(ctx context.Context) error {
    // Contextから仮想環境を取得
    env := kuniumi.GetVirtualEnv(ctx)
    
    // 環境変数の取得
    debug := env.Getenv("DEBUG")
    
    // ファイル書き込み (仮想パスを指定)
    err := env.WriteFile("output.txt", []byte("data"))
    return err
}
```

#### マウント機能 (`--mount`)

実行時に `--mount` フラグを使用することで、ホストOSのディレクトリを仮想環境内のパスにマッピングできます。

```bash
# Windows
./app serve --mount "C:\Users\data:/data"

# Linux/Mac
./app serve --mount "/home/user/data:/data"
```

この例では、関数内で `/data/file.txt` にアクセスすると、実際にはホストの `C:\Users\data\file.txt` (Windows) や `/home/user/data/file.txt` (Linux) にアクセスします。

**特徴:**
- **Windowsパス対応**: Windowsのドライブレター（`C:\`等）を含むパスも正しく処理されます。
- **パスの正規化**: 仮想環境内では常に `/` 区切りのパスを使用します。
- **サンドボックス**: マウントされていないパスへのアクセスはエラーになります。

## 4. アダプター (Adapters)

Kuniumiは以下の実行モード（サブコマンド）を標準でサポートしています。

| モード (Command) | 説明 | 用途 |
| :--- | :--- | :--- |
| **`serve`** | HTTPサーバー | REST APIとしての公開、ローカルテスト |
| **`mcp`** | MCPサーバー | Claude DesktopやCursorなどのAIエージェントとの連携 |
| **`cgi`** | CGI実行 | サーバーレス環境 (AWS Lambda Custom Runtime等) や既存Webサーバー配下での実行 |
| **`containerize`** | Dockerfile生成 | アプリケーションのコンテナ化支援 |

### 4.1 MCP アダプター

Model Context Protocol (MCP) に準拠したサーバーとして動作します。標準入出力 (Stdio) を使用してクライアントと通信します。

- 登録された関数はMCPの「Tool」として公開されます。
- 関数の引数はJSON Schemaとして定義され、LLMが理解可能な形式になります。

### 4.2 CGI アダプター

環境変数 (`PATH_INFO`) と標準入力 (JSON Body) を使用して、一度きりの関数実行を行います。

- ルーティング: `PATH_INFO` (例: `/Add` や `/functions/Add`) に基づいて実行する関数を決定します。
- 入力: 標準入力からJSONを受け取ります。
- 出力: 標準出力にJSON形式のレスポンスとHTTPヘッダー風のメタデータを書き込みます。

## 5. API リファレンス (主な型と関数)

### `package kuniumi`

#### `func New(cfg Config, opts ...Option) *App`

新しいアプリケーションインスタンスを初期化します。

- **引数**:
    - `cfg`: アプリケーションの構成情報（名前、バージョンなど）。
    - `opts`: 追加のオプション（現在は未使用）。
- **戻り値**: 初期化された `*App` インスタンス。
- **例**:
    ```go
    app := kuniumi.New(kuniumi.Config{
        Name:    "MyApp",
        Version: "1.0.0",
    })
    ```

#### `func (a *App) RegisterFunc(fn interface{}, desc string, opts ...FuncOption)`

関数をアプリケーションに登録し、公開可能な状態にします。

- **引数**:
    - `fn`: 公開するGo関数。第一引数に `context.Context`、最後の戻り値に `error` を持つ必要があります。
    - `desc`: 関数の説明（Docstring）。LLMやOpenAPIのDescriptionとして使用されます。
    - `opts`: 関数のその他のオプション（`WithArgs` など）。
- **例**:
    ```go
    app.RegisterFunc(MyFunc, "サンプルの関数です",
        kuniumi.WithParams(
            kuniumi.Param("param1", "パラメータ1の説明"),
            kuniumi.Param("param2", "パラメータ2の説明"),
        ),
        kuniumi.WithReturns("戻り値の説明"),
    )
    ```

#### `func WithParams(params ...ParamDef) FuncOption`

関数の引数名と説明文を定義します。
`kuniumi.Param(name, desc)` を使用して、引数の順序通りに定義を作成します。

- **引数**:
    - `params`: `ParamDef` のリスト。
- **例**:
    ```go
    kuniumi.WithParams(
        kuniumi.Param("x", "X coordinate"),
        kuniumi.Param("y", "Y coordinate"),
    )
    ```

#### `func WithReturns(desc string) FuncOption`

関数の戻り値に関する説明文を定義します。

#### `func WithArgs(names ...string) FuncOption`

(非推奨: `WithParams` の使用を推奨)
登録する関数の引数名を指定するためのオプションです。説明文は付与されません。

#### `func (a *App) Run() error`

アプリケーションを実行します。内部でCLI引数を解析し、適切なサブコマンド（`serve`, `mcp`, `cgi` 等）を起動します。
通常、`main` 関数の最後に呼び出します。

#### `type VirtualEnvironment`

仮想環境（ファイルシステム、環境変数）へのアクセスを提供する構造体です。
直接インスタンス化せず、`GetVirtualEnv` を通じて取得します。

**主なメソッド**:

- **`func (v *VirtualEnvironment) Getenv(key string) string`**
    - 指定されたキーの環境変数を取得します。
    - 仮想環境独自の変数を優先し、設定がない場合は空文字を返します（将来的にホスト環境変数へのフォールバック等を設定可能になる可能性があります）。

- **`func (v *VirtualEnvironment) WriteFile(path string, data []byte) error`**
    - 仮想パス `path` にファイルを作成（上書き）します。
    - ホスト上のマウントされたパスに解決されて書き込まれます。

- **`func (v *VirtualEnvironment) ReadFile(path string, offset, length int64) ([]byte, error)`**
    - 仮想パス `path` からデータを読み込みます。
    - `offset` と `length` を指定して部分読み込みが可能です。`length` が0以上の場合はその長さだけ、ファイル全体の読み込みには標準的なラッパーが必要になる場合がありますが、現状は低レベルAPIとして提供されています。

- **`func (v *VirtualEnvironment) ListFile(path string) ([]FileInfo, error)`**
    - 指定されたディレクトリ内のファイル一覧を取得します。

- **`func (v *VirtualEnvironment) FindFile(root string, pattern string, recursive bool) ([]string, error)`**
    - 指定されたパターン（Glob）に一致するファイルを検索します。

- **`func (v *VirtualEnvironment) ChangeCurrentDirectory(path string) error`**
    - 仮想環境内のカレントディレクトリを変更します。

- **`func (v *VirtualEnvironment) GetCurrentDirectory() string`**
    - 現在のカレントディレクトリ（仮想パス）を取得します。

- **`func (v *VirtualEnvironment) RewriteFile(path string, offset int64, data []byte) error`**
    - 指定されたオフセットの位置からデータを上書きします。ファイルの他の部分は保持されます。

- **`func (v *VirtualEnvironment) CopyFile(src, dst string) error`**
    - 仮想パス `src` から `dst` へファイルをコピーします。

- **`func (v *VirtualEnvironment) RemoveFile(path string) error`**
    - 指定された仮想パスのファイルを削除します。

- **`func (v *VirtualEnvironment) Chmod(path string, mode os.FileMode) error`**
    - 指定された仮想パスのファイルのパーミッションを変更します。

- **`func (v *VirtualEnvironment) ListEnv() map[string]string`**
    - 現在の仮想環境変数の一覧（コピー）を取得します。

#### `func GetVirtualEnv(ctx context.Context) *VirtualEnvironment`

`context.Context` から現在の `VirtualEnvironment` を取得します。
コンテキスト内に環境が存在しない場合（テスト時など）、安全なデフォルト（空の環境）を返します。

- **例**:
    ```go
    func MyFunc(ctx context.Context) error {
        env := kuniumi.GetVirtualEnv(ctx)
        val := env.Getenv("MY_VAR")
        // ...
    }
    ```
