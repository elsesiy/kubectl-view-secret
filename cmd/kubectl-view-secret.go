package main

import (
	"github.com/elsesiy/kubectl-view-secret/pkg/cmd"
	"os"
)

func main() {
	command := cmd.NewCmdViewSecret()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
