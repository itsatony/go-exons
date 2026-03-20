// Command exons provides a CLI for validating, linting, and compiling .exons files.
package main

import (
	"fmt"
	"os"
)

const (
	exitOK    = 0
	exitError = 1
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(exitOK)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("exons v0.1.0")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(exitError)
	}
}

func printUsage() {
	fmt.Println("Usage: exons <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  validate    Validate .exons file syntax and configuration")
	fmt.Println("  lint        Check for complexity, security, and performance issues")
	fmt.Println("  compile     Compile an agent spec and output the result")
	fmt.Println("  format      Normalize frontmatter formatting")
	fmt.Println("  version     Display version information")
	fmt.Println("  help        Show this help message")
}
