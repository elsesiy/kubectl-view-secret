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
		key     string // key to decode from the secret data
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
			"key",
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
			"key",
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
			"key",
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
			"key",
			"test\n",
			nil,
		},
		// test docker config json
		"docker config json": {
			func() Secret {
				dockerConfig := `{"auths":{"registry.example.com":{"auth":"dXNlcjpwYXNz"}}}`
				return Secret{
					Data: map[string]string{
						".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(dockerConfig)),
					},
					Type: DockerConfigJSON,
				}
			},
			".dockerconfigjson",
			`{
  "auths": {
    "registry.example.com": {
      "auth": "dXNlcjpwYXNz"
    }
  }
}`,
			nil,
		},
		// test docker config (legacy format)
		"docker config legacy": {
			func() Secret {
				dockerConfig := `{"registry.example.com":{"auth":"dXNlcjpwYXNz"}}`
				return Secret{
					Data: map[string]string{
						".dockercfg": base64.StdEncoding.EncodeToString([]byte(dockerConfig)),
					},
					Type: DockerCfg,
				}
			},
			".dockercfg",
			`{
  "registry.example.com": {
    "auth": "dXNlcjpwYXNz"
  }
}`,
			nil,
		},
		// test docker config with invalid json
		"docker config invalid json": {
			func() Secret {
				invalidJSON := `{"invalid": json}`
				return Secret{
					Data: map[string]string{
						".dockerconfigjson": base64.StdEncoding.EncodeToString([]byte(invalidJSON)),
					},
					Type: DockerConfigJSON,
				}
			},
			".dockerconfigjson",
			`{"invalid": json}`,
			nil,
		},
		// test unknown secret type
		"unknown type": {
			func() Secret {
				return Secret{
					Data: map[string]string{
						"key": "dGVzdAo=",
					},
					Type: "unknown",
				}
			},
			"key",
			"test\n",
			nil,
		},
		// test helm with invalid base64
		"helm invalid base64": {
			func() Secret {
				return Secret{
					Data: map[string]string{
						"key": "invalid-base64!",
					},
					Type: Helm,
				}
			},
			"key",
			"",
			base64.CorruptInputError(7), // Should return error for invalid base64
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := tt.data()

			got, err := data.Decode(data.Data[tt.key])
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
