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
    "fmt"
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
    // WithParams で引数名と説明を指定し、WithReturns で戻り値の説明を指定
    app.RegisterFunc(Add, "2つの整数を加算します",
        kuniumi.WithParams(
            kuniumi.Param("x", "1つ目の整数"),
            kuniumi.Param("y", "2つ目の整数"),
        ),
        kuniumi.WithReturns("加算結果"),
    )

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
- **メタデータの補完**: Goのリフレクションでは引数名や説明文を取得できないため、`kuniumi.WithParams` 等のオプションを使用して明示的に指定することを推奨します。
    - `WithParams` を使用しない場合でも動作しますが、生成されるAPIドキュメント（OpenAPI, MCP Tools）の品質が低下します。
- **メタデータ解析**: 関数のシグネチャ（引数の型、戻り値）とDocstringがメタデータとして解析され、MCPのツール定義やOpenAPIスペックの生成に使用されます。

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

### 4. アダプター (Adapters)

Kuniumiは以下の実行モード（サブコマンド）を標準でサポートしています。

| モード (Command) | 説明 | 用途 |
| :--- | :--- | :--- |
| **`serve`** | HTTPサーバー | REST APIとしての公開、ローカルテスト |
| **`mcp`** | MCPサーバー | Claude DesktopやCursorなどのAIエージェントとの連携 |
| **`cgi`** | CGI実行 | サーバーレス環境 (AWS Lambda Custom Runtime等) や既存Webサーバー配下での実行 |
| **`containerize`** | Dockerfile生成 | アプリケーションのコンテナ化支援 |

### 4.1 HTTP アダプター (`serve`)

REST APIとして関数を公開します。

- **Endpoint**: `POST /functions/{function_name}`
    - Body: JSON Object (Arguments)
    - Response: JSON Object (`{"result": ...}`)
- **Metadata**: `GET /openapi.json`
    - OpenAPI 3.0.0 形式で API 定義を返します。

### 4.2 MCP アダプター (`mcp`)

Model Context Protocol (MCP) に準拠したサーバーとして動作します。標準入出力 (Stdio) を使用してクライアントと通信します。

- 登録された関数はMCPの「Tool」として公開されます。
- 関数の引数はJSON Schemaとして定義され、LLMが理解可能な形式になります。

### 4.3 CGI アダプター (`cgi`)

環境変数 (`PATH_INFO`) と標準入力 (JSON Body) を使用して、一度きりの関数実行を行います。

- **ルーティング**: `PATH_INFO` (例: `/Add` や `/functions/Add`) に基づいて実行する関数を決定します。
- **入力**: 標準入力からJSONを受け取ります。
- **出力**: 標準出力にJSON形式のレスポンスとHTTPヘッダー風のメタデータを書き込みます。

### 4.4 Container アダプター (`containerize`)

アプリケーションを Docker コンテナ化するための `Dockerfile` を生成、ビルド、プッシュします。

- **Base Image**: `golang:1.24-alpine` (Builder), `alpine:latest` (Runtime)
- **Commands**:
    - `docker build`
    - `docker push` (Optional)

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
    - `opts`: 関数のその他のオプション（`WithParams` など）。
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

#### `RegisterFunc` Options

- **`kuniumi.WithParams(params ...ParamDef)`**
    - 関数の引数名と説明を指定します。
    - `kuniumi.Param(name, desc)` で定義を作成します。
- **`kuniumi.WithReturns(desc string)`**
    - 関数の戻り値の説明を指定します。

#### `type FunctionMetadata`

```go
type FunctionMetadata struct {
    Name        string
    Description string
    Args        []ArgMetadata
    Returns     []ArgMetadata // Currently usually size 1
}
```

### `type VirtualEnvironment`

#### `type FileInfo`

```go
type FileInfo struct {
    Name  string
    Size  int64
    IsDir bool
}
```

#### Methods

`kuniumi.GetVirtualEnv(ctx)` で取得したインスタンスに対して呼び出します。

- **`func (v *VirtualEnvironment) Getenv(key string) string`**
    - 指定された環境変数の値を取得します。
- **`func (v *VirtualEnvironment) ListEnv() map[string]string`**
    - 全環境変数のコピーを取得します。
- **`func (v *VirtualEnvironment) ReadFile(path string, offset int64, length int64) ([]byte, error)`**
    - 仮想パス上のファイルを読み込みます。
- **`func (v *VirtualEnvironment) WriteFile(path string, data []byte) error`**
    - 仮想パス上のファイルにデータを書き込みます（上書き）。
- **`func (v *VirtualEnvironment) RewriteFile(path string, offset int64, data []byte) error`**
    - ファイルの特定の位置からデータを書き込みます（部分更新）。
- **`func (v *VirtualEnvironment) CopyFile(src, dst string) error`**
    - ファイルをコピーします。
- **`func (v *VirtualEnvironment) RemoveFile(path string) error`**
    - ファイルを削除します。
- **`func (v *VirtualEnvironment) Chmod(path string, mode os.FileMode) error`**
    - ファイルの権限を変更します。
- **`func (v *VirtualEnvironment) ListFile(path string) ([]FileInfo, error)`**
    - 指定されたディレクトリ内のファイル一覧を取得します。
- **`func (v *VirtualEnvironment) FindFile(root string, pattern string, recursive bool) ([]string, error)`**
    - 指定されたパターン（Glob, 例: `*.go`）に一致するファイルを検索します。
    - `recursive` が `true` の場合、サブディレクトリも再帰的に検索します。
    - 戻り値は仮想パスのリストです。
- **`func (v *VirtualEnvironment) ChangeCurrentDirectory(path string) error`**
    - カレントディレクトリを変更します。
- **`func (v *VirtualEnvironment) GetCurrentDirectory() string`**
    - カレントディレクトリを取得します。

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
