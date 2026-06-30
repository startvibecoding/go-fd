// Command fd is a Go port of the `fd` file finder. It mirrors the original
// tool's command-line interface and behavior.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	gofd "github.com/startvibecoding/go-fd"
)

var version = "10.4.2-go"

func main() {
	os.Exit(int(run(os.Args[1:])))
}

func run(args []string) gofd.ExitCode {
	opts, baseDir, err := parseArgs(args)
	if err != nil {
		if err == errHelp {
			printHelp()
			return 0
		}
		if err == errVersion {
			fmt.Printf("fd %s\n", version)
			return 0
		}
		fmt.Fprintf(os.Stderr, "[fd error]: %v\n", err)
		return 1
	}

	// --base-directory changes the working directory before searching.
	if baseDir != "" {
		info, statErr := os.Stat(baseDir)
		if statErr != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "[fd error]: The '--base-directory' path '%s' is not a directory.\n", baseDir)
			return 1
		}
		if chErr := os.Chdir(baseDir); chErr != nil {
			fmt.Fprintf(os.Stderr, "[fd error]: Could not set '%s' as the current working directory\n", baseDir)
			return 1
		}
	}

	// Detect the common mistake of passing a path as the pattern. This check is
	// skipped for glob patterns, which may legitimately contain '/'.
	if !opts.FullPath && !opts.Glob && strings.Contains(opts.Pattern, "/") {
		fmt.Fprintf(os.Stderr, "[fd error]: The search pattern '%s' contains a path-separation character and will not lead to any search results.\n\n"+
			"If you want to search for all files inside the '%s' directory, use a match-all pattern:\n\n  fd . '%s'\n\n"+
			"Instead, if you want your pattern to match the full file path, use:\n\n  fd --full-path '%s'\n",
			opts.Pattern, opts.Pattern, opts.Pattern, opts.Pattern)
		return 1
	}

	paths, invalidPaths, err := gofd.ValidateSearchPaths(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[fd error]: %v\n", err)
		return 1
	}
	for _, p := range invalidPaths {
		fmt.Fprintf(os.Stderr, "[fd error]: Search path '%s' is not a directory.\n", p)
	}
	if len(paths) == 0 {
		fmt.Fprintf(os.Stderr, "[fd error]: No valid search paths given.\n")
		return 1
	}

	f, _, err := gofd.Compile(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[fd error]: %v\n", err)
		return 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	return f.Run(ctx, paths)
}

var (
	errHelp    = fmt.Errorf("help requested")
	errVersion = fmt.Errorf("version requested")
)

