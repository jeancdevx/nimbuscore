package workspace

type EngineConfig struct {
	Namespace    string
	ResticImage  string
	RelayImage   string
	IngressHost  string
	TLSSecret    string
	Snapshot     SnapshotConfig
	Ingress      IngressConfig
	Relay        RelayConfig
	IdleTimeout  int
}
