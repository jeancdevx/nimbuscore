package api

type APIConfig struct {
	Host         string
	Port         string
	LogLevel     string
	IngressHost  string

	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	Auth     AuthConfig
}

type DatabaseConfig struct {
	URL      string
	MaxConns int
	MinConns int
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type RabbitMQConfig struct {
	URL string
}

type AuthConfig struct {
	Issuer         string
	ExternalIssuer string
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	JWTSecret      string
	SessionTTL     int
}
