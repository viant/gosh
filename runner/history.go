package runner

type History struct {
	Commands []*Command
}

// NewHistory creates a command history
func NewHistory() *History {
	return &History{Commands: make([]*Command, 0)}
}
