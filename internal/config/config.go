package config

import (
	"os"

	"github.com/caarlos0/env/v10"
	"gopkg.in/yaml.v3"
)

type Config struct {
	MangaNelGraphQLEndpoint string         `env:"API_ENDPOINT" yaml:"api_endpoint"`
	RemoteChromeURL         string         `env:"REMOTE_CHROME_URL" yaml:"remote_chrome_url"`
	SeriesDataFolder        string         `env:"SERIES_DATAFOLDER" yaml:"series_data_folder"`
	Notifier                NotifierConfig `yaml:"notifier"`
}

type NotifierConfig struct {
	RecipientEmail string         `env:"NOTIFICATION_EMAIL_RECIPIENT" yaml:"recipient_email"`
	SenderEmail    string         `env:"NOTIFICATION_EMAIL_SENDER" yaml:"sender_email"`
	SendGrid       SendGridConfig `yaml:"sendgrid"`
	SMTP2GO        SMTP2GOConfig  `yaml:"smtp2go"`
}

type SendGridConfig struct {
	APIKey     string `env:"SENDGRID_API_KEY" yaml:"api_key"`
	TemplateID string `env:"SENDGRID_TEMPLATE_ID" yaml:"template_id"`
}

type SMTP2GOConfig struct {
	APIKey     string `env:"SMTP2GO_API_KEY" yaml:"api_key"`
	TemplateID string `env:"SMTP2GO_TEMPLATE_ID" yaml:"template_id"`
}

func Load(configFile string) (*Config, error) {
	cfg := Config{
		MangaNelGraphQLEndpoint: "https://api.mghcdn.com/graphql",
		SeriesDataFolder:        os.ExpandEnv("$HOME/repos/manga-updates/data"),
	}

	// If configFile argument is empty, try to load from ENV
	if configFile == "" {
		configFile = os.Getenv("CONFIG_FILE")
	}

	// If we have a config file path (either from arg or env), try to load from it
	if configFile != "" {
		content, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		// Expand environment variables in the file content
		expandedContent := os.ExpandEnv(string(content))

		if err := yaml.Unmarshal([]byte(expandedContent), &cfg); err != nil {
			return nil, err
		}

		return &cfg, nil
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
