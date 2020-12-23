package keycloakctl

type Components []Component

type Component struct {
	Name            string           `json:"name,omitempty"`
	ProviderID      string           `json:"providerId,omitempty"`
	ProviderType    string           `json:"providerType,omitempty"`
	ParentID        string           `json:"parentId,omitempty"`
	ComponentConfig *ComponentConfig `json:"config,omitempty"`
	SubType         string           `json:"subType,omitempty"`
}

type ComponentConfig struct {
	Priority    []string `json:"priority,omitempty"`
	Enabled     []string `json:"enabled,omitempty"`
	Active      []string `json:"active,omitempty"`
	Algorithm   []string `json:"algorithm,omitempty"`
	PrivateKey  []string `json:"privateKey,omitempty"`
	Certificate []string `json:"certificate,omitempty"`
}
