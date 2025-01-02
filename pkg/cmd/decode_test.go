package cmd

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	tests := map[string]struct {
		data func() Secret
		want string
	}{
		// base64 encoded
		"opaque": {
			func() Secret {
				return Secret{
					Data: map[string]string{
						"key": "dGVzdAo=",
					},
					Type: Opaque,
				}
			},
			"test\n",
		},
		// double base64 encoded + gzip'd
		"helm": {
			func() Secret {
				res := Secret{
					Type: Helm,
				}
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				if _, err := gz.Write([]byte("test\n")); err != nil {
					return res
				}
				if err := gz.Close(); err != nil {
					return res
				}

				b64k8s := base64.StdEncoding.EncodeToString(buf.Bytes())
				b64helm := base64.StdEncoding.EncodeToString([]byte(b64k8s))

				res.Data = SecretData{
					"key": b64helm,
				}

				return res
			},
			"test\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := tt.data()
			want := tt.want

			got, err := data.Decode(data.Data["key"])
			if err != nil {
				t.Errorf("got %v, want %v", got, want)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
