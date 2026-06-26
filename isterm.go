package gofd

import "os"

// isTerminal reports whether f refers to an interactive terminal. It uses a
// portable heuristic based on the file mode (character device), avoiding extra
// dependencies.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
