package main

import "fmt"

func printHelp() {
	fmt.Print(`fd ` + version + `
A program to find entries in your filesystem with regex and glob based matching.
By default, fd respects gitignore rules, ignores hidden directories, and is
case insensitive.

USAGE:
    fd [OPTIONS] [pattern] [path...]

ARGS:
    <pattern>    the search pattern (a regular expression, unless --glob is used; optional)
    <path>...    the root directories for the filesystem search (optional)

OPTIONS:
    -H, --hidden                  Search hidden files and directories
    -I, --no-ignore               Do not respect .(git|fd)ignore files
        --no-ignore-vcs           Do not respect .gitignore files
        --no-require-git          Respect gitignore even outside a git repository
        --no-ignore-parent        Do not respect ignore files in parent directories
        --no-global-ignore-file   Do not respect the global ignore file
    -u, --unrestricted            Unrestricted search, alias for '--no-ignore --hidden'
    -s, --case-sensitive          Case-sensitive search (default: smart case)
    -i, --ignore-case             Case-insensitive search (default: smart case)
    -g, --glob                    Glob-based search (default: regular expression)
        --regex                   Regular-expression based search (default)
    -F, --fixed-strings           Treat pattern as literal string instead of regex
        --exact                   Match the entire filename exactly (literal)
        --and <pattern>           Additional patterns that all need to match
    -a, --absolute-path           Show absolute instead of relative paths
    -l, --list-details            Use a long listing format with file metadata
    -L, --follow                  Follow symbolic links
    -p, --full-path               Search full abs. path (default: filename only)
    -0, --print0                  Separate results by the null character
    -d, --max-depth <depth>       Set maximum search depth (default: none)
        --min-depth <depth>       Only show results starting at the given depth
        --exact-depth <depth>     Only show results at the exact given depth
    -E, --exclude <glob>          Exclude entries that match the given glob pattern
        --prune                   Do not traverse into matching directories
    -t, --type <filetype>         Filter by type: file (f), directory (d), symlink (l),
                                  executable (x), empty (e), socket (s), pipe (p),
                                  char-device (c), block-device (b)
    -e, --extension <ext>         Filter by file extension
    -S, --size <size>             Limit results based on the size of files
        --changed-within <date>   Filter by file modification time (newer than)
        --changed-before <date>   Filter by file modification time (older than)
    -o, --owner <user:group>      Filter by owning user and/or group
        --format <fmt>            Print results according to template
    -x, --exec <cmd>...           Execute a command for each search result
    -X, --exec-batch <cmd>...     Execute a command with all search results at once
        --batch-size <size>       Max number of args to run as a batch with -X
        --ignore-file <path>      Add a custom ignore-file in '.gitignore' format
    -c, --color <when>            When to use colors [auto, always, never]
        --hyperlink[=<when>]      Add hyperlinks to output paths
        --ignore-contain <name>   Ignore directories containing the named entry
    -j, --threads <num>           Set number of threads for searching & executing
        --max-results <count>     Limit the number of search results
    -1                            Limit search to a single result
    -q, --quiet                   Print nothing, exit code 0 if match found
        --show-errors             Show filesystem errors
    -C, --base-directory <path>   Change current working directory
        --path-separator <sep>    Set path separator when printing file paths
        --search-path <path>      Provide paths to search (alternative to positional)
        --strip-cwd-prefix[=when] Strip the './' prefix [auto, always, never]
        --one-file-system         Do not descend into other file systems
    -h, --help                    Print help
    -V, --version                 Print version

Bugs can be reported on GitHub: https://github.com/sharkdp/fd/issues
`)
}
