package cmd

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	validSecretJson = `{
    "apiVersion": "v1",
    "data": {
        "key1": "dmFsdWUxCg==",
        "key2": "dmFsdWUyCg=="
    },
    "kind": "Secret",
    "metadata": {
        "creationTimestamp": "2024-08-02T21:25:40Z",
        "name": "test",
        "namespace": "default",
        "resourceVersion": "715",
        "uid": "0027fdc9-5371-4715-a0a8-61f3f78fdd36"
    },
    "type": "Opaque"
}`

	helmSecretJson = `{
	  "apiVersion": "v1",
	   "data": {
         "release": "blob"
		 },
	   "kind": "Secret",
	   "metadata": {
	       "name": "sh.helm.release.v1.wordpress.v1",
	       "namespace": "default"
	   },
	   "type": "helm.sh/release.v1"
	}`

	invalidSecretJson = `{
    "apiVersion": "v1",
    "data": {},
    "kind": "Secret",
    "metadata": {
        "name": "test-empty",
        "namespace": "default",
    },
    "type": "Opaque"
}`
)

func TestSerialize(t *testing.T) {
	tests := map[string]struct {
		input   string
		want    Secret
		wantErr error
	}{
		"empty opaque secret": {
			input: invalidSecretJson,
			want: Secret{
				Metadata: Metadata{
					Name:      "test",
					Namespace: "default",
				},
				Type: Opaque,
			},
			wantErr: errors.New("invalid character '}' looking for beginning of object key string"),
		},
		"valid opaque secret": {
			input: validSecretJson,
			want: Secret{
				Data: SecretData{
					"key1": "dmFsdWUxCg==",
					"key2": "dmFsdWUyCg==",
				},
				Metadata: Metadata{
					Name:      "test",
					Namespace: "default",
				},
				Type: Opaque,
			},
		},
		"valid helm secret": {
			input: helmSecretJson,
			want: Secret{
				Data: SecretData{
					"release": "blob",
				},
				Metadata: Metadata{
					Name:      "sh.helm.release.v1.wordpress.v1",
					Namespace: "default",
				},
				Type: Helm,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got Secret
			err := json.Unmarshal([]byte(tt.input), &got)
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

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatal(err)
			}
		})
	}
}
