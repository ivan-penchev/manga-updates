package config

import "github.com/caarlos0/env/v10"

type Config struct {
	MangaNelGraphQLEndpoint    string `env:"API_ENDPOINT" envDefault:"https://api.mghcdn.com/graphql"`
	SeriesDataFolder           string `env:"SERIES_DATAFOLDER" envDefault:"$HOME/repos/manga-updates/data" envExpand:"true"`
	SendGridAPIKey             string `env:"SENDGRID_API_KEY"`
	SendGridTemplateId         string `env:"SENDGRID_TEMPLATE_ID"`
	SMTP2GOApiKey              string `env:"SMTP2GO_API_KEY"`
	SMTP2GOTemplateId          string `env:"SMTP2GO_TEMPLATE_ID"`
	NotificationRecipientEmail string `env:"NOTIFICATION_EMAIL_RECIPIENT,required"`
	NotificationSenderEmail    string `env:"NOTIFICATION_EMAIL_SENDER,required"`
}

func Load() (*Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
