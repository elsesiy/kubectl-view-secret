package cmd

import (
	"bytes"
	"fmt"
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
