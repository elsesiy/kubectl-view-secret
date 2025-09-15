package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Test completion function logic without kubectl dependency
func TestCompletionFunctionLogic(t *testing.T) {
	t.Run("getNamespaces returns correct directive", func(t *testing.T) {
		cmd := &cobra.Command{}
		_, directive := getNamespaces(cmd, nil, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})

	t.Run("getSecrets returns correct directive", func(t *testing.T) {
		cmd := &cobra.Command{}
		_, directive := getSecrets(cmd, nil, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})

	t.Run("getDataKeys with no args returns nil", func(t *testing.T) {
		cmd := &cobra.Command{}
		result, directive := getDataKeys(cmd, []string{}, "")
		assert.Nil(t, result)
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})

	t.Run("getDataKeys with args returns correct directive", func(t *testing.T) {
		cmd := &cobra.Command{}
		_, directive := getDataKeys(cmd, []string{"test-secret"}, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})
}

func TestValidArgsFunctionLogic(t *testing.T) {
	cmd := NewCmdViewSecret()

	t.Run("no args returns secrets directive", func(t *testing.T) {
		_, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})

	t.Run("one arg returns keys directive", func(t *testing.T) {
		_, directive := cmd.ValidArgsFunction(cmd, []string{"secret"}, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})

	t.Run("two args returns no completion", func(t *testing.T) {
		result, directive := cmd.ValidArgsFunction(cmd, []string{"secret", "key"}, "")
		assert.Nil(t, result)
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	})
}
