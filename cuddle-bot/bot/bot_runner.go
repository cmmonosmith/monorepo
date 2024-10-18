// Package bot encapsulates all Discord-specific behavior, like sessions and sending/receiving messages.
package bot

import (
	"log/slog"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

var (
	instance *bot
)

// Run creates and starts the Discord session. Once running, it waits for an interrupt signal, after which it will exit.
// Fatal errors elsewhere may forcibly exit without the signal.
func Run(token string) int {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("failed to create discordgo session", slog.Any("error", err))
		return 1
	}

	session.AddHandler(newMessage)
	session.AddHandler(interactionCreate)

	err = session.Open()
	if err != nil {
		slog.Error("failed to open session", slog.Any("error", err))
		return 1
	}
	defer session.Close()

	if session.State == nil || session.State.User == nil {
		slog.Error("no valid user in session")
		return 1
	}
	instance = newBot(session, newMessenger(session))
	instance.registerCommands()

	slog.Info("bot is running")
	waitForInterrupt()
	slog.Info("interrupt receieved, bot shutting down")
	return 0
}

// waitForInterrupt waits for the OS interrupt signal (Ctrl+C).
func waitForInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

// newMessage is the handler for Discord's MESSAGE_CREATE event, which simply calls the bot's implementation.
func newMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	instance.newMessage(message)
}

// interactionCreate is the handler for Discord's INTERACTION_CREATE event, which simply calls the bot's implementation.
func interactionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	instance.interactionCreate(interaction)
}
