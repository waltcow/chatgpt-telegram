package config

import (
	"bytes"
	"errors"
	"log"
	"os"

	"github.com/spf13/viper"
)

type EnvConfig struct {
	TelegramID      []int64 `mapstructure:"TELEGRAM_ID"`
	TelegramToken   string  `mapstructure:"TELEGRAM_TOKEN"`
	EditWaitSeconds int     `mapstructure:"EDIT_WAIT_SECONDS"`
	OpenAISession   string  `mapstructure:"OPENAI_SESSION"`
}

// emptyConfig is used to initialize viper.
// It is required to register config keys with viper when in case no config file is provided.
const emptyConfig = `TELEGRAM_ID=
TELEGRAM_TOKEN=
EDIT_WAIT_SECONDS=
OPENAI_SESSION=`

func (e *EnvConfig) HasTelegramID(id int64) bool {
	for _, v := range e.TelegramID {
		if v == id {
			return true
		}
	}
	return false
}

// LoadEnvConfig loads config from .env file, variables from environment take precedence if provided.
// If no .env file is provided, config is loaded from environment variables.
func LoadEnvConfig(path string) (*EnvConfig, error) {
	fileExists := fileExists(path)
	if !fileExists {
		log.Printf("config file %s does not exist, using env variables", path)
	}

	v := viper.New()
	v.SetConfigType("env")
	v.AutomaticEnv()
	if err := v.ReadConfig(bytes.NewBufferString(emptyConfig)); err != nil {
		return nil, err
	}
	if fileExists {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}
	}

	var cfg EnvConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	}
	return true
}

func (e *EnvConfig) ValidateWithDefaults() error {
	if e.TelegramToken == "" {
		return errors.New("TELEGRAM_TOKEN is not set")
	}
	if len(e.TelegramID) == 0 {
		log.Printf("TELEGRAM_ID is not set, all users will be able to use the bot")
	}
	if e.EditWaitSeconds < 0 {
		log.Printf("EDIT_WAIT_SECONDS not set, defaulting to 1")
		e.EditWaitSeconds = 1
	}
	if e.OpenAISession == "" {
		log.Printf("OPENAI_SESSION not set, defaulting to empty")
		return errors.New("OPENAI_SESSION is not set")
	}
	return nil
}

func UpdateEnvConfig(path string, key string, value interface{}) (*EnvConfig, error) {
	fileExists := fileExists(path)
	if !fileExists {
		log.Printf("config file %s does not exist, using env variables", path)
	}

	v := viper.New()
	v.SetConfigType("env")
	v.AutomaticEnv()
	if err := v.ReadConfig(bytes.NewBufferString(emptyConfig)); err != nil {
		return nil, err
	}
	if fileExists {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}
	}

	var cfg EnvConfig

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	switch key {
	case "TELEGRAM_ID":
		cfg.TelegramID = append(cfg.TelegramID, value.(int64))
	case "TELEGRAM_TOKEN":
		cfg.TelegramToken = value.(string)
	case "EDIT_WAIT_SECONDS":
		cfg.EditWaitSeconds = value.(int)
	case "OPENAI_SESSION":
		cfg.OpenAISession = value.(string)
	}

	// Write to file
	err := v.WriteConfig()

	if err != nil {
		log.Printf("Error writing config file: %s", err)
		return nil, err
	}

	return &cfg, nil
}
