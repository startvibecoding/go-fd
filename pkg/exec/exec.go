// Package exec implements fd's command execution feature (-x/--exec and
// -X/--exec-batch), including placeholder substitution and batching.
package exec

import (
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"sync"

	"github.com/startvibecoding/go-fd/pkg/format"
)

// Mode selects between per-result and batch execution.
type Mode int

const (
	// OneByOne runs the command once per search result.
	OneByOne Mode = iota
	// Batch runs the command once with all results as arguments.
	Batch
)

// commandTemplate is a single command (a sequence of argument templates).
type commandTemplate struct {
	args []*format.Template
}

func newCommandTemplate(input []string) (*commandTemplate, error) {
	if len(input) == 0 {
		return nil, errors.New("No executable provided for --exec or --exec-batch")
	}
	args := make([]*format.Template, 0, len(input)+1)
	hasPlaceholder := false
	for _, a := range input {
		t := format.Parse(a)
		if t.HasTokens() {
			hasPlaceholder = true
		}
		args = append(args, t)
	}
	if !hasPlaceholder {
		args = append(args, format.PlaceholderTemplate())
	}
	return &commandTemplate{args: args}, nil
}

func (c *commandTemplate) numberOfTokens() int {
	count := 0
	for _, a := range c.args {
		if a.HasTokens() {
			count++
		}
	}
	return count
}

// CommandSet is a collection of command templates run in one of the two modes.
type CommandSet struct {
	mode     Mode
	commands []*commandTemplate
}

// NewCommandSet builds a one-by-one command set. Each element of input is a
// command (a list of arguments).
func NewCommandSet(input [][]string) (*CommandSet, error) {
	cs := &CommandSet{mode: OneByOne}
	for _, cmd := range input {
		t, err := newCommandTemplate(cmd)
		if err != nil {
			return nil, err
		}
		cs.commands = append(cs.commands, t)
	}
	return cs, nil
}

// NewBatchCommandSet builds a batch command set.
func NewBatchCommandSet(input [][]string) (*CommandSet, error) {
	cs := &CommandSet{mode: Batch}
	for _, cmd := range input {
		t, err := newCommandTemplate(cmd)
		if err != nil {
			return nil, err
		}
		if t.numberOfTokens() > 1 {
			return nil, errors.New("Only one placeholder allowed for batch commands")
		}
		if t.args[0].HasTokens() {
			return nil, errors.New("First argument of exec-batch is expected to be a fixed executable")
		}
		cs.commands = append(cs.commands, t)
	}
	return cs, nil
}

// InBatchMode reports whether the set runs in batch mode.
func (cs *CommandSet) InBatchMode() bool { return cs.mode == Batch }

// Execute runs all commands for a single input path. Returns true on success.
func (cs *CommandSet) Execute(input string, pathSeparator string, bufferOutput bool) bool {
	ok := true
	for _, c := range cs.commands {
		argv := make([]string, len(c.args))
		for i, a := range c.args {
			argv[i] = a.Generate(input, pathSeparator)
		}
		if !runCommand(argv, bufferOutput) {
			ok = false
		}
	}
	return ok
}

// ExecuteBatch runs the command(s) once (or in chunks of limit) with all paths.
func (cs *CommandSet) ExecuteBatch(paths []string, limit int, pathSeparator string) bool {
	ok := true
	for _, c := range cs.commands {
		var pre []string
		var pathTmpl *format.Template
		var post []string
		for _, a := range c.args {
			if a.HasTokens() {
				pathTmpl = a
			} else if pathTmpl == nil {
				pre = append(pre, a.Generate("", ""))
			} else {
				post = append(post, a.Generate("", ""))
			}
		}
		if pathTmpl == nil {
			pathTmpl = format.PlaceholderTemplate()
		}

		chunks := [][]string{}
		if limit <= 0 {
			chunks = append(chunks, paths)
		} else {
			for i := 0; i < len(paths); i += limit {
				end := i + limit
				if end > len(paths) {
					end = len(paths)
				}
				chunks = append(chunks, paths[i:end])
			}
		}
		for _, chunk := range chunks {
			if len(chunk) == 0 {
				continue
			}
			argv := append([]string{}, pre...)
			for _, p := range chunk {
				argv = append(argv, pathTmpl.Generate(p, pathSeparator))
			}
			argv = append(argv, post...)
			if !runCommand(argv, false) {
				ok = false
			}
		}
	}
	return ok
}

var outputMu sync.Mutex

func runCommand(argv []string, bufferOutput bool) bool {
	if len(argv) == 0 {
		return false
	}
	cmd := osexec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	if bufferOutput {
		out, err := cmd.CombinedOutput()
		outputMu.Lock()
		os.Stdout.Write(out)
		outputMu.Unlock()
		if err != nil {
			reportErr(argv[0], err)
			return false
		}
		return true
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		reportErr(argv[0], err)
		return false
	}
	return true
}

func reportErr(program string, err error) {
	var execErr *osexec.Error
	if errors.As(err, &execErr) && errors.Is(execErr.Err, osexec.ErrNotFound) {
		fmt.Fprintf(os.Stderr, "[fd error]: Command not found: %s\n", program)
		return
	}
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		return // non-zero exit, already reflected in return value
	}
	fmt.Fprintf(os.Stderr, "[fd error]: Problem while executing command: %v\n", err)
}
