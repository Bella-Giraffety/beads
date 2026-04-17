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

// Replace swaps the entire override stack for the duration of a scoped
// operation. This is used when opening a different workspace's store so the
// caller can avoid leaking the current workspace's redirect-derived database
// selection into the routed open.
func Replace(database string) func() {
	mu.Lock()
	prev := append([]string(nil), stack...)
	if database == "" {
		stack = nil
	} else {
		stack = []string{database}
	}
	mu.Unlock()

	return func() {
		mu.Lock()
		stack = prev
		mu.Unlock()
	}
}