// parseArgs implements a hand-rolled parser for fd's CLI.
func parseArgs(args []string) (gofd.Options, string, error) {
	var opts gofd.Options
	opts.Color = "auto"
	opts.Hyperlink = "never"

	var positionals []string
	baseDir := ""
	unrestrictedCount := 0

	i := 0
	for i < len(args) {
		arg := args[i]
		i++

		// "--" terminates option parsing.
		if arg == "--" {
			positionals = append(positionals, args[i:]...)
			break
		}

		if strings.HasPrefix(arg, "--") {
			name := arg[2:]
			value := ""
			hasValue := false
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				value = name[eq+1:]
				name = name[:eq]
				hasValue = true
			}
			// next returns the inline value or consumes the next arg.
			next := func() (string, error) {
				if hasValue {
					return value, nil
				}
				if i >= len(args) {
					return "", fmt.Errorf("option '--%s' requires a value", name)
				}
				v := args[i]
				i++
				return v, nil
			}

			switch name {
			case "help":
				return opts, "", errHelp
			case "version":
				return opts, "", errVersion
			case "hidden":
				opts.Hidden = true
			case "no-hidden":
				opts.Hidden = false
			case "no-ignore":
				opts.NoIgnore = true
			case "ignore":
				opts.NoIgnore = false
			case "no-ignore-vcs":
				opts.NoIgnoreVcs = true
			case "no-require-git":
				opts.NoRequireGit = true
			case "require-git":
				opts.NoRequireGit = false
			case "no-ignore-parent":
				opts.NoIgnoreParent = true
			case "no-global-ignore-file":
				opts.NoGlobalIgnore = true
			case "unrestricted":
				unrestrictedCount++
			case "case-sensitive":
				opts.CaseSensitive = true
			case "ignore-case":
				opts.IgnoreCase = true
			case "glob":
				opts.Glob = true
			case "regex":
				opts.Glob = false
			case "fixed-strings", "literal":
				opts.FixedStrings = true
			case "exact":
				opts.Exact = true
			case "and":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Exprs = append(opts.Exprs, v)
			case "absolute-path":
				opts.AbsolutePath = true
			case "relative-path":
				opts.AbsolutePath = false
			case "list-details":
				opts.ListDetails = true
			case "follow", "dereference":
				opts.FollowLinks = true
			case "no-follow":
				opts.FollowLinks = false
			case "full-path":
				opts.FullPath = true
			case "print0":
				opts.NullSeparator = true
			case "max-depth", "maxdepth":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.MaxDepth, err = atoiPositive(v, "max-depth")
				if err != nil {
					return opts, "", err
				}
			case "min-depth", "mindepth":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.MinDepth, err = atoiPositive(v, "min-depth")
				if err != nil {
					return opts, "", err
				}
			case "exact-depth":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.ExactDepth, err = atoiPositive(v, "exact-depth")
				if err != nil {
					return opts, "", err
				}
			case "exclude":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Exclude = append(opts.Exclude, v)
			case "prune":
				opts.Prune = true
			case "type":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Types = append(opts.Types, normalizeType(v))
			case "extension":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Extensions = append(opts.Extensions, v)
			case "size":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Sizes = append(opts.Sizes, v)
			case "changed-within", "change-newer-than", "newer", "changed-after":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.ChangedWithin = v
			case "changed-before", "change-older-than", "older":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.ChangedBefore = v
			case "owner":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Owner = v
			case "format":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Format = v
			case "batch-size":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.BatchSize, err = atoiPositive(v, "batch-size")
				if err != nil {
					return opts, "", err
				}
			case "ignore-file":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.IgnoreFiles = append(opts.IgnoreFiles, v)
			case "color":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Color = v
			case "hyperlink", "hyper":
				// Optional value; defaults to "auto".
				if hasValue {
					opts.Hyperlink = value
				} else {
					opts.Hyperlink = "auto"
				}
			case "ignore-contain":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.IgnoreContain = append(opts.IgnoreContain, v)
			case "threads":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Threads, err = atoiPositive(v, "threads")
				if err != nil {
					return opts, "", err
				}
			case "max-results":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.MaxResults, err = atoiPositive(v, "max-results")
				if err != nil {
					return opts, "", err
				}
			case "base-directory":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				baseDir = v
			case "path-separator":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.PathSeparator = v
			case "search-path":
				v, err := next()
				if err != nil {
					return opts, "", err
				}
				opts.Paths = append(opts.Paths, v)
			case "strip-cwd-prefix":
				when := "always"
				if hasValue {
					when = value
				}
				b := when != "never"
				opts.StripCwdPrefix = &b
			case "one-file-system", "mount", "xdev":
				opts.OneFileSystem = true
			case "show-errors":
				opts.ShowErrors = true
			case "has-results":
				opts.Quiet = true
			case "quiet":
				opts.Quiet = true
			case "exec":
				rest, consumed := collectCommand(args[i:])
				opts.Exec = rest
				i += consumed
			case "exec-batch":
				rest, consumed := collectCommand(args[i:])
				opts.ExecBatch = rest
				i += consumed
			default:
				return opts, "", fmt.Errorf("unexpected argument '--%s'", name)
			}
			continue
		}

		if len(arg) > 1 && arg[0] == '-' && arg != "-" {
			// Short option cluster.
			newI, _, err := parseShort(arg[1:], args, i, &opts, &positionals, &baseDir, &unrestrictedCount)
			if err != nil {
				if err == errHelp {
					return opts, "", errHelp
				}
				if err == errVersion {
					return opts, "", errVersion
				}
				return opts, "", err
			}
			i = newI
			continue
		}

		// Positional argument.
		positionals = append(positionals, arg)
	}

	if unrestrictedCount > 0 {
		opts.Unrestricted = true
	}

	// First positional is the pattern, the rest are search paths.
	if len(positionals) > 0 {
		opts.Pattern = positionals[0]
		if len(opts.Paths) == 0 {
			opts.Paths = positionals[1:]
		} else {
			opts.Paths = append(opts.Paths, positionals[1:]...)
		}
	}

	return opts, baseDir, nil
}

