package cmd

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
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

	emptySecretJson = `{
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
		"empty secret": {
			input: emptySecretJson,
			want: Secret{
				Metadata: Metadata{
					Name:      "test",
					Namespace: "default",
				},
			},
			wantErr: errors.New("invalid character '}' looking for beginning of object key string"),
		},
		"valid secret": {
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
					t.Fatalf("unexpected error: %v", err)
				} else if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			} else if tt.wantErr != nil {
				t.Errorf("expected error %v, got nil", tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatal(err)
			}
		})
	}
}
