package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/goccy/go-json"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// KeyValue represents a key-value pair for sorted output
type KeyValue struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

const (
	example = `
	# print secret keys
	%[1]s view-secret <secret>

	# decode specific entry
	%[1]s view-secret <secret> <key>

	# decode all contents
	%[1]s view-secret <secret> -a/--all

	# print keys for secret in different namespace
	%[1]s view-secret <secret> -n/--namespace <ns>

	# print keys for secret in different context
	%[1]s view-secret <secret> -c/--context <ctx>

	# print keys for secret by providing kubeconfig
	%[1]s view-secret <secret> -k/--kubeconfig <cfg>

	# suppress info output
	%[1]s view-secret <secret> -q/--quiet

	# output in json (or yaml) instead of text
	%[1]s view-secret <secret> -o/--output json
`

	secretDescription     = "Found %d keys in secret %q. Choose one or select 'all' to view."
	secretListDescription = "Found %d secrets. Choose one."
	secretListTitle       = "Available Secrets"
	secretTitle           = "Secret Data"
	singleKeyDescription  = "Viewing only available key: %[1]s"
)

var (
	// ErrNoSecretFound is thrown when no secret name was provided but we didn't find any secrets
	ErrNoSecretFound = errors.New("no secrets found")

	// ErrSecretEmpty is thrown when there's no data in the secret
	ErrSecretEmpty = errors.New("secret is empty")

	// ErrSecretKeyNotFound is thrown if the key doesn't exist in the secret
	ErrSecretKeyNotFound = errors.New("provided key not found in secret")
)

// CommandOpts is the struct holding common properties
type CommandOpts struct {
	customContext       string
	customNamespace     string
	decodeAll           bool
	impersonateAs       string
	impersonateAsGroups string
	kubeConfig          string
	outputFormat        string
	quiet               bool
	secretKey           string
	secretName          string
}

