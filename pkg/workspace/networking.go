package workspace

type IngressConfig struct {
	Enabled   bool   `json:"enabled"`
	Host      string `json:"host"`
	TLSSecret string `json:"tls_secret,omitempty"`
}

type PortForwardRule struct {
	Name      string `json:"name"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Subdomain string `json:"subdomain"`
}

type RelayConfig struct {
	Enabled bool   `json:"enabled"`
	Image   string `json:"image"`
	Port    int    `json:"port"`
}
