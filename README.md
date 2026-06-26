# go-fd

A pure-Go port of [`fd`](https://github.com/sharkdp/fd) — a simple, fast and
user-friendly alternative to `find`. It provides both a CLI tool (`fd`) that
mirrors the original's interface and a Go SDK for programmatic use.

[![Go](https://img.shields.io/badge/Go%201.21+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Intuitive syntax** — `fd PATTERN` instead of `find -iname '*PATTERN*'`.
- **Regex (default) and glob** matching (`-g/--glob`).
- **Fast, parallel** directory traversal using goroutines.
- **Smart case** — case-insensitive unless the pattern has an uppercase char.
- **Respects ignore files** — `.gitignore`, `.ignore`, `.fdignore`, global and
  custom ignore files, with nested-directory support.
- **Hidden-file handling**, exclusion globs (`-E`), and directory pruning.
- **Rich filters** — by type (`-t`), extension (`-e`), size (`-S`), modification
  time (`--changed-within`/`--changed-before`) and owner (`-o`, unix).
- **Command execution** — run a command per result (`-x`) or batched (`-X`).
- **Output control** — colors (LS_COLORS), `--format` templates, `--print0`,
  hyperlinks, custom path separators, depth and result limits.
- **NPM distribution** — install via npm/yarn for Node.js projects.
- **Pure Go SDK** — embed fd-style search in your own programs.

## Installation

### From source

```bash
git clone https://github.com/startvibecoding/go-fd.git
cd go-fd
make build          # produces ./bin/fd
```

### Via `go install`

```bash
go install github.com/startvibecoding/go-fd/cmd/fd@latest
```

### As a Go library

```bash
go get github.com/startvibecoding/go-fd@latest
```

Then import the module root. The import path contains `go-fd`, but the package
name is `gofd`:

```go
import gofd "github.com/startvibecoding/go-fd"
```

### Via the install script

```bash
curl -fsSL https://raw.githubusercontent.com/startvibecoding/go-fd/main/install.sh | bash
# Install to a custom directory:
curl -fsSL https://raw.githubusercontent.com/startvibecoding/go-fd/main/install.sh | bash -s -- -d ~/.local/bin
# Uninstall:
curl -fsSL https://raw.githubusercontent.com/startvibecoding/go-fd/main/install.sh | bash -s -- --uninstall
```

### Via npm

The npm package ships a small launcher plus per-platform binary packages
(`optionalDependencies`), so you only download the binary for your platform.

```bash
npm install -g go-fd-installer
# Binary available as `fd`
```

### Pre-built binaries

Download a `.tar.gz` (Linux/macOS/FreeBSD) or `.zip` (Windows) from the
[GitHub Releases](https://github.com/startvibecoding/go-fd/releases) page.

### Supported platforms

go-fd is pure Go and builds for a broad OS/architecture matrix. Pre-built
binaries and npm packages are published for:

| OS | Architectures |
|----|---------------|
| Linux (glibc) | amd64, arm64, arm (v7), 386, loong64, riscv64, ppc64le, s390x |
| Linux (musl, static) | amd64, arm64 |
| macOS | amd64 (Intel), arm64 (Apple Silicon) |
| Windows | amd64, arm64, 386 |
| FreeBSD | amd64, arm64 |

The source additionally compiles for other Go targets (NetBSD, OpenBSD,
DragonFly, illumos/Solaris, Android, and more) — run `make build-<os>` or set
`GOOS`/`GOARCH` directly.

## CLI usage

```bash
# Find entries matching a regex (filename only, by default)
fd netfl

# Match all files with a given extension
fd -e go

# Glob search
fd -g '*.txt'

# Search hidden + ignored files
fd -u pattern

# Filter by type and size, in a specific path
fd -t f -S +1m '\.log$' /var/log

# Execute a command per result
fd -e jpg -x convert {} {.}.png

# Batch execution
fd -e rs -X wc -l

# Custom output template
fd -e go --format '{//} -> {/.}'
```

Run `fd --help` for the full option list.

## SDK usage

The module root exposes a friendly API in package `gofd`.

```go
package main

import (
	"context"
	"fmt"

	gofd "github.com/startvibecoding/go-fd"
)

func main() {
	// Collect all matching paths.
	paths, err := gofd.Find(context.Background(), gofd.Options{
		Pattern: `\.go$`,
		Paths:   []string{"."},
		Hidden:  false,
	})
	if err != nil {
		panic(err)
	}
	for _, p := range paths {
		fmt.Println(p)
	}

	// Or stream results as they are discovered.
	results, errs, err := gofd.Stream(context.Background(), gofd.Options{
		Pattern: "main",
		Glob:    false,
		Paths:   []string{"."},
	})
	if err != nil {
		panic(err)
	}
	for r := range results {
		fmt.Println("found:", r.Path)
	}
	for range errs {
		// non-fatal traversal errors
	}
}
```

### `gofd.Options`

The `Options` struct exposes the same knobs as the CLI: pattern interpretation
(`Glob`, `FixedStrings`, `Exact`), case handling (`CaseSensitive`,
`IgnoreCase`), ignore handling (`Hidden`, `NoIgnore`, `Unrestricted`, ...),
traversal (`MaxDepth`, `MinDepth`, `FollowLinks`, `Prune`, `Threads`), filters
(`Types`, `Extensions`, `Sizes`, `ChangedWithin`, `ChangedBefore`, `Owner`,
`Exclude`) and output (`NullSeparator`, `Format`, `MaxResults`, `Color`).

For lower-level control, the `github.com/startvibecoding/go-fd/pkg/finder`
package lets you build a `finder.Config` directly and call `finder.New(cfg)`.

## Go module compatibility

The module path is:

```text
github.com/startvibecoding/go-fd
```

It has no external Go dependencies and declares Go 1.21 as its minimum version.
Other Go projects can depend on it with `go get github.com/startvibecoding/go-fd@latest`
or a specific tagged version.

When publishing Git tags for Go consumers, use semantic module tags that match
the module path. Because the path does not include a major-version suffix like
`/v2`, publish library-compatible tags as `v0.x.y` or `v1.x.y`. The CLI can
still report the upstream fd compatibility version, such as `10.4.2-go`, through
`fd --version`.

Before pushing a release tag:

```bash
gofmt -l .
go vet ./...
go test ./...
go list ./...
```

## Project layout

```
cmd/fd/            CLI entry point and argument parser
pkg/finder/        Core engine: config, parallel walk, filtering, output, SDK
pkg/glob/          Glob -> regex translation
pkg/ignore/        gitignore-style pattern matching
pkg/filter/        Size, time and owner filters
pkg/format/        Placeholder templates ({}, {/}, {//}, {.}, {/.})
pkg/exec/          Command execution (-x / -X)
fd.go              High-level SDK (package gofd)
tests/             Integration tests
```

## Testing

```bash
make test     # go test ./...
make vet      # go vet ./...
```

## License

Licensed under the [MIT License](LICENSE).
