package main

import (
	"log/slog"
	"os"

	"github.com/cmmonosmith/cuddle-dev/bot"
)

func main() {
	name := os.Getenv("DISCORD_DEV_NAME")
	if name == "" {
		slog.Error("DISCORD_DEV_NAME must be set")
		os.Exit(1)
	}
	token := os.Getenv("DISCORD_DEV_TOKEN")
	if token == "" {
		slog.Error("DISCORD_DEV_TOKEN must be set")
		os.Exit(1)
	}
	logDebug := os.Getenv("DISCORD_DEV_LOG_DEBUG")
	if logDebug == "1" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	os.Exit(bot.Run(name, token))
}
