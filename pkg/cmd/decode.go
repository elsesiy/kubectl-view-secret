package cmd

import (
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

type SecretDecoder interface {
	Decode(input string) (string, error)
}

func (s Secret) Decode(input string) (string, error) {
	switch s.Type {
	// TODO handle all secret types
	case Opaque:
		b64d, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			return "", nil
		}
		return string(b64d), nil
	case Helm:
		b64dk8s, _ := base64.StdEncoding.DecodeString(input)
		b64dhelm, _ := base64.StdEncoding.DecodeString(string(b64dk8s))

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
