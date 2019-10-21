package main

import (
	"github.com/elsesiy/kubectl-view-secret/pkg/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
)

func main() {
	command := cmd.NewCmdViewSecret(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
