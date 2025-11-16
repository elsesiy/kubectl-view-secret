package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	secret = SecretData{
		"TEST_CONN_STR":   "bW9uZ29kYjovL215REJSZWFkZXI6RDFmZmljdWx0UCU0MHNzdzByZEBtb25nb2RiMC5leGFtcGxlLmNvbToyNzAxNy8/YXV0aFNvdXJjZT1hZG1pbg==",
		"TEST_PASSWORD":   "c2VjcmV0Cg==",
		"TEST_PASSWORD_2": "dmVyeXNlY3JldAo=",
	}

	secretSingle = SecretData{
		"SINGLE_PASSWORD": "c2VjcmV0Cg==",
	}

	// echo "helm-test" | gzip -c | base64 | base64
	secretHelm = SecretData{
		"release": "SDRzSUFGb2FlR2NBQTh0SXpjblZMVWt0THVFQ0FQdWt3aHdLQUFBQQo=",
	}

	secretEmpty = SecretData{}
)

func TestParseArgs(t *testing.T) {
	opts := CommandOpts{}
	tests := map[string]struct {
		opts     CommandOpts
		args     []string
		wantOpts CommandOpts
	}{
		"one arg":  {opts, []string{"test"}, CommandOpts{secretName: "test"}},
		"two args": {opts, []string{"test", "key"}, CommandOpts{secretName: "test", secretKey: "key"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			test.opts.ParseArgs(test.args)
			got := test.opts
			if got != test.wantOpts {
				t.Errorf("got %v, want %v", got, test.wantOpts)
			}
		})
	}
}

