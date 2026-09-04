package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"snowgo/internal/modulegen"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("modulegen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	module := flags.String("module", "demo", "business module name")
	domain := flags.String("domain", "admin", "business domain name")
	root := flags.String("root", ".", "project root")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	result, err := modulegen.Generate(modulegen.Options{
		Root:   *root,
		Domain: *domain,
		Module: *module,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	for _, path := range result.Created {
		if _, err := fmt.Fprintf(stdout, "created %s\n", path); err != nil {
			return 1
		}
	}
	for _, path := range result.Updated {
		if _, err := fmt.Fprintf(stdout, "updated %s\n", path); err != nil {
			return 1
		}
	}
	return 0
}
