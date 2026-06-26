# AGENTS.md

Guidance for AI agents and contributors working on **go-fd**, a pure-Go port of
[`fd`](https://github.com/sharkdp/fd) (the fast `find` alternative). The project
ships both a CLI (`cmd/fd`) and an embeddable SDK (root package `gofd`).

## Project goals

- **Faithful behavior parity** with upstream `fd` (currently tracking v10.4.2).
  When in doubt about a feature's behavior, compare against the real `fd`
  binary; matching its observable output is the priority.
- **Pure Go, zero external dependencies.** Do not add third-party modules
  without strong justification. The standard library (`regexp`, `os`, `io/fs`,
  `path/filepath`, `runtime`, `os/exec`, ...) covers current needs.
- **Two consumption modes:** a CLI compatible with fd's flags, and a clean Go
  SDK for programmatic use. Keep both working.

## Repository layout

```
cmd/fd/            CLI entry point
  main.go          hand-rolled argument parser + run()
  help.go          --help text and version constant
fd.go              High-level SDK (package gofd): Options, Compile, Find, Stream
isterm.go          terminal detection helper
pkg/finder/        Core engine
  config.go        Config struct + FileType(s)
  direntry.go      lazy DirEntry abstraction
  filetypes.go     --type filtering
  lscolors.go      LS_COLORS parsing + ANSI painting
  output.go        printing (plain/colorized/format, hyperlinks)
  walk.go          parallel, ignore-aware directory traversal
  finder.go        Finder: matching, Stream/Find SDK, Run (CLI behavior)
  util.go          numCPU helper
pkg/glob/          glob -> regex translation
pkg/ignore/        gitignore-style pattern matching
pkg/filter/        size / time / owner filters (owner is build-tagged unix)
pkg/format/        placeholder templates ({}, {/}, {//}, {.}, {/.})
pkg/exec/          command execution (-x / -X)
tests/             SDK-level integration tests
```

The Rust source being ported lives at `../fd` (sibling directory) and is the
reference for behavior. Each Go file/package roughly corresponds to a Rust
module: `pkg/finder/walk.go` ≈ `src/walk.rs`, `fd.go` ≈ `src/main.rs` +
`src/config.rs` construction, `cmd/fd` ≈ `src/cli.rs`, etc.

## Build / test / verify

```bash
make build      # -> ./bin/fd
make test       # go test ./...
make vet        # go vet ./...
make fmt        # gofmt -w .
make cross      # cross-compile common OS/arch targets
```

Requirements before considering a change done:
- `gofmt -l .` prints nothing (code is formatted).
- `go vet ./...` is clean.
- `go test ./...` passes.

Module path is `github.com/startvibecoding/go-fd` (Go 1.26+). Internal imports
use that prefix, e.g. `github.com/startvibecoding/go-fd/pkg/finder`. The SDK is
imported as `gofd "github.com/startvibecoding/go-fd"`.

## Behavior-parity testing

When changing matching, ignore, or output logic, diff against the real `fd`:

```bash
MINE=./bin/fd; REAL=$(command -v fd)
compare() {
  diff <($REAL "$@" -c never 2>/dev/null | sort) \
       <($MINE "$@" -c never 2>/dev/null | sort) \
    && echo "OK: fd $*" || echo "DIFF: fd $*"
}
compare -e go
compare -g '**/*.rs'
compare -t d -H -I
```

Run such comparisons in a representative tree (e.g. the sibling `../fd` repo).

## Conventions and gotchas

- **Default fd semantics** (applied in `fd.go` / `buildConfig`): smart case
  (case-insensitive unless the pattern has an uppercase char or
  `CaseSensitive`), respect gitignore/hidden by default, strip the `./` prefix
  only when no search path is given and not in null/exec mode.
- **Pattern target:** patterns match the **file name only** unless `FullPath`
  is set. This applies to glob patterns too (e.g. `-g '**/*.rs'` matches because
  `**/` can match zero directories against a bare filename).
- **Path-separator guard:** the CLI rejects a pattern containing `/` (the "you
  passed a path" hint), but this check is **skipped for `--glob`** and
  `--full-path`. Keep it that way — real fd does.
- **Ignore precedence:** in `pkg/ignore`, the last matching pattern wins;
  deeper (later-pushed) gitignore files in the walk stack override shallower
  ones; a `!` whitelist can re-include a previously ignored path.
- **Parallel walk:** `pkg/finder/walk.go` uses a semaphore-bounded goroutine
  fan-out. The semaphore is released *before* spawning child-directory
  goroutines to avoid deadlock — preserve this ordering when editing.
- **Exit codes:** `ExitSuccess` (0) / `ExitGeneralError` (1). Quiet mode returns
  0 iff at least one match was found.
- **Cross-platform:** owner filtering is unix-only via build tags
  (`owner_unix.go` / `owner_other.go`). Guard any new syscall-dependent code the
  same way and keep Windows/other builds compiling. The code is verified to
  cross-compile for the full release matrix (Linux amd64/arm64/arm/386/loong64/
  riscv64/ppc64le/s390x incl. musl, macOS amd64/arm64, Windows amd64/arm64/386,
  FreeBSD amd64/arm64) plus other Go targets. Run `make build-all` or
  `GOOS=... GOARCH=... go build ./...` to check.
- **No backtracking regex:** Go uses RE2 (`regexp`). Avoid features that assume
  PCRE backtracking; translate globs/gitignore into RE2-compatible expressions.

## Adding a feature

1. Check the upstream Rust implementation in `../fd/src` for exact semantics.
2. Add/extend the relevant `pkg/*` package with a focused unit test.
3. Wire the option through `gofd.Options` and `buildConfig` in `fd.go`.
4. Add the CLI flag(s) in `cmd/fd/main.go` (long + short, with parser test) and
   document it in `cmd/fd/help.go` and `README.md`.
5. Verify with `make fmt vet test` and a parity comparison against real `fd`.

## Style

- Standard Go style; run `gofmt`. Exported identifiers carry doc comments.
- Keep packages dependency-light and single-purpose. Prefer adding logic to the
  matching package rather than the CLI parser.
- Error messages mirror fd's phrasing where user-facing (prefixed with
  `[fd error]: ` on stderr).