// NewCmdViewSecret creates the cobra command to be executed
//
// This command provides an interactive way to view Kubernetes secrets
// in plaintext. It supports various output formats and secret types.
func NewCmdViewSecret() *cobra.Command {
	res := &CommandOpts{}

	cmd := &cobra.Command{
		Args:         cobra.RangeArgs(0, 2),
		Example:      fmt.Sprintf(example, "kubectl"),
		Short:        "Decode a kubernetes secret by name & key in the current context/cluster/namespace",
		SilenceUsage: true,
		Use:          "view-secret [secret-name] [secret-key]",
		RunE: func(c *cobra.Command, args []string) error {
			res.ParseArgs(args)
			if err := res.Retrieve(c); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().
		BoolVarP(&res.decodeAll, "all", "a", res.decodeAll, "if true, decodes all secrets without specifying the individual secret keys")
	cmd.Flags().BoolVarP(&res.quiet, "quiet", "q", res.quiet, "if true, suppresses info output")
	cmd.Flags().
		StringVarP(&res.customNamespace, "namespace", "n", res.customNamespace, "override the namespace defined in the current context")
	cmd.Flags().StringVarP(&res.customContext, "context", "c", res.customContext, "override the current context")
	cmd.Flags().StringVarP(&res.kubeConfig, "kubeconfig", "k", res.kubeConfig, "explicitly provide the kubeconfig to use")
	cmd.Flags().StringVar(&res.impersonateAs, "as", res.impersonateAs, "Username to impersonate for the operation. User could be a regular user or a service account in a namespace.")
	cmd.Flags().StringVar(&res.impersonateAsGroups, "as-group", res.impersonateAsGroups, "Groups to impersonate for the operation. Multipe groups can be specified by comma separated.")
	cmd.Flags().StringVarP(&res.outputFormat, "output", "o", "text", "output format: text, json, yaml")

	// Add shell completion functions
	_ = cmd.RegisterFlagCompletionFunc("namespace", getNamespaces)
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			return getSecrets(cmd, args, toComplete)
		case 1:
			return getDataKeys(cmd, args, toComplete)
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return cmd
}

// ParseArgs serializes the user supplied program arguments
func (c *CommandOpts) ParseArgs(args []string) {
	argLen := len(args)
	if argLen >= 1 {
		c.secretName = args[0]

		if argLen == 2 {
			c.secretKey = args[1]
		}
	}
}

// Retrieve reads the kubeconfig and decodes the secret
//
// It executes kubectl to fetch secret data, handles user interaction
// for secret selection, and outputs the decoded content in the
// specified format.
func (c *CommandOpts) Retrieve(cmd *cobra.Command) error {
	commandArgs := c.buildKubectlCommand(cmd)
	output, err := c.executeKubectlCommand(commandArgs)
	if err != nil {
		return err
	}

	secret, err := c.parseSecretResponse(output, cmd)
	if err != nil {
		return err
	}

	if c.quiet {
		return ProcessSecretWithOptions(cmd.OutOrStdout(), io.Discard, cmd.InOrStdin(), secret, c.secretKey, c.decodeAll, c.outputFormat)
	}

	return ProcessSecretWithOptions(cmd.OutOrStdout(), cmd.OutOrStderr(), cmd.InOrStdin(), secret, c.secretKey, c.decodeAll, c.outputFormat)
}

// buildKubectlCommand builds the kubectl command arguments
func (c *CommandOpts) buildKubectlCommand(cmd *cobra.Command) []string {
	nsOverride, _ := cmd.Flags().GetString("namespace")
	ctxOverride, _ := cmd.Flags().GetString("context")
	kubeConfigOverride, _ := cmd.Flags().GetString("kubeconfig")
	impersonateOverride, _ := cmd.Flags().GetString("as")
	impersonateGroupOverride, _ := cmd.Flags().GetString("as-group")

	commandArgs := []string{"get", "secret", "-o", "json"}
	if c.secretName != "" {
		commandArgs = []string{"get", "secret", c.secretName, "-o", "json"}
	}

	if nsOverride != "" {
		commandArgs = append(commandArgs, "-n", nsOverride)
	}

	if ctxOverride != "" {
		commandArgs = append(commandArgs, "--context", ctxOverride)
	}

	if kubeConfigOverride != "" {
		commandArgs = append(commandArgs, "--kubeconfig", kubeConfigOverride)
	}

	if impersonateOverride != "" {
		commandArgs = append(commandArgs, "--as", impersonateOverride)
	}

	if impersonateGroupOverride != "" {
		commandArgs = append(commandArgs, "--as-group", impersonateGroupOverride)
	}

	return commandArgs
}

// executeKubectlCommand executes the kubectl command and returns the output
func (c *CommandOpts) executeKubectlCommand(commandArgs []string) ([]byte, error) {
	var res, cmdErr bytes.Buffer

	out := exec.Command("kubectl", commandArgs...)
	out.Stdout = &res
	out.Stderr = &cmdErr
	err := out.Run()
	if err != nil {
		if cmdErr.Len() > 0 {
			return nil, fmt.Errorf("%sError: kubectl command failed: %w", cmdErr.String(), err)
		}
		return nil, fmt.Errorf("kubectl command failed: %w", err)
	}

	return res.Bytes(), nil
}

// parseSecretResponse parses the kubectl output and handles secret selection
func (c *CommandOpts) parseSecretResponse(output []byte, cmd *cobra.Command) (Secret, error) {
	var secret Secret
	if c.secretName == "" {
		return c.handleSecretSelection(output, cmd)
	}

	if err := json.Unmarshal(output, &secret); err != nil {
		return secret, fmt.Errorf("failed to parse kubectl output as secret: %w", err)
	}

	return secret, nil
}

// handleSecretSelection handles the interactive selection of a secret from a list
func (c *CommandOpts) handleSecretSelection(output []byte, cmd *cobra.Command) (Secret, error) {
	var secretList SecretList
	if err := json.Unmarshal(output, &secretList); err != nil {
		return Secret{}, fmt.Errorf("failed to parse kubectl output as secret list: %w", err)
	}

	// Since we don't query valid namespaces, we'll avoid prompting the user to select a secret if we didn't retrieve any secrets
	if len(secretList.Items) == 0 {
		return Secret{}, ErrNoSecretFound
	}

	opts := []string{}
	secretMap := map[string]Secret{}
	for _, v := range secretList.Items {
		opts = append(opts, v.Metadata.Name)
		secretMap[v.Metadata.Name] = v
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(secretListTitle).
				Description(fmt.Sprintf(secretListDescription, len(secretList.Items))).
				Options(huh.NewOptions(opts...)...).
				Value(&c.secretName),
		),
	).WithProgramOptions(tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout())).Run()
	if err != nil {
		return Secret{}, fmt.Errorf("failed to get user selection: %w", err)
	}

	return secretMap[c.secretName], nil
}

// ProcessSecret takes the secret and user input to determine the output
func ProcessSecret(outWriter, errWriter io.Writer, inputReader io.Reader, secret Secret, secretKey string, decodeAll bool) error {
	return ProcessSecretWithOptions(outWriter, errWriter, inputReader, secret, secretKey, decodeAll, "text")
}

