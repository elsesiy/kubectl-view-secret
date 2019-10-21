package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	clioptions "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	"os/exec"
)

const example = `
	# decode secret by name
	%[1]s view-secret <secret>

	# decode secret by name in different namespace
	%[1]s view-secret <secret> -n/--namespace ns
`

// CommandOpts is the struct holding common properties
type CommandOpts struct {
	configFlags *clioptions.ConfigFlags
	clioptions.IOStreams

	cmdArgs []string
}

// NewCmdViewSecret creates the cobra command to be executed
func NewCmdViewSecret(streams clioptions.IOStreams) *cobra.Command {
	res := &CommandOpts{
		configFlags: &clioptions.ConfigFlags{Namespace: pointer.StringPtr("")},
		IOStreams:   streams,
	}

	cmd := &cobra.Command{
		Use:          "view-secret [secret-name]",
		Short:        "Decode a kubernetes secret by name in the current context/cluster/namespace",
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

	res.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Validate ensures proper command usage
func (d *CommandOpts) Validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("\nplease provide only the secret name to be decoded, see --help for usage instructions")
	}

	d.cmdArgs = args

	return nil
}

// Retrieve reads the kubeconfig and decodes the secret
func (d *CommandOpts) Retrieve(c *cobra.Command) error {
	kubeCfg, err := d.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}

	currCtx := kubeCfg.Contexts[kubeCfg.CurrentContext]
	currCluster := currCtx.Cluster
	currNs := currCtx.Namespace

	nsOverride, _ := c.Flags().GetString("namespace")
	if nsOverride != "" {
		currNs = nsOverride
	}

	var res, cmdErr bytes.Buffer
	commandArgs := []string{"get", "secret", d.cmdArgs[0], "-n", currNs, "-o", "json"}
	out := exec.Command("kubectl", commandArgs...)
	out.Stdout = &res
	out.Stderr = &cmdErr
	err = out.Run()
	if err != nil {
		fmt.Print(cmdErr.String())
		return nil
	}

	var secJson map[string]interface{}
	if err := json.Unmarshal(res.Bytes(), &secJson); err != nil {
		return err
	}

	secrets := secJson["data"].(map[string]interface{})

	if len(secrets) > 0 {
		fmt.Printf("Decoded secret '%s' in namespace '%s' for cluster '%s'\n\n", d.cmdArgs[0], currNs, currCluster)
		for k, v := range secrets {
			b64d, _ := base64.URLEncoding.DecodeString(v.(string))
			fmt.Printf("%s=%s\n", k, b64d)
		}
	} else {
		fmt.Println("the provided secret is empty")
	}

	return nil
}
