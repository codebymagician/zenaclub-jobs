package config

// Config stays in the same shape crm-jobs uses (a struct LoadConfig()
// returns) so a future job ported from crm-jobs -- one that needs Redis, GCP
// or Telegram config -- adds a field here rather than inventing a second
// config convention. Only LogLevel is populated today because
// subscription-due-charges.go is the only job that runs.
type Config struct {
	LogLevel string
}

func LoadConfig() *Config {
	return &Config{
		LogLevel: "INFO",
	}
}
