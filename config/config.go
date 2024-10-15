package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ProjectName string `env:"PROJECT_NAME" env-default:"Unknown (please set PROJECT_NAME)"`

	GrpcHost string `env:"GRPC_HOST"    env-default:"localhost"`
	GrpcPort string `env:"GRPC_PORT"    env-default:"5001"`

	PostgresHost     string `env:"POSTGRES_HOST"     env-default:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT"     env-default:"5432"`
	PostgresUser     string `env:"POSTGRES_USER"     env-default:"postgres"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" env-default:"postgres"`
	PostgresDatabase string `env:"POSTGRES_DATABASE" env-default:"sentinel"`

	TelegramBotToken    string  `env:"TELEGRAM_BOT_TOKEN" env-required:"true"`  // no default
	TelegramsRecipients []int64 `env:"TELEGRAM_RECIPIENTS" env-required:"true"` // no default

	AlertCooldownMinutes int      `env:"ALERT_COOLDOWN_MINUTES" env-default:"5"`
	AlertVisibleDetails  []string `env:"ALERT_VISIBLE_DETAILS" env-default:"trace_id,user_id"`
}

func LoadConfig() (Config, error) {
	var cfg Config

	err := cleanenv.ReadConfig(".env", &cfg)
	if err != nil {
		return cfg, fmt.Errorf("LoadConfig: %w", err)
	}

	return cfg, nil
}
