package main

import (
	"os"
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
	LogLevel                   string
}

// loadConfig reads configuration from environment variables, applying defaults
// where values are not set.
func loadConfig() Config {
	return Config{
		RedisAddr:                  getEnv("REDIS_ADDR", "host.docker.internal:6379"),
		RedisPassword:              getEnv("REDIS_PASSWORD", ""),
		RedisChannel:               getEnv("REDIS_CHANNEL", "slack-commands"),
		RedisViewSubmissionChannel: getEnv("REDIS_VIEW_SUBMISSION_CHANNEL", "slack-relay-view-submission"),
		RedisPoppitList:            getEnv("REDIS_POPPIT_LIST", "poppit:commands"),
		RedisPoppitOutputChannel:   getEnv("REDIS_POPPIT_OUTPUT_CHANNEL", "poppit:command-output"),
		RedisSlackLinerList:        getEnv("REDIS_SLACKLINER_LIST", "slack_messages"),
		SlackBotToken:              getEnv("SLACK_BOT_TOKEN", ""),
		SlackChannelID:             getEnv("SLACK_CHANNEL_ID", ""),
		LogLevel:                   getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
