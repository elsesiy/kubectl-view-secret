package cmd

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SecretDecoder is an interface for decoding various kubernetes secret resources
type SecretDecoder interface {
	Decode(input string) (string, error)
}

// decodeBase64 performs standard base64 decoding
func decodeBase64(input string) (string, error) {
	b64d, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}
	return string(b64d), nil
}

// Decode decodes a secret based on its type
//
// Supports various Kubernetes secret types including:
// - Opaque: standard base64 encoded data
// - Helm: double base64 encoded + gzip compressed
// - TLS: PEM encoded certificates and keys
// - Docker config: JSON configuration data
// - SSH: private key data
// - Basic auth: username/password pairs
// - Service account tokens: JWT tokens
func (s Secret) Decode(input string) (string, error) {
	switch s.Type {
	case Helm:
		return s.decodeHelm(input)
	case DockerCfg, DockerConfigJSON:
		return s.decodeDockerConfig(input)
	case Opaque, TLS, SSHAuth, BasicAuth, ServiceAccountToken, Token:
		return decodeBase64(input)
	default:
		// Unknown secret type - try base64 decoding with enhanced error message
		result, err := decodeBase64(input)
		if err != nil {
			return "", fmt.Errorf("couldn't decode unknown secret type %q: %w", s.Type, err)
		}
		return result, nil
	}
}

// decodeHelm decodes Helm release secrets
func (s Secret) decodeHelm(input string) (string, error) {
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
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()

	data, err := io.ReadAll(gz)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// decodeDockerConfig decodes Docker configuration secrets
func (s Secret) decodeDockerConfig(input string) (string, error) {
	b64d, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	// Try to pretty-print JSON if it's valid
	var jsonData any
	if json.Unmarshal(b64d, &jsonData) == nil {
		prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
		if err == nil {
			return string(prettyJSON), nil
		}
	}

	return string(b64d), nil
}
