package kuniumi

// Config holds the application configuration.
// It defines the metadata for the application, such as its name and version,
// which are used in CLI help messages and Open API specifications.
type Config struct {
	// Name is the name of the application.
	Name string
	// Version is the version of the application.
	Version string
}

// Option is a functional option for configuring the App during initialization.
// It is used with the New function to customize the App instance.
type Option func(*App)

// WithName sets the application name.
func WithName(name string) Option {
	return func(a *App) {
		a.config.Name = name
	}
}