// decodeAllData decodes all data in the secret and returns sorted key-value pairs
func decodeAllData(secret Secret, data SecretData) ([]KeyValue, error) {
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var decodedData []KeyValue
	for _, k := range keys {
		v := data[k]
		s, err := secret.Decode(v)
		if err != nil {
			return nil, fmt.Errorf("failed to decode key %s: %w", k, err)
		}
		decodedData = append(decodedData, KeyValue{Key: k, Value: s})
	}
	return decodedData, nil
}

// ProcessSecretWithOptions takes the secret and user input with full options
func ProcessSecretWithOptions(outWriter, errWriter io.Writer, inputReader io.Reader, secret Secret, secretKey string, decodeAll bool, outputFormat string) error {
	data := secret.Data
	if len(data) == 0 {
		return ErrSecretEmpty
	}

	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if decodeAll {
		decodedData, err := decodeAllData(secret, data)
		if err != nil {
			return err
		}
		return outputFormattedSecret(outWriter, secret, decodedData, outputFormat)
	} else if len(data) == 1 {
		if _, err := fmt.Fprintf(errWriter, singleKeyDescription+"\n", keys[0]); err != nil {
			return fmt.Errorf("failed to write to stderr: %w", err)
		}
		decodedData, err := decodeAllData(secret, data)
		if err != nil {
			return err
		}
		return outputFormattedSecret(outWriter, secret, decodedData, outputFormat)
	} else if secretKey != "" {
		if v, ok := data[secretKey]; ok {
			s, err := secret.Decode(v)
			if err != nil {
				return fmt.Errorf("failed to decode key %s: %w", secretKey, err)
			}
			if outputFormat == "text" {
				if _, err := fmt.Fprintf(outWriter, "%s\n", s); err != nil {
					return fmt.Errorf("failed to write output: %w", err)
				}
			} else {
				decodedData := []KeyValue{
					{
						Key:   secretKey,
						Value: s,
					},
				}
				return outputFormattedSecret(outWriter, secret, decodedData, outputFormat)
			}
		} else {
			return ErrSecretKeyNotFound
		}
	} else {
		opts := make([]string, len(keys)+1)
		opts[0] = "all"
		copy(opts[1:], keys)

		var selection string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(secretTitle).
					Description(fmt.Sprintf(secretDescription, len(data), secret.Metadata.Name)).
					Options(huh.NewOptions(opts...)...).
					Value(&selection),
			),
		).WithProgramOptions(tea.WithInput(inputReader), tea.WithOutput(outWriter)).Run()
		if err != nil {
			return err
		}

		if selection == "all" {
			decodeAll = true
		}

		return ProcessSecretWithOptions(outWriter, errWriter, inputReader, secret, selection, decodeAll, outputFormat)
	}

	return nil
}

// buildOutputMap builds the common output structure for JSON/YAML formats
func buildOutputMap(secret Secret, sortedData []KeyValue) map[string]any {
	return map[string]any{
		"name":      secret.Metadata.Name,
		"namespace": secret.Metadata.Namespace,
		"type":      secret.Type,
		"data":      sortedData,
	}
}

// outputFormattedSecret outputs the secret in the specified format
func outputFormattedSecret(outWriter io.Writer, secret Secret, decodedData []KeyValue, outputFormat string) error {
	switch outputFormat {
	case "json":
		return outputJSON(outWriter, secret, decodedData)
	case "yaml":
		return outputYAML(outWriter, secret, decodedData)
	default:
		return outputText(outWriter, decodedData)
	}
}

// outputJSON outputs secret data as JSON
func outputJSON(outWriter io.Writer, secret Secret, decodedData []KeyValue) error {
	output := buildOutputMap(secret, decodedData)
	encoder := json.NewEncoder(outWriter)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputYAML outputs secret data as YAML
func outputYAML(outWriter io.Writer, secret Secret, decodedData []KeyValue) error {
	output := buildOutputMap(secret, decodedData)
	return yaml.NewEncoder(outWriter).Encode(output)
}

// outputText outputs secret data as plain text
func outputText(outWriter io.Writer, sortedData []KeyValue) error {
	var format string

	if len(sortedData) == 1 {
		format = "%s\n"
	} else {
		format = "%s='%s'\n"
	}

	for _, kv := range sortedData {
		var args []any
		if len(sortedData) == 1 {
			args = []any{kv.Value}
		} else {
			args = []any{kv.Key, kv.Value}
		}
		if _, err := fmt.Fprintf(outWriter, format, args...); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}
	return nil
}
