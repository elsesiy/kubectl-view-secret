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
		data    func() Secret
		want    string
		wantErr error
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
			nil,
		},
		// base64 encoded
		"opaque invalid": {
			func() Secret {
				return Secret{
					Data: map[string]string{
						"key": "dGVzdAo}}}=",
					},
					Type: Opaque,
				}
			},
			"",
			base64.CorruptInputError(7),
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
			nil,
		},
		// basic auth via default case
		"basic auth": {
			func() Secret {
				return Secret{
					Data: map[string]string{
						"key": "dGVzdAo=",
					},
					Type: BasicAuth,
				}
			},
			"test\n",
			nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := tt.data()

			got, err := data.Decode(data.Data["key"])
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

			assert.Equal(t, tt.want, got)
		})
	}
}
