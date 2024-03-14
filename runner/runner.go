package runner

type Runner interface {
	Run(command string, options ...Option) (string, int, error)
	PID() int
}
