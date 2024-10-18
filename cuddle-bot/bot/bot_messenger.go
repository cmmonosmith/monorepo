package bot

import (
	"log/slog"
	"os"

	"github.com/bwmarrin/discordgo"
)

type messenger struct {
	s *discordgo.Session
}

func newMessenger(session *discordgo.Session) *messenger {
	return &messenger{
		s: session,
	}
}

// channelMessageSend wraps the session ChannelMessageSend function to log any errors
func (m *messenger) channelMessageSend(channelID string, message string) {
	if _, err := m.s.ChannelMessageSend(channelID, message); err != nil {
		slog.Error("failed to send channel message", slog.Any("error", err))
	}
}

// channelMessageSendWithFile wraps the session ChannelMessageSendComplex function to attach a file and log any errors
func (m *messenger) channelMessageSendWithFile(channelID string, message string, filename string) {
	reader, err := os.Open(filename)
	if err != nil {
		slog.Error("failed to open file for sending", slog.Any("error", err))
		m.channelMessageSend(channelID, ":x: sorry, i couldn't open the file to send :grimmace:")
		return
	}
	defer reader.Close()
	m.s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{File: &discordgo.File{Name: filename, Reader: reader}, Content: message})
}
