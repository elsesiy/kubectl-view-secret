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
		"custom ctx":                 {args: []string{"test", "--context", "gotest"}},
		"custom kubecfg":             {args: []string{"test", "--kubeconfig", "cfg"}},
		"custom ns (does not exist)": {args: []string{"test", "--namespace", "bob"}, want: `Error from server (NotFound): namespaces "bob" not found`},
		"custom ns (no secret)":      {args: []string{"test", "--namespace", "another"}, want: `Error from server (NotFound): secrets "test" not found`},
		"custom ns (valid secret)":   {args: []string{"gopher", "--namespace", "another"}, want: `Viewing only available key: foo\nbar`},
		"impersonate group":          {args: []string{"test", "--as", "gopher"}},
		"impersonate user & group":   {args: []string{"test", "--as", "gopher", "--as-group", "golovers"}},
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
					assert.Equal(t, tt.wantErr, err)
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
		wantStdOut []string
		wantStdErr []string
		secretKey  string
		decodeAll  bool
		err        error
		feedkeys   string
	}{
		"view-secret <secret>": {
			secret,
			[]string{
				"TEST_CONN_STR='mongodb://myDBReader:D1fficultP%40ssw0rd@mongodb0.example.com:27017/?authSource=admin'",
				"TEST_PASSWORD='secret'",
				"TEST_PASSWORD_2='verysecret'",
			},
			[]string{},
			"",
			false,
			nil,
			"\r", // selects 'all' as it's the default selection
		},
		"view-secret <secret-single-key>": {
			secretSingle,
			[]string{"secret"},
			[]string{fmt.Sprintf(singleKeyDescription, "SINGLE_PASSWORD")},
			"",
			false,
			nil,
			"",
		},
		"view-secret test TEST_PASSWORD": {secret, []string{"secret"}, nil, "TEST_PASSWORD", false, nil, ""},
		"view-secret test -a": {
			secret,
			[]string{
				"TEST_CONN_STR='mongodb://myDBReader:D1fficultP%40ssw0rd@mongodb0.example.com:27017/?authSource=admin'",
				"TEST_PASSWORD='secret'",
				"TEST_PASSWORD_2='verysecret'",
			},
			nil,
			"",
			true,
			nil,
			"",
		},
		"view-secret test NONE":      {secret, nil, nil, "NONE", false, ErrSecretKeyNotFound, ""},
		"view-secret <secret-empty>": {secretEmpty, nil, nil, "", false, ErrSecretEmpty, ""},
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

			err := ProcessSecret(&stdOutBuf, &stdErrBuf, &readBuf, Secret{Data: test.secretData}, test.secretKey, test.decodeAll)

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
