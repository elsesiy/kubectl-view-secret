package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"os/exec"
)

const example = `
	# print secret keys
	%[1]s view-secret <secret>

	# decode secret specific key
	%[1]s view-secret <secret> <key>

	# decode all contents of a secret
	%[1]s view-secret <secret> -a/--all

	# print keys for secret in different namespace
	%[1]s view-secret <secret> -n/--namespace <ns>
`

var ErrSecretKeyNotFound = errors.New("provided key not found in secret")

// CommandOpts is the struct holding common properties
type CommandOpts struct {
	customNamespace string
	decodeAll       bool
	secretName      string
	secretKey       string
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

	cmd.Flags().BoolVarP(&res.decodeAll, "all", "a", res.decodeAll, "if true, decodes all secrets without specifying the individual secret keys")
	cmd.Flags().StringVarP(&res.customNamespace, "namespace", "n", res.customNamespace, "override the namespace defined in the current context")

	return cmd
}

// Validate ensures proper command usage
func (c *CommandOpts) Validate(args []string) error {
	argLen := len(args)
	if argLen < 1 || argLen > 2 {
		return fmt.Errorf("\nincorrect number or arguments, see --help for usage instructions")
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

	var res, cmdErr bytes.Buffer
	commandArgs := []string{"get", "secret", c.secretName, "-o", "json"}
	if nsOverride != "" {
		commandArgs = append(commandArgs, "-n", nsOverride)
	}

	out := exec.Command("kubectl", commandArgs...)
	out.Stdout = &res
	out.Stderr = &cmdErr
	err := out.Run()
	if err != nil {
		fmt.Print(cmdErr.String())
		return nil
	}

	var secret map[string]interface{}
	if err := json.Unmarshal(res.Bytes(), &secret); err != nil {
		return err
	}

	return ProcessSecret(os.Stdout, secret, c.secretKey, c.decodeAll)
}

// ProcessSecret takes the secret and user input to determine the output
func ProcessSecret(w io.Writer, secret map[string]interface{}, secretKey string, decodeAll bool) error {
	data := secret["data"].(map[string]interface{})

	if decodeAll {
		for k, v := range data {
			b64d, _ := base64.URLEncoding.DecodeString(v.(string))
			_, _ = fmt.Fprintf(w, "%s=%s\n", k, b64d)
		}
	} else if secretKey != "" {
		if v, ok := data[secretKey]; ok {
			b64d, _ := base64.URLEncoding.DecodeString(v.(string))
			_, _ = fmt.Fprintln(w, string(b64d))
		} else {
			return ErrSecretKeyNotFound
		}
	} else {
		for k := range data {
			_, _ = fmt.Fprintln(w, k)
		}
	}

	return nil
}
