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
func Run(name string, token string) int {
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
	instance = New(name, session.State.User.ID)
	instance.registerCommands(session)

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
	instance.newMessage(session, message)
}

// interactionCreate is the handler for Discord's INTERACTION_CREATE event, which simply calls the bot's implementation.
func interactionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	instance.interactionCreate(session, interaction)
}

// channelMessageSend wraps the session function to log any errors
func channelMessageSend(session *discordgo.Session, channelID string, message string) {
	if _, err := session.ChannelMessageSend(channelID, message); err != nil {
		slog.Error("failed to send channel message", slog.Any("error", err))
	}
}

func channelMessageSendWithFile(session *discordgo.Session, channelID string, message string, filename string) {
	// open up the image from disk
	reader, err := os.Open(filename)
	if err != nil {
		slog.Error("failed to open file for sending", slog.Any("error", err))
		channelMessageSend(session, channelID, ":x: sorry, i couldn't open the file to send :grimmace:")
		return
	}
	defer reader.Close()
	session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{File: &discordgo.File{Name: filename, Reader: reader}, Content: message})
}
