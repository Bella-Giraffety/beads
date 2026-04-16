package doltdboverride

import "sync"

var (
	mu    sync.Mutex
	stack []string
)

// Push installs a temporary in-process Dolt database override.
// Call the returned restore func when the scoped override should end.
func Push(database string) func() {
	if database == "" {
		return func() {}
	}

	mu.Lock()
	stack = append(stack, database)
	mu.Unlock()

	return func() {
		mu.Lock()
		if n := len(stack); n > 0 {
			stack = stack[:n-1]
		}
		mu.Unlock()
	}
}

// Current returns the active override, if any.
func Current() string {
	mu.Lock()
	defer mu.Unlock()
	if len(stack) == 0 {
		return ""
	}
	return stack[len(stack)-1]
}
