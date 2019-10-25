package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/magiconair/properties/assert"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var testSecret = `
{
    "data": {
        "TEST_PASSWORD": "c2VjcmV0Cg==",
        "TEST_PASSWORD_2": "dmVyeXNlY3JldAo="
    }
}
`

var testSecretSingle = `
{
    "data": {
        "SINGLE_PASSWORD": "c2VjcmV0Cg=="
    }
}
`

func TestProcessSecret(t *testing.T) {
	var secret, secretSingle map[string]interface{}
	_ = json.Unmarshal([]byte(testSecret), &secret)
	_ = json.Unmarshal([]byte(testSecretSingle), &secretSingle)

	tests := map[string]struct {
		secretData map[string]interface{}
		wantStdOut []string
		wantStdErr []string
		secretKey  string
		decodeAll  bool
		err        error
	}{
		"view-secret <secret>":            {secret, []string{"-> TEST_PASSWORD", "-> TEST_PASSWORD_2"}, []string{listDescription}, "", false, nil},
		"view-secret <secret-single-key>": {secretSingle, []string{"secret"}, []string{fmt.Sprintf(singleKeyDescription, "SINGLE_PASSWORD")}, "", false, nil},
		"view-secret test TEST_PASSWORD":  {secret, []string{"secret"}, nil, "TEST_PASSWORD", false, nil},
		"view-secret test -a":             {secret, []string{"", "", "", "", "TEST_PASSWORD=secret", "TEST_PASSWORD_2=verysecret"}, nil, "", true, nil},
		"view-secret test NONE":           {secret, nil, nil, "NONE", false, ErrSecretKeyNotFound},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotStdOut := bytes.Buffer{}
			gotStdErr := bytes.Buffer{}
			err := ProcessSecret(&gotStdOut, &gotStdErr, test.secretData, test.secretKey, test.decodeAll)

			if test.err != nil {
				assert.Equal(t, err, test.err)
			} else {
				var gotStdOutArr, gotStdErrArr []string
				scanner := bufio.NewScanner(strings.NewReader(gotStdOut.String()))
				for scanner.Scan() {
					gotStdOutArr = append(gotStdOutArr, scanner.Text())
				}

				scanner = bufio.NewScanner(strings.NewReader(gotStdErr.String()))
				for scanner.Scan() {
					gotStdErrArr = append(gotStdErrArr, scanner.Text())
				}

				sort.Strings(gotStdOutArr)
				sort.Strings(gotStdErrArr)

				if !reflect.DeepEqual(gotStdOutArr, test.wantStdOut) {
					t.Errorf("got %v, want %v", gotStdOutArr, test.wantStdOut)
				}

				if !reflect.DeepEqual(gotStdErrArr, test.wantStdErr) {
					t.Errorf("got %v, want %v", gotStdErrArr, test.wantStdErr)
				}
			}

		})
	}
}
