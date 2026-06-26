package gofd_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	gofd "github.com/startvibecoding/go-fd"
)

func ExampleFind() {
	dir, err := os.MkdirTemp("", "gofd-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	for _, name := range []string{"main.go", "README.md", "internal/util.go"} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			panic(err)
		}
	}

	paths, err := gofd.Find(context.Background(), gofd.Options{
		Pattern: `\.go$`,
		Paths:   []string{dir},
	})
	if err != nil {
		panic(err)
	}

	var rels []string
	for _, path := range paths {
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			panic(err)
		}
		rels = append(rels, filepath.ToSlash(rel))
	}
	sort.Strings(rels)
	for _, rel := range rels {
		fmt.Println(rel)
	}

	// Output:
	// internal/util.go
	// main.go
}

func ExampleStream() {
	dir, err := os.MkdirTemp("", "gofd-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	for _, name := range []string{"main.go", "util.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644); err != nil {
			panic(err)
		}
	}

	results, errs, err := gofd.Stream(context.Background(), gofd.Options{
		Pattern: `\.go$`,
		Paths:   []string{dir},
	})
	if err != nil {
		panic(err)
	}

	count := 0
	for range results {
		count++
	}
	for err := range errs {
		if err != nil {
			panic(err)
		}
	}
	fmt.Println(count)

	// Output:
	// 2
}
