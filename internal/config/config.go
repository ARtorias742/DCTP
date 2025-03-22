package config

import "github.com/spf13/viper"

type Config struct {
	APIPort      string `mapstructure:"API_PORT"`
	RedisAddr    string `mapstructure:"REDIS_ADDR`
	DBConnection string `mapstructure:"DB_CONNECTION"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
