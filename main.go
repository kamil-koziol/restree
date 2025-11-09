package main

import (
	"fmt"
	"os"

	"github.com/kamil-koziol/restree/cmd"
)

type Subcommand struct {
	Run         func([]string, []string) int
	Description string
}

// Define all subcommands here.
var subcommands = map[string]Subcommand{
	"build": {
		Run:         cmd.Build,
		Description: "Recursively build http file",
	},
	"init": {
		Run:         cmd.Init,
		Description: "Simple restree starter",
	},
	"run": {
		Run:         cmd.Run,
		Description: "Run http file",
	},
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		fmt.Println("Usage:")
		fmt.Printf("  %s <subcommand> [args]\n", os.Args[0])
		fmt.Println("\nAvailable subcommands:")
		for name, sub := range subcommands {
			fmt.Printf("  %-10s %s\n", name, sub.Description)
		}
		os.Exit(0)
	}

	subcommand := os.Args[1]

	if sub, ok := subcommands[subcommand]; ok {
		os.Exit(sub.Run(os.Args[:2], os.Args[2:]))
	} else {
		fmt.Fprintf(os.Stderr, "Error: unknown subcommand %q\n\n", subcommand)
		os.Exit(1)
	}
}
