package runner

// Listener represent command listener (it will send stdout fragments as thier being available on stdout)
type Listener func(stdout string, hasMore bool)
