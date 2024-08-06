package cmd

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/goccy/go-json"
	"github.com/spf13/cobra"
)

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
`

	secretDescription     = "Found %d keys in secret %q. Choose one or select 'all' to view."
	secretListDescription = "Multiple (%d) secrets found. Choose one."
	secretListTitle       = "Available Secrets"
	secretTitle           = "Secret Data"
	singleKeyDescription  = "Viewing only available key: %[1]s"
)

var (
	// ErrInsufficientArgs is thrown if arg len <1 or >2
	ErrInsufficientArgs = fmt.Errorf("\nincorrect number or arguments, see --help for usage instructions")

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
	quiet               bool
	secretKey           string
	secretName          string
}

// NewCmdViewSecret creates the cobra command to be executed
func NewCmdViewSecret() *cobra.Command {
	res := &CommandOpts{}

	cmd := &cobra.Command{
		Use:          "view-secret [secret-name] [secret-key]",
		Short:        "Decode a kubernetes secret by name & key in the current context/cluster/namespace",
		Example:      fmt.Sprintf(example, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := res.Validate(args); err != nil {
				return err
			}
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

	return cmd
}

// Validate ensures proper command usage
func (c *CommandOpts) Validate(args []string) error {
	argLen := len(args)
	switch argLen {
	case 1:
		c.secretName = args[0]
	case 2:
		c.secretName = args[0]
		c.secretKey = args[1]
	default:
		if argLen < 0 || argLen > 2 {
			return ErrInsufficientArgs
		}
	}

	return nil
}

// Retrieve reads the kubeconfig and decodes the secret
func (c *CommandOpts) Retrieve(cmd *cobra.Command) error {
	nsOverride, _ := cmd.Flags().GetString("namespace")
	ctxOverride, _ := cmd.Flags().GetString("context")
	kubeConfigOverride, _ := cmd.Flags().GetString("kubeconfig")
	impersonateOverride, _ := cmd.Flags().GetString("as")
	impersonateGroupOverride, _ := cmd.Flags().GetString("as-group")

	var res, cmdErr bytes.Buffer

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

	out := exec.Command("kubectl", commandArgs...)
	out.Stdout = &res
	out.Stderr = &cmdErr
	err := out.Run()
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, cmdErr.String())
		return nil
	}

	var secret Secret
	if c.secretName == "" {
		var secretList SecretList
		if err := json.Unmarshal(res.Bytes(), &secretList); err != nil {
			return err
		}

		// Since we don't query valid namespaces, we'll avoid prompting the user to select a secret if we didn't retrieve any secrets
		if len(secretList.Items) == 0 {
			return ErrNoSecretFound
		}

		opts := []string{}
		secretMap := map[string]Secret{}
		for _, v := range secretList.Items {
			opts = append(opts, v.Metadata.Name)
			secretMap[v.Metadata.Name] = v
		}

		err := huh.NewSelect[string]().
			Title(secretListTitle).
			Description(fmt.Sprintf(secretListDescription, len(secretList.Items))).
			Options(huh.NewOptions(opts...)...).
			Value(&c.secretName).
			Run()
		if err != nil {
			return err
		}

		secret = secretMap[c.secretName]
	} else {
		if err := json.Unmarshal(res.Bytes(), &secret); err != nil {
			return err
		}
	}

	if c.quiet {
		return ProcessSecret(os.Stdout, io.Discard, os.Stdin, secret, c.secretKey, c.decodeAll)
	}

	return ProcessSecret(os.Stdout, os.Stderr, os.Stdin, secret, c.secretKey, c.decodeAll)
}

// ProcessSecret takes the secret and user input to determine the output
func ProcessSecret(outWriter, errWriter io.Writer, inputReader io.Reader, secret Secret, secretKey string, decodeAll bool) error {
	data := secret.Data
	if len(data) == 0 {
		return ErrSecretEmpty
	}

	var keys []string
	for k := range secret.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if decodeAll {
		for _, k := range keys {
			b64d, _ := base64.StdEncoding.DecodeString(data[k])
			_, _ = fmt.Fprintf(outWriter, "%s='%s'\n", k, strings.TrimSpace(string(b64d)))
		}
	} else if len(data) == 1 {
		for k, v := range data {
			_, _ = fmt.Fprintf(errWriter, singleKeyDescription+"\n", k)
			b64d, _ := base64.StdEncoding.DecodeString(v)
			_, _ = fmt.Fprint(outWriter, string(b64d))
		}
	} else if secretKey != "" {
		if v, ok := data[secretKey]; ok {
			b64d, _ := base64.StdEncoding.DecodeString(v)
			_, _ = fmt.Fprint(outWriter, string(b64d))
		} else {
			return ErrSecretKeyNotFound
		}
	} else {
		opts := []string{"all"}
		for k := range data {
			opts = append(opts, k)
		}

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

		return ProcessSecret(outWriter, errWriter, inputReader, secret, selection, decodeAll)
	}

	return nil
}
