package main

import (
	"fmt" // fmt is used to print human-readable output to the terminal
	"os"  // os is used so we can exit with a non-zero status on error

	// This import path must match the module path you defined in go.mod.
	// We import the cli package from the internal folder, which handles
	// parsing command-line flags and calling the generator.
	"generate_ldap_entires/internal/cli"
)

// MainConfig is a placeholder struct that could hold global settings
// for the main package in the future, such as version information or
// build metadata. Right now it is kept small but still has an
// initializer function to match the project rule that every struct
// has an initializer.
type MainConfig struct{}

// NewMainConfig is an initializer function for MainConfig.
// It returns a pointer to an empty MainConfig. Even though we do not
// use fields yet, having this function keeps struct creation
// consistent across the codebase.
func NewMainConfig() *MainConfig {
	return &MainConfig{}
}

// main is the entry point for the program. It keeps the logic very
// small by delegating all the real work to the cli.Run function.
// That makes main easy to read and easier to test in isolation.
func main() {
	// We call NewMainConfig() here to follow the pattern, even if
	// we do not yet use the config fields. This leaves a clear
	// place to add global options later.
	_ = NewMainConfig()

	// cli.Run() will parse command-line flags, possibly read an input
	// file, and then call the generator package. If it returns an
	// error, we print it and exit with a non-zero exit code so shell
	// scripts can detect failure.
	if err := cli.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
