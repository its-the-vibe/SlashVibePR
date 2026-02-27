package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/slack-go/slack"
)

func main() {
	config := loadConfig()

	SetLogLevel(config.LogLevel)

	if config.SlackBotToken == "" {
		Fatal("SLACK_BOT_TOKEN environment variable is required")
	}
	if config.SlackChannelID == "" {
		Fatal("slack.channel_id must be set in config.yaml")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       0,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		Fatal("Failed to connect to Redis: %v", err)
	}
	Info("Connected to Redis at %s", config.RedisAddr)

	slackClient := slack.New(config.SlackBotToken)

	go subscribeToSlashCommands(ctx, rdb, slackClient, config)
	go subscribeToViewSubmissions(ctx, rdb, slackClient, config)
	go subscribeToPoppitOutput(ctx, rdb, slackClient, config)

	log.Println("SlashVibePR service started")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	Info("Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}