// parseShort handles a cluster of short options like "-tf", "-LH", "-d3".
// Returns the updated args index.
func parseShort(cluster string, args []string, i int, opts *gofd.Options, positionals *[]string, baseDir *string, unrestricted *int) (int, bool, error) {
	for idx := 0; idx < len(cluster); idx++ {
		c := cluster[idx]
		// rest is the remainder of the cluster after this char.
		rest := cluster[idx+1:]
		// value pulls an attached value (rest) or the next argument.
		value := func() (string, error) {
			if rest != "" {
				return rest, nil
			}
			if i >= len(args) {
				return "", fmt.Errorf("option '-%c' requires a value", c)
			}
			v := args[i]
			i++
			return v, nil
		}
		switch c {
		case 'h':
			return i, false, errHelp
		case 'V':
			return i, false, errVersion
		case 'H':
			opts.Hidden = true
		case 'I':
			opts.NoIgnore = true
		case 'u':
			*unrestricted++
		case 's':
			opts.CaseSensitive = true
		case 'i':
			opts.IgnoreCase = true
		case 'g':
			opts.Glob = true
		case 'F':
			opts.FixedStrings = true
		case 'a':
			opts.AbsolutePath = true
		case 'l':
			opts.ListDetails = true
		case 'L':
			opts.FollowLinks = true
		case 'p':
			opts.FullPath = true
		case '0':
			opts.NullSeparator = true
		case 'q':
			opts.Quiet = true
		case '1':
			opts.MaxResults = 1
		case 'd':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			opts.MaxDepth, err = atoiPositive(v, "max-depth")
			return i, true, err
		case 'E':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			opts.Exclude = append(opts.Exclude, v)
			return i, true, nil
		case 't':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			opts.Types = append(opts.Types, normalizeType(v))
			return i, true, nil
		case 'e':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			opts.Extensions = append(opts.Extensions, v)
			return i, true, nil
		case 'S':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			opts.Sizes = append(opts.Sizes, v)
			return i, true, nil
		case 'o':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			opts.Owner = v
			return i, true, nil
		case 'c':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			opts.Color = v
			return i, true, nil
		case 'j':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			opts.Threads, err = atoiPositive(v, "threads")
			return i, true, err
		case 'C':
			v, err := value()
			if err != nil {
				return i, false, err
			}
			*baseDir = v
			return i, true, nil
		case 'x':
			rest, consumed := collectCommand(args[i:])
			opts.Exec = rest
			return i + consumed, true, nil
		case 'X':
			rest, consumed := collectCommand(args[i:])
			opts.ExecBatch = rest
			return i + consumed, true, nil
		default:
			return i, false, fmt.Errorf("unexpected argument '-%c'", c)
		}
	}
	return i, false, nil
}

// collectCommand gathers command arguments up to a ';' terminator. It returns
// the collected args and the number of input tokens consumed (including the
// terminator).
func collectCommand(rest []string) ([]string, int) {
	var cmd []string
	consumed := 0
	for _, a := range rest {
		consumed++
		if a == ";" {
			break
		}
		cmd = append(cmd, a)
	}
	return cmd, consumed
}

func normalizeType(v string) string {
	switch v {
	case "f", "file":
		return "f"
	case "d", "dir", "directory":
		return "d"
	case "l", "symlink":
		return "l"
	case "x", "executable":
		return "x"
	case "e", "empty":
		return "e"
	case "s", "socket":
		return "s"
	case "p", "pipe":
		return "p"
	case "b", "block-device":
		return "b"
	case "c", "char-device":
		return "c"
	default:
		return v
	}
}

func atoiPositive(s, name string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid value '%s' for '--%s'", s, name)
	}
	return n, nil
}
