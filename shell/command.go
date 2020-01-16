package shell

// Command is an interface which functions as wrapper for the application
type Command interface {
	Name() string
	Exists() bool
	Version() string
}
