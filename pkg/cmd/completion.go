package cmd

import (
	"bytes"
	"os/exec"
	"sort"

	"github.com/goccy/go-json"
	"github.com/spf13/cobra"
)

// getNamespaces returns a list of namespaces for shell completion
func getNamespaces(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var res bytes.Buffer

	kubectlCmd := exec.Command("kubectl", "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")
	kubectlCmd.Stdout = &res
	if err := kubectlCmd.Run(); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	namespaces := []string{}
	for ns := range bytes.SplitSeq(res.Bytes(), []byte(" ")) {
		if len(ns) > 0 {
			namespaces = append(namespaces, string(ns))
		}
	}

	return namespaces, cobra.ShellCompDirectiveNoFileComp
}

// getSecrets returns a list of secrets for shell completion
func getSecrets(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var res bytes.Buffer

	ns, _ := cmd.Flags().GetString("namespace")
	kubectlArgs := []string{"get", "secrets", "-o", "jsonpath={.items[*].metadata.name}"}
	if ns != "" {
		kubectlArgs = append(kubectlArgs, "-n", ns)
	}

	kubectlCmd := exec.Command("kubectl", kubectlArgs...)
	kubectlCmd.Stdout = &res
	if err := kubectlCmd.Run(); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	secrets := []string{}
	for secret := range bytes.SplitSeq(res.Bytes(), []byte(" ")) {
		if len(secret) > 0 {
			secrets = append(secrets, string(secret))
		}
	}

	return secrets, cobra.ShellCompDirectiveNoFileComp
}

// getDataKeys returns a list of data keys for shell completion
func getDataKeys(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) < 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	secretName := args[0]
	var res bytes.Buffer

	ns, _ := cmd.Flags().GetString("namespace")
	kubectlArgs := []string{"get", "secret", secretName, "-o", "jsonpath={.data}"}
	if ns != "" {
		kubectlArgs = append(kubectlArgs, "-n", ns)
	}

	kubectlCmd := exec.Command("kubectl", kubectlArgs...)
	kubectlCmd.Stdout = &res
	if err := kubectlCmd.Run(); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Parse the JSON output to extract keys
	var data map[string]string
	if err := json.Unmarshal(res.Bytes(), &data); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys, cobra.ShellCompDirectiveNoFileComp
}
