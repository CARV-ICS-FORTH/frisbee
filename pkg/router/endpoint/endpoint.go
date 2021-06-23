package endpoint

// Handler is the glue between object and backend controller.
type Handler interface {
	Name() string

	// Apply invokes a command
	// Apply(ctx context.Context, obj InnerObject) error

	// Recover retracts the side effects of a command, if possible
	// Recover(ctx context.Context, obj InnerObject) error
}