func TestNewCmdViewSecret(t *testing.T) {
	tests := map[string]struct {
		args     []string
		feedkeys string
		want     string
		wantErr  error
	}{
		"all":                        {args: []string{"test", "--all"}, want: `key1='value1'\nkey2='value2'`},
		"custom ctx":                 {args: []string{"test", "--context", "gotest"}, wantErr: errors.New("Error in configuration: context was not found for specified context: gotest\nError: kubectl command failed: exit status 1")},
		"custom kubecfg":             {args: []string{"test", "--kubeconfig", "cfg"}, wantErr: errors.New("error: stat cfg: no such file or directory\nError: kubectl command failed: exit status 1")},
		"custom ns (does not exist)": {args: []string{"test", "--namespace", "bob"}, wantErr: errors.New("Error from server (NotFound): namespaces \"bob\" not found\nError: kubectl command failed: exit status 1")},
		"custom ns (no secret)":      {args: []string{"test", "--namespace", "another"}, wantErr: errors.New("Error from server (NotFound): secrets \"test\" not found\nError: kubectl command failed: exit status 1")},
		"custom ns (valid secret)":   {args: []string{"gopher", "--namespace", "another"}, want: `Viewing only available key: foo\nbar`},
		"helm":                       {args: []string{"test3", "--namespace", "helm"}, want: `Viewing only available key: release\nhelm-test`},
		"impersonate group":          {args: []string{"test", "--all", "--as", "gopher"}, want: `key1='value1'\nkey2='value2'`},
		"impersonate user & group":   {args: []string{"test", "--all", "--as", "gopher", "--as-group", "golovers"}, want: `key1='value1'\nkey2='value2'`},
		// make bootstrap sources 2 test secrets in the default namespace, select the first one and print all values
		"interactive":                       {args: []string{"--all"}, feedkeys: "\r", want: `key1='value1'\nkey2='value2'`},
		"interactive custom ns (no secret)": {args: []string{"--namespace", "empty"}, wantErr: ErrNoSecretFound},
		"invalid arg count":                 {args: []string{"a", "b", "c"}, wantErr: errors.New("accepts between 0 and 2 arg(s), received 3")},
		"quiet":                             {args: []string{"test2", "--quiet"}, want: `value1`},
		"unknown flag":                      {args: []string{"--version"}, wantErr: errors.New("unknown flag: --version")},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := NewCmdViewSecret()
			outBuf := bytes.NewBufferString("")
			readBuf := &strings.Reader{}
			if tt.feedkeys != "" {
				readBuf = strings.NewReader(tt.feedkeys)
			}

			cmd.SetOut(outBuf)
			cmd.SetIn(readBuf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				if tt.wantErr == nil {
					assert.Fail(t, "unexpected error", err)
				} else if err.Error() != tt.wantErr.Error() {
					t.Errorf("error message mismatch:\nexpected: %q\nactual: %q", tt.wantErr.Error(), err.Error())
				}
				return
			} else if tt.wantErr != nil {
				assert.Fail(t, "expected error, got nil", tt.wantErr)
				return
			}

			_, err = io.ReadAll(outBuf)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProcessSecret(t *testing.T) {
	tests := map[string]struct {
		secretData SecretData
		secretType SecretType
		wantStdOut []string
		wantStdErr []string
		secretKey  string
		decodeAll  bool
		err        error
		feedkeys   string
	}{
		"view-secret <secret>": {
			secret,
			Opaque,
			[]string{
				"TEST_CONN_STR='mongodb://myDBReader:D1fficultP%40ssw0rd@mongodb0.example.com:27017/?authSource=admin'",
				"TEST_PASSWORD='secret\n'",
				"TEST_PASSWORD_2='verysecret\n'",
			},
			[]string{},
			"",
			false,
			nil,
			"\r", // selects 'all' as it's the default selection
		},
		"view-secret <secret-single-key>": {
			secretSingle,
			Opaque,
			[]string{"secret"},
			[]string{fmt.Sprintf(singleKeyDescription, "SINGLE_PASSWORD")},
			"",
			false,
			nil,
			"",
		},
		"view-secret <helm-secret>": {
			secretHelm,
			Helm,
			[]string{"helm-test"},
			[]string{fmt.Sprintf(singleKeyDescription, "release")},
			"",
			false,
			nil,
			"",
		},
		"view-secret test TEST_PASSWORD": {
			secret,
			Opaque,
			[]string{"secret"},
			nil,
			"TEST_PASSWORD",
			false,
			nil,
			"",
		},
		"view-secret test -a": {
			secret,
			Opaque,
			[]string{
				"TEST_CONN_STR='mongodb://myDBReader:D1fficultP%40ssw0rd@mongodb0.example.com:27017/?authSource=admin'",
				"TEST_PASSWORD='secret\n'",
				"TEST_PASSWORD_2='verysecret\n'",
			},
			nil,
			"",
			true,
			nil,
			"",
		},
		"view-secret test NONE": {
			secret,
			Opaque,
			nil,
			nil,
			"NONE",
			false,
			ErrSecretKeyNotFound,
			"",
		},
		"view-secret <secret-empty>": {
			secretEmpty,
			Opaque,
			nil,
			nil,
			"",
			false,
			ErrSecretEmpty,
			"",
		},
		"view-secret <secret> select specific key": {
			secret,
			Opaque,
			[]string{"secret"},
			[]string{},
			"",
			false,
			nil,
			"\x1b[B\x1b[B\r", // navigate to TEST_PASSWORD and select
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stdOutBuf := bytes.Buffer{}
			stdErrBuf := bytes.Buffer{}
			readBuf := strings.Reader{}

			if test.feedkeys != "" {
				readBuf = *strings.NewReader(test.feedkeys)
			}

			err := ProcessSecret(&stdOutBuf, &stdErrBuf, &readBuf, Secret{Data: test.secretData, Type: test.secretType}, test.secretKey, test.decodeAll)

			if test.err != nil {
				assert.Equal(t, err, test.err)
			} else {
				gotStdOut := stdOutBuf.String()
				gotStdErr := stdErrBuf.String()

				for _, s := range test.wantStdOut {
					if !assert.Contains(t, gotStdOut, s) {
						t.Errorf("got %v, want %v", gotStdOut, s)
					}
				}

				for _, s := range test.wantStdErr {
					if !assert.Contains(t, gotStdErr, s) {
						t.Errorf("got %v, want %v", gotStdErr, s)
					}
				}
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tests := map[string]struct {
		name        string
		setupCmd    func(*CommandOpts)
		expectError bool
	}{
		"invalid namespace": {
			name: "invalid namespace",
			setupCmd: func(opts *CommandOpts) {
				opts.secretName = "test"
				opts.customNamespace = "invalid-namespace"
			},
			expectError: true,
		},
		"nonexistent secret": {
			name: "nonexistent secret",
			setupCmd: func(opts *CommandOpts) {
				opts.secretName = "nonexistent"
			},
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			opts := &CommandOpts{}
			tt.setupCmd(opts)
		})
	}
}

func TestOutputText(t *testing.T) {
	tests := map[string]struct {
		decodedData map[string]string
		want        string
	}{
		"single key": {
			map[string]string{"key": "value"},
			"value\n",
		},
		"multiple keys": {
			map[string]string{"key1": "value1", "key2": "value2"},
			"key1='value1'\nkey2='value2'\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputText(&buf, tt.decodedData)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestCompletionFlagRegistration(t *testing.T) {
	cmd := NewCmdViewSecret()

	// Check that namespace flag has completion registered
	namespaceFlag := cmd.Flags().Lookup("namespace")
	assert.NotNil(t, namespaceFlag, "namespace flag should exist")

	// Check that ValidArgsFunction is set
	assert.NotNil(t, cmd.ValidArgsFunction, "ValidArgsFunction should be set")
}
