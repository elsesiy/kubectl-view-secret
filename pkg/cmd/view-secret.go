package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os/exec"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
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

	# impersonate a user and groups (repeat --as-group for each group)
	%[1]s view-secret <secret> --as <user> --as-group <group1> --as-group <group2>
`

	secretDescription     = "Found %d keys in secret %q. Choose one or select 'all' to view."
	secretListDescription = "Found %d secrets. Choose one."
	secretListTitle       = "Available Secrets"
	secretTitle           = "Secret Data"
	singleKeyDescription  = "Viewing only available key: %[1]s"
)

// outputFormats lists the values accepted by the --output flag
var outputFormats = []string{"text", "json", "yaml"}

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
	decodeAll    bool
	outputFormat string
	quiet        bool
	secretKey    string
	secretName   string
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
			if !slices.Contains(outputFormats, res.outputFormat) {
				return fmt.Errorf("invalid output format %q: must be one of %s", res.outputFormat, strings.Join(outputFormats, ", "))
			}

			res.ParseArgs(args)

			return res.Retrieve(c)
		},
	}

	cmd.Flags().
		BoolVarP(&res.decodeAll, "all", "a", res.decodeAll, "if true, decodes all secrets without specifying the individual secret keys")
	cmd.Flags().BoolVarP(&res.quiet, "quiet", "q", res.quiet, "if true, suppresses info output")
	// These flags are only forwarded to kubectl, see kubectlFlags
	cmd.Flags().StringP("namespace", "n", "", "override the namespace defined in the current context")
	cmd.Flags().StringP("context", "c", "", "override the current context")
	cmd.Flags().StringP("kubeconfig", "k", "", "explicitly provide the kubeconfig to use")
	cmd.Flags().String("as", "", "Username to impersonate for the operation. User could be a regular user or a service account in a namespace.")
	cmd.Flags().StringArray("as-group", nil, "Group to impersonate for the operation. Repeat the flag to specify multiple groups.")
	cmd.Flags().StringVarP(&res.outputFormat, "output", "o", "text", "output format: "+strings.Join(outputFormats, ", "))

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
	flags := kubectlFlags(cmd)

	if c.secretName == "" {
		if err := c.selectSecret(cmd, flags); err != nil {
			return err
		}
	}

	output, err := runKubectl(append([]string{"get", "secret", c.secretName, "-o", "json"}, flags...)...)
	if err != nil {
		return err
	}

	var secret Secret
	if err := json.Unmarshal(output, &secret); err != nil {
		return fmt.Errorf("failed to parse kubectl output as secret: %w", err)
	}

	errWriter := cmd.OutOrStderr()
	if c.quiet {
		errWriter = io.Discard
	}

	return ProcessSecretWithOptions(cmd.OutOrStdout(), errWriter, cmd.InOrStdin(), secret, c.secretKey, c.decodeAll, c.outputFormat)
}

// kubectlFlags returns the flags that select the cluster, namespace and identity kubectl talks to
func kubectlFlags(cmd *cobra.Command) []string {
	var args []string
	for _, name := range []string{"namespace", "context", "kubeconfig", "as"} {
		if v, _ := cmd.Flags().GetString(name); v != "" {
			args = append(args, "--"+name, v)
		}
	}

	groups, _ := cmd.Flags().GetStringArray("as-group")
	for _, g := range groups {
		args = append(args, "--as-group", g)
	}

	return args
}

// runKubectl executes kubectl and returns its output
func runKubectl(args ...string) ([]byte, error) {
	var res, cmdErr bytes.Buffer

	out := exec.Command("kubectl", args...)
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

// kubectlNames returns the names of all resources of the given kind
//
// Unlike -o json or -o jsonpath, the default output makes kubectl ask the API server for a
// table of metadata, so no secret data is downloaded just to list names.
func kubectlNames(kind string, flags ...string) ([]string, error) {
	out, err := runKubectl(append([]string{"get", kind, "--no-headers"}, flags...)...)
	if err != nil {
		return nil, err
	}

	var names []string
	for line := range strings.Lines(string(out)) {
		if fields := strings.Fields(line); len(fields) > 0 {
			names = append(names, fields[0]) // the first column is always the name
		}
	}

	return names, nil
}

// selectSecret lets the user pick one of the available secrets and stores the choice
func (c *CommandOpts) selectSecret(cmd *cobra.Command, flags []string) error {
	names, err := kubectlNames("secrets", flags...)
	if err != nil {
		return err
	}

	// Since we don't query valid namespaces, we'll avoid prompting the user to select a secret if we didn't retrieve any secrets
	if len(names) == 0 {
		return ErrNoSecretFound
	}

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(secretListTitle).
				Description(fmt.Sprintf(secretListDescription, len(names))).
				Options(huh.NewOptions(names...)...).
				Value(&c.secretName),
		),
	).WithProgramOptions(tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout())).Run()
	if err != nil {
		return fmt.Errorf("failed to get user selection: %w", err)
	}

	return nil
}

// decodeAllData decodes the given keys of the secret and returns them as key-value pairs
func decodeAllData(secret Secret, keys []string) ([]KeyValue, error) {
	var decodedData []KeyValue
	for _, k := range keys {
		s, err := secret.Decode(secret.Data[k])
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

	keys := slices.Sorted(maps.Keys(data))

	switch {
	case decodeAll:
	case secretKey != "":
		v, ok := data[secretKey]
		if !ok {
			return ErrSecretKeyNotFound
		}
		s, err := secret.Decode(v)
		if err != nil {
			return fmt.Errorf("failed to decode key %s: %w", secretKey, err)
		}
		return outputFormattedSecret(outWriter, secret, []KeyValue{{Key: secretKey, Value: s}}, outputFormat)
	case len(keys) == 1:
		if _, err := fmt.Fprintf(errWriter, singleKeyDescription+"\n", keys[0]); err != nil {
			return fmt.Errorf("failed to write to stderr: %w", err)
		}
	default:
		var selection string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(secretTitle).
					Description(fmt.Sprintf(secretDescription, len(data), secret.Metadata.Name)).
					Options(huh.NewOptions(append([]string{"all"}, keys...)...)...).
					Value(&selection),
			),
		).WithProgramOptions(tea.WithInput(inputReader), tea.WithOutput(outWriter)).Run()
		if err != nil {
			return err
		}

		return ProcessSecretWithOptions(outWriter, errWriter, inputReader, secret, selection, selection == "all", outputFormat)
	}

	decodedData, err := decodeAllData(secret, keys)
	if err != nil {
		return err
	}
	return outputFormattedSecret(outWriter, secret, decodedData, outputFormat)
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
	for _, kv := range sortedData {
		line := kv.Value + "\n"
		if len(sortedData) > 1 {
			line = fmt.Sprintf("%s='%s'\n", kv.Key, kv.Value)
		}
		if _, err := io.WriteString(outWriter, line); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}
	return nil
}
