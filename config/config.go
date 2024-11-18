package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	AlertProviderDiscord  = "discord"
	AlertProviderTelegram = "telegram"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" env-required:"true"`

	GrpcHost string `env:"GRPC_HOST"    env-default:"localhost"`
	GrpcPort string `env:"GRPC_PORT"    env-default:"5001"`

	PostgresHost     string `env:"POSTGRES_HOST"     env-default:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT"     env-default:"5432"`
	PostgresUser     string `env:"POSTGRES_USER"     env-default:"postgres"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" env-required:"true"`
	PostgresDatabase string `env:"POSTGRES_DATABASE" env-default:"sentinel"`

	AlertProvider        string   `env:"ALERT_PROVIDER"         env-required:"true"`
	AlertCooldownMinutes int      `env:"ALERT_COOLDOWN_MINUTES" env-default:"5"`
	AlertVisibleDetails  []string `env:"ALERT_VISIBLE_DETAILS"  env-default:"trace_id,user_id"`

	TelegramBotToken string  `env:"TELEGRAM_BOT_TOKEN"`
	TelegramsChatIDs []int64 `env:"TELEGRAM_CHAT_IDS"`

	DiscordBotToken   string   `env:"DISCORD_BOT_TOKEN"`
	DiscordChannelIDs []string `env:"DISCORD_CHANNEL_IDS"`
}

func LoadConfig() (Config, error) {
	var cfg Config

	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return cfg, fmt.Errorf("LoadConfig: %w", err)
	}

	err = cfg.validate()
	if err != nil {
		return cfg, fmt.Errorf("LoadConfig: %w", err)
	}

	return cfg, nil
}

func (cfg Config) validate() error {
	// Validate provider type
	if cfg.AlertProvider != AlertProviderDiscord && cfg.AlertProvider != AlertProviderTelegram {
		return fmt.Errorf("Config.validate: invalid alert provider: %q. Choices are: %q, %q",
			cfg.AlertProvider, AlertProviderDiscord, AlertProviderTelegram)
	}

	// Validate token and chat/channel IDs based on provider
	if cfg.AlertProvider == AlertProviderTelegram {
		if cfg.TelegramBotToken == "" {
			return fmt.Errorf("Config.validate: TELEGRAM_BOT_TOKEN is required for Telegram alert provider")
		}
		if len(cfg.TelegramsChatIDs) == 0 {
			return fmt.Errorf("Config.validate: TELEGRAM_CHAT_IDS is required for Telegram alert provider")
		}
	}
	if cfg.AlertProvider == AlertProviderDiscord {
		if cfg.DiscordBotToken == "" {
			return fmt.Errorf("Config.validate: DISCORD_BOT_TOKEN is required for Discord alert provider")
		}
		if len(cfg.DiscordChannelIDs) == 0 {
			return fmt.Errorf("Config.validate: DISCORD_CHANNEL_IDS is required for Discord alert provider")
		}
	}

	return nil
}
