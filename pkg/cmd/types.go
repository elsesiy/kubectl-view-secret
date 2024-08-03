package cmd

type SecretList struct {
	Items []Secret `json:"items"`
}

type Secret struct {
	Data     SecretData `json:"data"`
	Metadata Metadata   `json:"metadata"`
}

type SecretData map[string]string

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
