// Command schwab-cli is a Go port of the Python library schwab-py: a
// command-line client for the Charles Schwab Trader API.
//
// See https://github.com/alexgolec/schwab-py for the upstream project and
// README.md in this repository for the method-to-command parity checklist.
package main

import "github.com/rayli09/schwab-cli/cmd"

func main() {
	cmd.Execute()
}
