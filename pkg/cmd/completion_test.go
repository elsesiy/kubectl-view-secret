package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeKubectl puts a kubectl on PATH that records its arguments, prints stdout and exits with code
func fakeKubectl(t *testing.T, stdout string, code int) (argsFile string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake kubectl is a shell script")
	}

	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\nprintf '%s' '" + stdout + "'\nexit " + strconv.Itoa(code) + "\n"
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return argsFile
}

func TestCompletion(t *testing.T) {
	flags := []string{"-n", "ns", "-c", "ctx", "--as", "bob", "--as-group", "g1", "--as-group", "g2"}
	wantFlags := "--namespace ns --context ctx --as bob --as-group g1 --as-group g2"

	tests := map[string]struct {
		args     []string
		stdout   string
		exitCode int
		want     []string
		wantArgs string
	}{
		"secrets": {
			args:     nil,
			stdout:   "alpha   Opaque   2   5d\nbeta   helm.sh/release.v1   1   2h\ngamma   Opaque   1   1m\n",
			want:     []string{"alpha", "beta", "gamma"},
			wantArgs: "get secrets --no-headers " + wantFlags,
		},
		"keys of a secret are sorted": {
			args:     []string{"mysecret"},
			stdout:   `{"b":"x","a":"y"}`,
			want:     []string{"a", "b"},
			wantArgs: "get secret mysecret -o jsonpath={.data} " + wantFlags,
		},
		"kubectl failure": {
			args:     nil,
			exitCode: 1,
			want:     nil,
			wantArgs: "get secrets --no-headers " + wantFlags,
		},
		"no completion after secret and key": {
			args: []string{"mysecret", "key"},
			want: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			argsFile := fakeKubectl(t, tt.stdout, tt.exitCode)

			cmd := NewCmdViewSecret()
			assert.NoError(t, cmd.ParseFlags(flags))

			got, _ := cmd.ValidArgsFunction(cmd, tt.args, "")
			assert.Equal(t, tt.want, got)

			recorded, _ := os.ReadFile(argsFile)
			assert.Equal(t, tt.wantArgs, strings.Join(strings.Fields(string(recorded)), " "))
		})
	}
}
