package config

type Config struct {
	Port          int
	JWTSecret     string
	EncryptionKey string
	SkipAuth      bool
	LogLevel      string
}
