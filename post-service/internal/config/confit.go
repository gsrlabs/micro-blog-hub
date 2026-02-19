package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App     AppConfig     `mapstructure:"app"`
	Mongo   MongoConfig   `mapstructure:"mongo"`
	Redis   RedisConfig   `mapstructure:"redis"`
	GRPС    GRPCConfig    `mapstructure:"grpc"`
	Logging LoggingConfig `mapstructure:"logging"`
}

type AppConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type MongoConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	DB   string `mapstructure:"db"`
}

type RedisConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type GRPCConfig struct {
	AuthHost string `mapstructure:"auth_host"`
	AuthPort string `mapstructure:"auth_port"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.AutomaticEnv()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = v.BindEnv("app.port", "POST_SERVICE_APP_PORT")

	_ = v.BindEnv("mongo.host", "MONGO_HOST")
	_ = v.BindEnv("mongo.port", "MONGO_PORT")
	_ = v.BindEnv("mongo.db", "MONGO_DB")

	_ = v.BindEnv("redis.host", "REDIS_HOST")
	_ = v.BindEnv("redis.port", "REDIS_PORT")

	_ = v.BindEnv("grpc.auth_host", "AUTH_GRPC_HOST")
	_ = v.BindEnv("grpc.auth_port", "AUTH_GRPC_PORT")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.App.Port == "" {
		return fmt.Errorf("POST_SERVICE_APP_PORT is required")
	}

	if c.Mongo.Host == "" {
		return fmt.Errorf("MONGO_HOST is required")
	}
	if c.Mongo.Port == "" {
		return fmt.Errorf("MONGO_PORT is required")
	}
	if c.Mongo.DB == "" {
		return fmt.Errorf("MONGO_DB is required")
	}

	if c.Redis.Host == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	if c.Redis.Port == "" {
		return fmt.Errorf("REDIS_PORT is required")
	}

	if c.GRPС.AuthHost == "" {
		return fmt.Errorf("AUTH_GRPC_HOS is required")
	}
	if c.GRPС.AuthPort == "" {
		return fmt.Errorf("AUTH_GRPC_PORT is required")
	}

	return nil
}
