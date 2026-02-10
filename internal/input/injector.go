package input

// Injector injects input events into the system.
type Injector interface {
	Inject(event *InputEvent) error
}
