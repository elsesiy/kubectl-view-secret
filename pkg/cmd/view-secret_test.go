package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/magiconair/properties/assert"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var testSecret = `
{
    "apiVersion": "v1",
    "data": {
        "TEST_PASSWORD": "c2VjcmV0Cg==",
        "TEST_PASSWORD_2": "dmVyeXNlY3JldAo="
    },
    "kind": "Secret",
    "metadata": {
        "creationTimestamp": "2019-09-16T17:57:46Z",
        "labels": {
            "app": "test"
        },
        "name": "test",
        "namespace": "test",
        "resourceVersion": "1",
        "selfLink": "/api/v1/namespaces/test/secrets/test",
		"uid": "00000000-0000-0000-0000-000000000000"
    },
    "type": "Opaque"
}
`

func TestProcessSecret(t *testing.T) {
	var secret map[string]interface{}
	_ = json.Unmarshal([]byte(testSecret), &secret)

	tests := map[string]struct {
		want      []string
		secretKey string
		decodeAll bool
		err       error
	}{
		"view-secret <secret>":           {[]string{"TEST_PASSWORD", "TEST_PASSWORD_2"}, "", false, nil,},
		"view-secret test TEST_PASSWORD": {[]string{"", "secret"}, "TEST_PASSWORD", false, nil},
		"view-secret test -a":            {[]string{"", "", "TEST_PASSWORD=secret", "TEST_PASSWORD_2=verysecret"}, "", true, nil},
		"view-secret test NONE":          {nil, "NONE", false, ErrSecretKeyNotFound},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := bytes.Buffer{}
			err := ProcessSecret(&got, secret, test.secretKey, test.decodeAll)

			if test.err != nil {
				assert.Equal(t, err, test.err)
			} else {
				var gotArr []string
				scanner := bufio.NewScanner(strings.NewReader(got.String()))
				for scanner.Scan() {
					gotArr = append(gotArr, scanner.Text())
				}

				sort.Strings(gotArr)

				if !reflect.DeepEqual(gotArr, test.want) {
					t.Errorf("got %v, want %v", gotArr, test.want)
				}
			}

		})
	}
}
