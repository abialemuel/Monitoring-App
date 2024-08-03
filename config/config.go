package config

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config interface {
	Init(configPath string) error
	Get() *MainConfig
}

type config struct {
	Config *MainConfig
}

func New() Config {
	return &config{
		Config: &MainConfig{},
	}
}

func (c *config) Init(configPath string) error {
	if err := c.load(c.Config, ".", configPath); err != nil {
		return err
	}
	err := validator.New().Struct(c.Config)
	if err != nil {
		return err
	}

	return nil
}

func (c *config) Get() *MainConfig {
	return c.Config
}

func (c *config) load(cfg any, path string, configPath string) error {

	// default value
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("server.listen", ":8881")
	viper.SetDefault("broker.primary.host", []string{"localhost:9092"})
	viper.SetDefault("broker.primary.group", "controller-new")
	viper.SetDefault("broker.primary.topics", []string{"public"})
	viper.SetDefault("broker.primary.timeout.connect", 5)
	viper.SetDefault("broker.primary.timeout.transaction", 30)
	viper.SetDefault("broker.primary.mechanism", "PLAIN")
	viper.SetDefault("broker.primary.protocol", "PLAINTEXT")
	viper.SetDefault("broker.primary.pool.producer", 10)
	viper.SetDefault("broker.primary.pool.consumer", 10)

	viper.AddConfigPath(path)
	if configPath != "" {
		viper.SetConfigFile(configPath)
	}
	viper.SetConfigType("yaml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if len(configPath) > 0 {
			return err
		}
	}
	return viper.Unmarshal(&cfg)
}
