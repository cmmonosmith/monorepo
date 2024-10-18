package main

import (
	"log/slog"
	"os"

	"github.com/cmmonosmith/cuddle-bot/bot"
)

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		slog.Error("DISCORD_BOT_TOKEN must be set")
		os.Exit(1)
	}
	logDebug := os.Getenv("DISCORD_BOT_LOG_DEBUG")
	if logDebug == "1" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	os.Exit(bot.Run(token))
}
