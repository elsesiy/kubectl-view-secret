package cmd

// SecretList represents a list of secrets
type SecretList struct {
	Items []Secret `json:"items"`
}

// Secret represents a kubernetes secret
type Secret struct {
	Data     SecretData `json:"data"`
	Metadata Metadata   `json:"metadata"`
	Type     SecretType `json:"type"`
}

// SecretData represents the data of a secret
type SecretData map[string]string

// Metadata represents the metadata of a secret
type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// SecretType represents the type of a secret
//
// Opaque	arbitrary user-defined data
// kubernetes.io/service-account-token	ServiceAccount token
// kubernetes.io/dockercfg	serialized ~/.dockercfg file
// kubernetes.io/dockerconfigjson	serialized ~/.docker/config.json file
// kubernetes.io/basic-auth	credentials for basic authentication
// kubernetes.io/ssh-auth	credentials for SSH authentication
// kubernetes.io/tls	data for a TLS client or server
// bootstrap.kubernetes.io/token	bootstrap token data
// helm.sh/release.v1	Helm v3 release data
//
// refs:
// - https://kubernetes.io/docs/concepts/configuration/secret/#secret-types
// - https://gist.github.com/DzeryCZ/c4adf39d4a1a99ae6e594a183628eaee
type SecretType string

const (
	BasicAuth           SecretType = "kubernetes.io/basic-auth"
	DockerCfg           SecretType = "kubernetes.io/dockercfg"
	DockerConfigJson    SecretType = "kubernetes.io/dockerconfigjson"
	Helm                SecretType = "helm.sh/release.v1"
	Opaque              SecretType = "Opaque"
	ServiceAccountToken SecretType = "kubernetes.io/service-account-token"
	SshAuth             SecretType = "kubernetes.io/ssh-auth"
	Tls                 SecretType = "kubernetes.io/tls"
	Token               SecretType = "bootstrap.kubernetes.io/token"
)
