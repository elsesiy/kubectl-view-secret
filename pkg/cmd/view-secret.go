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

	singleKeyDescription = "Choosing key: %[1]s"
	listDescription      = "Multiple sub keys found. Specify another argument, one of:"
	listPrefix           = "->"
)

// ErrSecretKeyNotFound is thrown if the key doesn't exist in the secret
var ErrSecretKeyNotFound = errors.New("provided key not found in secret")

// ErrSecretEmpty is thrown when there's no data in the secret
var ErrSecretEmpty = errors.New("secret is empty")

// ErrInsufficientArgs is thrown if arg len <1 or >2
var ErrInsufficientArgs = fmt.Errorf("\nincorrect number or arguments, see --help for usage instructions")

// CommandOpts is the struct holding common properties
type CommandOpts struct {
	customNamespace string
	customContext   string
	decodeAll       bool
	kubeConfig      string
	secretName      string
	secretKey       string
	quiet           bool
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

	return cmd
}

// Validate ensures proper command usage
func (c *CommandOpts) Validate(args []string) error {
	argLen := len(args)
	if argLen < 1 || argLen > 2 {
		return ErrInsufficientArgs
	}

	c.secretName = args[0]
	if argLen == 2 {
		c.secretKey = args[1]
	}

	return nil
}

// Retrieve reads the kubeconfig and decodes the secret
func (c *CommandOpts) Retrieve(cmd *cobra.Command) error {
	nsOverride, _ := cmd.Flags().GetString("namespace")
	ctxOverride, _ := cmd.Flags().GetString("context")
	kubeConfigOverride, _ := cmd.Flags().GetString("kubeconfig")

	var res, cmdErr bytes.Buffer
	commandArgs := []string{"get", "secret", c.secretName, "-o", "json"}
	if nsOverride != "" {
		commandArgs = append(commandArgs, "-n", nsOverride)
	}

	if ctxOverride != "" {
		commandArgs = append(commandArgs, "--context", ctxOverride)
	}

	if kubeConfigOverride != "" {
		commandArgs = append(commandArgs, "--kubeconfig", kubeConfigOverride)
	}

	out := exec.Command("kubectl", commandArgs...)
	out.Stdout = &res
	out.Stderr = &cmdErr
	err := out.Run()
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, cmdErr.String())
		return nil
	}

	var secret map[string]interface{}
	if err := json.Unmarshal(res.Bytes(), &secret); err != nil {
		return err
	}

	if c.quiet {
		return ProcessSecret(os.Stdout, io.Discard, secret, c.secretKey, c.decodeAll)
	}

	return ProcessSecret(os.Stdout, os.Stderr, secret, c.secretKey, c.decodeAll)
}

// ProcessSecret takes the secret and user input to determine the output
func ProcessSecret(outWriter, errWriter io.Writer, secret map[string]interface{}, secretKey string, decodeAll bool) error {
	data, ok := secret["data"].(map[string]interface{})
	if !ok {
		return ErrSecretEmpty
	}

	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if decodeAll {
		for _, k := range keys {
			b64d, _ := base64.StdEncoding.DecodeString(data[k].(string))
			_, _ = fmt.Fprintf(outWriter, "%s=\\'%s\\'\n", k, strings.TrimSpace(string(b64d)))
		}
	} else if len(data) == 1 {
		for k, v := range data {
			_, _ = fmt.Fprintf(errWriter, singleKeyDescription+"\n", k)
			b64d, _ := base64.StdEncoding.DecodeString(v.(string))
			_, _ = fmt.Fprint(outWriter, string(b64d))
		}
	} else if secretKey != "" {
		if v, ok := data[secretKey]; ok {
			b64d, _ := base64.StdEncoding.DecodeString(v.(string))
			_, _ = fmt.Fprint(outWriter, string(b64d))
		} else {
			return ErrSecretKeyNotFound
		}
	} else {
		_, _ = fmt.Fprintln(errWriter, listDescription)
		for k := range data {
			_, _ = fmt.Fprintf(outWriter, "%s %s\n", listPrefix, k)
		}
	}

	return nil
}
