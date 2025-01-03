package cmd

import (
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// SecretDecoder is an interface for decoding various kubernetes secret resources
type SecretDecoder interface {
	Decode(input string) (string, error)
}

// Decode decodes a secret based on its type, currently supporting only Opaque and Helm secrets
func (s Secret) Decode(input string) (string, error) {
	switch s.Type {
	// TODO handle all secret types
	case Opaque:
		b64d, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return "", err
		}
		return string(b64d), nil
	case Helm:
		b64dk8s, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return "", err
		}
		b64dhelm, err := base64.StdEncoding.DecodeString(string(b64dk8s))
		if err != nil {
			return "", err
		}

		gz, err := gzip.NewReader(strings.NewReader(string(b64dhelm)))
		if err != nil {
			return "", err
		}
		defer gz.Close()

		s, err := io.ReadAll(gz)
		if err != nil {
			return "", err
		}

		return string(s), nil
	}

	return "", fmt.Errorf("couldn't decode unknown secret type %q", s.Type)
}
