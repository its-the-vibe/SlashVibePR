package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all runtime configuration for the service.
type Config struct {
	RedisAddr                  string
	RedisPassword              string
	RedisChannel               string
	RedisViewSubmissionChannel string
	RedisPoppitList            string
	RedisPoppitOutputChannel   string
	RedisSlackLinerList        string
	SlackBotToken              string
	SlackChannelID             string
	GitHubOrg                  string
	LogLevel                   string
}

// configFile mirrors the structure of config.yaml. All fields have sensible
// defaults so a minimal config file only needs to override what differs from
// the defaults.
type configFile struct {
	Redis struct {
		Addr string `yaml:"addr"`
	} `yaml:"redis"`
	Channels struct {
		SlashCommands   string `yaml:"slash_commands"`
		ViewSubmissions string `yaml:"view_submissions"`
		PoppitOutput    string `yaml:"poppit_output"`
	} `yaml:"channels"`
	Lists struct {
		PoppitCommands     string `yaml:"poppit_commands"`
		SlackLinerMessages string `yaml:"slackliner_messages"`
	} `yaml:"lists"`
	Slack struct {
		ChannelID string `yaml:"channel_id"`
	} `yaml:"slack"`
	GitHub struct {
		Org string `yaml:"org"`
	} `yaml:"github"`
	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// defaultConfigFile returns a configFile pre-populated with built-in defaults.
func defaultConfigFile() configFile {
	var cf configFile
	cf.Redis.Addr = "host.docker.internal:6379"
	cf.Channels.SlashCommands = "slack-commands"
	cf.Channels.ViewSubmissions = "slack-relay-view-submission"
	cf.Channels.PoppitOutput = "poppit:command-output"
	cf.Lists.PoppitCommands = "poppit:commands"
	cf.Lists.SlackLinerMessages = "slack_messages"
	cf.Logging.Level = "INFO"
	return cf
}

// loadConfig reads non-secret configuration from the YAML config file (default
// path: config.yaml, overridable via CONFIG_FILE) and the two secrets
// (REDIS_PASSWORD, SLACK_BOT_TOKEN) from environment variables.
func loadConfig() Config {
	cfgPath := getEnv("CONFIG_FILE", "config.yaml")

	cf := defaultConfigFile()

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		// If the config file is missing, fall back to defaults. The service
		// will still require the two secret env vars to be set.
		Warn("Config file %q not found, using built-in defaults: %v", cfgPath, err)
	} else if err = yaml.Unmarshal(data, &cf); err != nil {
		Fatal("Failed to parse config file %q: %v", cfgPath, err)
	}

	return Config{
		RedisAddr:                  cf.Redis.Addr,
		RedisPassword:              os.Getenv("REDIS_PASSWORD"),
		RedisChannel:               cf.Channels.SlashCommands,
		RedisViewSubmissionChannel: cf.Channels.ViewSubmissions,
		RedisPoppitList:            cf.Lists.PoppitCommands,
		RedisPoppitOutputChannel:   cf.Channels.PoppitOutput,
		RedisSlackLinerList:        cf.Lists.SlackLinerMessages,
		SlackBotToken:              os.Getenv("SLACK_BOT_TOKEN"),
		SlackChannelID:             cf.Slack.ChannelID,
		GitHubOrg:                  cf.GitHub.Org,
		LogLevel:                   cf.Logging.Level,
	}
}

// getEnv returns the value of an environment variable or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadConfigFromBytes parses YAML bytes into a configFile and merges with
// defaults, returning the resulting Config. Secrets are taken from the
// supplied redisPassword and slackBotToken arguments rather than from the
// environment so that tests remain hermetic.
func loadConfigFromBytes(data []byte, redisPassword, slackBotToken string) (Config, error) {
	cf := defaultConfigFile()

	if err := yaml.Unmarshal(data, &cf); err != nil {
		return Config{}, fmt.Errorf("yaml parse error: %w", err)
	}

	return Config{
		RedisAddr:                  cf.Redis.Addr,
		RedisPassword:              redisPassword,
		RedisChannel:               cf.Channels.SlashCommands,
		RedisViewSubmissionChannel: cf.Channels.ViewSubmissions,
		RedisPoppitList:            cf.Lists.PoppitCommands,
		RedisPoppitOutputChannel:   cf.Channels.PoppitOutput,
		RedisSlackLinerList:        cf.Lists.SlackLinerMessages,
		SlackBotToken:              slackBotToken,
		SlackChannelID:             cf.Slack.ChannelID,
		GitHubOrg:                  cf.GitHub.Org,
		LogLevel:                   cf.Logging.Level,
	}, nil
}
