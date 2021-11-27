package main

import (
	"os"

	"github.com/elsesiy/kubectl-view-secret/pkg/cmd"
)

func main() {
	command := cmd.NewCmdViewSecret()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
