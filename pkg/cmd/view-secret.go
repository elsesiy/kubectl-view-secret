package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"

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
	quiet               bool
	secretKey           string
	secretName          string
}

// NewCmdViewSecret creates the cobra command to be executed
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
			return err
		}

		secret = secretMap[c.secretName]
	} else {
		if err := json.Unmarshal(res.Bytes(), &secret); err != nil {
			return err
		}
	}

	if c.quiet {
		return ProcessSecret(cmd.OutOrStdout(), io.Discard, cmd.InOrStdin(), secret, c.secretKey, c.decodeAll)
	}

	return ProcessSecret(cmd.OutOrStdout(), cmd.OutOrStderr(), cmd.InOrStdin(), secret, c.secretKey, c.decodeAll)
}

// ProcessSecret takes the secret and user input to determine the output
func ProcessSecret(outWriter, errWriter io.Writer, inputReader io.Reader, secret Secret, secretKey string, decodeAll bool) error {
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
		for _, k := range keys {
			s, _ := secret.Decode(data[k])
			_, _ = fmt.Fprintf(outWriter, "%s='%s'\n", k, s)
		}
	} else if len(data) == 1 {
		for k, v := range data {
			_, _ = fmt.Fprintf(errWriter, singleKeyDescription+"\n", k)
			s, _ := secret.Decode(v)
			_, _ = fmt.Fprintf(outWriter, "%s\n", s)
		}
	} else if secretKey != "" {
		if v, ok := data[secretKey]; ok {
			s, _ := secret.Decode(v)
			_, _ = fmt.Fprintf(outWriter, "%s\n", s)
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

		return ProcessSecret(outWriter, errWriter, inputReader, secret, selection, decodeAll)
	}

	return nil
}
