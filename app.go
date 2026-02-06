package kuniumi

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// App is the main application structure for Kuniumi.
// It manages the lifecycle of the application, including configuration,
// function registration, command-line argument parsing, and environment management.
//
// App embeds a VirtualEnvironment to provide a consistent runtime across different
// execution modes (CLI, Server, etc.).
type App struct {
	config    Config
	functions []*RegisteredFunc
	rootCmd   *cobra.Command
	env       *VirtualEnvironment
}

// RegisteredFunc holds metadata about a registered function.
// It includes the function's name, description, and reflection metadata
// used for runtime invocation and schema generation.
type RegisteredFunc struct {
	Name        string
	Description string
	Meta        *FunctionMetadata
	paramDefs   []ParamDef
	returnDesc  string
}

// ArgNamesOption allows specifying argument names for a function.
// This is useful when reflection cannot deduce meaningful parameter names.
type ArgNamesOption struct {
	Names []string
}

// WithArgNames returns an Option that configures argument names.
// Note: This Option type is for App configuration, but usage here seems to imply
// function configuration. Use WithArgs for function registration options.
func WithArgNames(names ...string) Option {
	return func(a *App) {
		// No-op: argument names are handled via WithArgs passed to RegisterFunc.
	}
}

// FuncOption is a functional option for configuring a registered function.
// It is used with App.RegisterFunc to customize metadata such as argument names.
type FuncOption func(*RegisteredFunc)

// WithArgs returns a FuncOption that specifies custom names for function arguments.
//
// Example:
//
//	app.RegisterFunc(MyFunc, "Desc", kuniumi.WithArgs("param1", "param2"))
func WithArgs(names ...string) FuncOption {
	return func(rf *RegisteredFunc) {
		// Apply names to metadata
		if len(names) > len(rf.Meta.Args) {
			// warning?
		}
		for i, name := range names {
			if i < len(rf.Meta.Args) {
				rf.Meta.Args[i].Name = name
			}
		}
	}
}

// New creates a new Kuniumi application with the given configuration and options.
//
// Arguments:
//   - cfg: The initial configuration for the app (Name, Version).
//   - opts: A variadic list of Option functions to further customize the app.
//
// Returns a pointer to the initialized App.
func New(cfg Config, opts ...Option) *App {
	app := &App{
		config: cfg,
		rootCmd: &cobra.Command{
			Use:   strings.ToLower(cfg.Name),
			Short: fmt.Sprintf("%s v%s", cfg.Name, cfg.Version),
		},
	}

	for _, opt := range opts {
		opt(app)
	}

	// Setup Global Flags
	app.rootCmd.PersistentFlags().StringSlice("env", []string{}, "Environment variables (KEY=VALUE)")
	app.rootCmd.PersistentFlags().StringSlice("mount", []string{}, "Mount directories (HOST:VIRTUAL)")

	viper.BindPFlag("env", app.rootCmd.PersistentFlags().Lookup("env"))
	viper.BindPFlag("mount", app.rootCmd.PersistentFlags().Lookup("mount"))

	return app
}

// RegisterFunc registers a Go function to the application, exposing it as a tool/command.
//
// Arguments:
//   - fn: The function to register. It supports various signatures, but generally should accept
//     `context.Context` as the first argument and return `(any, error)` or similar.
//   - desc: A human-readable description of what the function does.
//   - opts: Optional functional options to customize metadata (e.g., argument names).
//
// Example:
//
//	app.RegisterFunc(MyFunc, "Does something useful", kuniumi.WithArgs("arg1", "arg2"))
//
// Panics if `fn` is not a function or if analysis fails.
func (a *App) RegisterFunc(fn interface{}, desc string, opts ...FuncOption) {
	// Extract function name
	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		panic("RegisterFunc: expected a function")
	}
	funcName := runtime.FuncForPC(val.Pointer()).Name()
	// Extract simple name (e.g. "main.Add" -> "Add")
	parts := strings.Split(funcName, ".")
	name := parts[len(parts)-1]
	// Handle method values which have -fm suffix
	name = strings.TrimSuffix(name, "-fm")

	meta, err := AnalyzeFunction(fn, name, desc)
	if err != nil {
		panic(fmt.Sprintf("RegisterFunc failed: %v", err))
	}

	// We can use that as default.
	// For now, let's just make it "func_N" unless user overrides or we implement the name extraction.
	// Better: Require name OR extract it.

	rf := &RegisteredFunc{
		Name:        meta.Name, // AnalyzeFunction sets this
		Description: desc,
		Meta:        meta,
	}

	// Apply options
	for _, opt := range opts {
		opt(rf)
	}

	// Apply Param definitions (Name and Description) to Meta by index
	// We assume params are provided in order.
	for i, pd := range rf.paramDefs {
		if i < len(meta.Args) {
			meta.Args[i].Name = pd.Name
			meta.Args[i].Description = pd.Desc
		}
	}

	// Apply Return description to Meta
	if rf.returnDesc != "" && len(meta.Returns) > 0 {
		// Currently only support single return value description (last error is ignored)
		meta.Returns[0].Description = rf.returnDesc
	}

	// Update Meta name if RF name changed
	meta.Name = rf.Name

	a.functions = append(a.functions, rf)
}

// Run executes the application.
// This generic entry point automatically handles CLI argument parsing and dispatches
// to the appropriate subcommand or mode.
//
// Capabilities:
//   - **serve**: Starts a clear Web API server.
//   - **mcp**: Runs as a Model Context Protocol server (stdio).
//   - **cgi**: Executes a single function in CGI mode (useful for serverless/hooks).
//   - **containerize**: (Experimental) Helps package the app.
//
// It also parses global flags like `--env` and `--mount` to initialize the Virtual Environment.
func (a *App) Run() error {
	// Initialize Virtual Environment from flags
	a.rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		envKV := make(map[string]string)
		mounts := make(map[string]string)

		// Parse --env
		envFlags, _ := cmd.Flags().GetStringSlice("env")
		for _, e := range envFlags {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envKV[parts[0]] = parts[1]
			}
		}

		// Parse --mount
		mountFlags, _ := cmd.Flags().GetStringSlice("mount")
		for _, m := range mountFlags {
			// Handle Windows paths (C:\foo:/bar)
			// Split by LAST colon to separate Host and Virtual path,
			// usually Virtual path starts with / and is on the right.
			lastColon := strings.LastIndex(m, ":")
			if lastColon > 0 {
				hostPath := m[:lastColon]
				virtualPath := m[lastColon+1:]
				mounts[hostPath] = virtualPath
			} else {
				// Fallback or error? For now fallback to simple split if only one colon found by LastIndex
				// If LastIndex == -1, no colon.
			}
		}

		a.env = NewVirtualEnvironment(envKV, mounts)
	}

	// Add subcommands
	a.rootCmd.AddCommand(a.buildServeCmd())
	a.rootCmd.AddCommand(a.buildCgiCmd())
	a.rootCmd.AddCommand(a.buildMcpCmd())
	a.rootCmd.AddCommand(a.buildContainerizeCmd())

	return a.rootCmd.Execute()
}

// ContextWithEnv returns a new context with the application's VirtualEnvironment attached.
// This context should be passed to handler functions so they can access the environment
// via `kuniumi.GetVirtualEnv(ctx)`.
func (a *App) ContextWithEnv(ctx context.Context) context.Context {
	if a.env == nil {
		// Should not happen if PreRun executed, but for safety
		return ctx
	}
	return WithVirtualEnv(ctx, a.env)
}
