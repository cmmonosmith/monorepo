package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"

	"github.com/cmmonosmith/cuddle-bot/asciify"
)

// A bot <TODO>.
type bot struct {
	name string
	id   string
}

// New constructs a bot instance with the name from the host environment and the user ID from an active session.
func New(name string, id string) *bot {
	return &bot{
		name: name,
		id:   id,
	}
}

// newMessage handles Discord MESSAGE_CREATE events, looking for messages that start with direct mentions of the bot followed
// by specific keywords or commands.
// TODO: Unrecognized messages may be handled by context-specific handlers, e.g. when a user is playing a text adventure in a
// specific channel and doesn't need to mention the bot.
func (b *bot) newMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	// ignore own messages
	if b.id == message.Author.ID {
		return
	}

	// respond to user message if it starts with an @me
	parts := strings.Split(message.Content, " ")
	if parts[0] != fmt.Sprintf("<@%s>", b.id) {
		return
	}

	// debug print the message
	json, err := json.MarshalIndent(message, "", "    ")
	if err != nil {
		slog.Error("failed to marshal message", slog.Any("error", err))
	}
	slog.Debug("message=" + string(json))

	// evaluate the rest of the message
	if len(parts) == 1 {
		channelMessageSend(session, message.ChannelID, "you have to tell me what you want :weary:")
		return
	}
	parts = parts[1:]
	switch {
	case parts[0] == "hi":
		channelMessageSend(session, message.ChannelID, "sup sup :sunglasses:")
	case parts[0] == "asciify":
		b.asciify(session, message, false)
	case parts[0] == "asciifile":
		b.asciify(session, message, true)
	default:
		channelMessageSend(session, message.ChannelID, "sorry, i don't follow :sweat_smile:")
	}
}

// asciify checks for a png attachment, downloads it to a randomly named file, passes it to the asciify package function, then
// deletes the download
func (b *bot) asciify(session *discordgo.Session, message *discordgo.MessageCreate, toFile bool) {
	if len(message.Attachments) == 0 {
		channelMessageSend(session, message.ChannelID, "i can't asciify what you don't send me :disappointed:")
		return
	} else if len(message.Attachments) > 1 {
		channelMessageSend(session, message.ChannelID, "only send me one attachment, please... :weary:")
		return
	}
	attachment := message.Attachments[0]
	if attachment.ContentType != "image/png" {
		channelMessageSend(session, message.ChannelID, "i can only asciify .png attachments :weary:")
		return
	}

	filename := fmt.Sprintf("%s.png", uuid.New().String())
	if err := b.download(attachment.URL, filename); err != nil {
		slog.Error("failed to download attachment", slog.Any("error", err))
		channelMessageSend(session, message.ChannelID, ":x: sorry, i couldn't download your image :grimmace:")
		return
	}
	defer os.Remove(filename)

	maxWidth, maxHeight := 60, 30 // default keeps it well under the 2000 character limit for non-Nitro messages
	if toFile {
		maxWidth, maxHeight = 256, 128
	}
	ascii, err := asciify.Asciify(filename, maxWidth, maxHeight)
	if err != nil {
		slog.Error("failed to asciify attachment", slog.Any("error", err))
		channelMessageSend(session, message.ChannelID, ":x: sorry, i couldn't asciify that :grimmace:")
		return
	}
	if toFile {
		outFilename := fmt.Sprintf("%s.txt", filename[:strings.LastIndex(filename, ".")])
		if err := b.createTxt(outFilename, ascii); err != nil {
			channelMessageSend(session, message.ChannelID, ":x: sorry, i couldn't write that file :grimmace:")
		}
		defer os.Remove(outFilename)
		channelMessageSendWithFile(session, message.ChannelID, ":white_check_mark: asciified file: :nerd:", outFilename)
	} else {
		channelMessageSend(session, message.ChannelID, fmt.Sprintf(":white_check_mark: asciified: :nerd:\n```%s```", ascii))
	}
}

// download fetches the attachment from the Discord cdn and writes it to disk
func (b *bot) download(url string, filename string) error {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// createTxt creates a .txt file whose contents are a given string
func (b *bot) createTxt(filename string, content string) error {
	out, err := os.Create(filename)
	if err != nil {
		slog.Error("failed to create asciified file", slog.Any("error", err))
		return err
	}
	defer out.Close()
	io.WriteString(out, content)
	return nil
}

// interactionCreate handles Discord INTERACTION_CREATE events, specifically new application "slash" commands.
func (b *bot) interactionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	slog.Debug("created", slog.Any("interaction", interaction))

	// ignore interactions that aren't application commands for now
	if interaction.Type != discordgo.InteractionApplicationCommand {
		slog.Info("ignoring non-application command interaction")
		return
	}

	// see if we can get a user from the interaction, and make sure it isn't us
	var userID string
	if interaction.User != nil {
		userID = interaction.User.ID
	} else if interaction.Member != nil {
		userID = interaction.Member.User.ID
	} else {
		slog.Error("no user available in interaction")
		return
	}
	slog.Debug("interaction", slog.Any("user", userID))
	if b.id == userID {
		slog.Error("somehow got command from self")
		return
	}

	// read the command data
	data := interaction.ApplicationCommandData()
	json, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		slog.Error("failed to marshal interaction application command data", slog.Any("error", err))
	}
	slog.Debug("interaction application command data=" + string(json))

	// respond to the command
	err = session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: ":question: you know as much as I do, dawg...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		slog.Error("failed to respond to interaction", slog.Any("error", err))
	}
}

// registerCommands tells Discord what application "slash" commands users can call when interacting with the bot, which also
// provides the users with auto-complete and tooltips.
func (b *bot) registerCommands(session *discordgo.Session) {
	command := &discordgo.ApplicationCommand{
		Name:        b.name,
		Description: fmt.Sprintf("Information about %s", b.name),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "command",
				Description: fmt.Sprintf("Get information about %s", b.name),
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "help",
						Value: "help",
					},
				},
			},
		},
	}
	_, err := session.ApplicationCommandCreate(instance.id, "", command)
	if err != nil {
		slog.Error("failed to create slash command", slog.Any("error", err))
	}
}
