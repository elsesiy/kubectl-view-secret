package cmd

import (
	"maps"
	"slices"

	"github.com/goccy/go-json"
	"github.com/spf13/cobra"
)

// getNamespaces returns a list of namespaces for shell completion
func getNamespaces(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	namespaces, _ := kubectlNames("namespaces")

	return namespaces, cobra.ShellCompDirectiveNoFileComp
}

// getSecrets returns a list of secrets for shell completion
func getSecrets(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var flags []string
	if ns, _ := cmd.Flags().GetString("namespace"); ns != "" {
		flags = []string{"-n", ns}
	}

	secrets, _ := kubectlNames("secrets", flags...)

	return secrets, cobra.ShellCompDirectiveNoFileComp
}

// getDataKeys returns a list of data keys for shell completion
func getDataKeys(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) < 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	kubectlArgs := []string{"get", "secret", args[0], "-o", "jsonpath={.data}"}
	if ns, _ := cmd.Flags().GetString("namespace"); ns != "" {
		kubectlArgs = append(kubectlArgs, "-n", ns)
	}

	out, err := runKubectl(kubectlArgs...)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Parse the JSON output to extract keys
	var data map[string]string
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return slices.Sorted(maps.Keys(data)), cobra.ShellCompDirectiveNoFileComp
}
