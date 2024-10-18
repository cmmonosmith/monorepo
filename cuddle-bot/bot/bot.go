package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"

	"github.com/cmmonosmith/cuddle-bot/asciify"
)

// A bot <TODO>.
type bot struct {
	name string
	id   string
	s    *discordgo.Session
	m    *messenger
}

const (
	cmdHelp      = "help"
	cmdHi        = "hi"
	cmdAsciify   = "asciify"
	cmdAsciifile = "asciifile"
)

var (
	//TODO: move commands, arguments, and help docs to config
	cmdHelps = map[string]string{
		cmdHelp:      "print this help text, or print more detailed help text for a specific command",
		cmdHi:        "respond to your casual greeting",
		cmdAsciify:   "convert a PNG or JPEG image to ascii directly in the response",
		cmdAsciifile: "convert a PNG or JPEG image to ascii and attach it to the response as a TXT file",
	}
	asciifyAttachmentTypes = []string{"image/png", "image/jpeg"}
)

// New constructs a bot instance with the name from the host environment and the user ID from an active session.
func newBot(session *discordgo.Session, messenger *messenger) *bot {
	return &bot{
		name: session.State.User.Username,
		id:   session.State.User.ID,
		s:    session,
		m:    messenger,
	}
}

// newMessage handles Discord MESSAGE_CREATE events, looking for messages that start with direct mentions of the bot followed
// by specific keywords or commands.
// TODO: Unrecognized messages may be handled by context-specific handlers, e.g. when a user is playing a text adventure in a
// specific channel and doesn't need to mention the bot.
func (b *bot) newMessage(message *discordgo.MessageCreate) {
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
		b.m.channelMessageSend(message.ChannelID, "you have to tell me what you want :weary:")
		return
	}
	parts = parts[1:]
	switch {
	case parts[0] == cmdHelp:
		b.help(message, parts)
	case parts[0] == cmdHi:
		b.m.channelMessageSend(message.ChannelID, "sup sup :sunglasses:")
	case parts[0] == cmdAsciify:
		b.asciify(message, parts, false)
	case parts[0] == cmdAsciifile:
		b.asciify(message, parts, true)
	default:
		b.m.channelMessageSend(message.ChannelID, "sorry, i don't follow :sweat_smile:")
	}
}

// printHelp sends the user a quick rundown of the available commands, or a specific command if one was supplied
func (b *bot) help(message *discordgo.MessageCreate, parts []string) {
	var sb strings.Builder
	sb.WriteString("```")

	// if no arguments passed to `help`
	if len(parts) == 1 {
		sb.WriteString(fmt.Sprintf("usage: @%s <command> [args ...]\n", b.name))
		sb.WriteString(fmt.Sprintf("       /%s <command> [args ...]\n\n", b.name))
		sb.WriteString(fmt.Sprintf("%s: A friendly Discord bot, for fun and development practice\n\n", b.name))
		sb.WriteString(fmt.Sprintf("%s listens for your mentions or slash commands and responds or acts accordingly\n\n", b.name))
		sb.WriteString("Commands:\n")
		for command, description := range cmdHelps {
			sb.WriteString(fmt.Sprintf("  %-16s%s\n", command, description))
		}
	} else {
		sb.WriteString("specific command help info not implemented yet...")
	}

	sb.WriteString("```")
	b.m.channelMessageSend(message.ChannelID, sb.String())
}

// asciify checks for a png attachment, downloads it to a randomly named file, passes it to the asciify package function, then
// deletes the download
func (b *bot) asciify(message *discordgo.MessageCreate, parts []string, toFile bool) {
	// validate parameters
	if len(message.Attachments) == 0 {
		b.m.channelMessageSend(message.ChannelID, fmt.Sprintf("i can't %s what you don't send me :disappointed:", parts[0]))
		return
	} else if len(message.Attachments) > 1 {
		b.m.channelMessageSend(message.ChannelID, "only send me one attachment, please... :weary:")
		return
	}
	attachment := message.Attachments[0]
	if !slices.Contains(asciifyAttachmentTypes, attachment.ContentType) {
		b.m.channelMessageSend(message.ChannelID, fmt.Sprintf("i can only %s png and jpeg attachments :weary:", parts[0]))
		return
	}
	maxWidth, maxHeight := 60, 30 // default keeps it well under the 2000 character limit for non-Nitro messages
	if toFile {
		maxWidth, maxHeight = 256, 128
	}
	if len(parts) > 1 {
		if len(parts) != 3 {
			b.m.channelMessageSend(message.ChannelID, fmt.Sprintf("ope, bad parameters, I need `%s [maxWidth maxHeight]` :face_with_open_eyes_and_hand_over_mouth:", parts[0]))
			return
		}
		width, err := strconv.Atoi(parts[1])
		if err != nil || width > 60 {
			b.m.channelMessageSend(message.ChannelID, fmt.Sprintf("ope, bad parameters, for `%s [maxWidth maxHeight]` I need `maxWidth` to be an integer no greater than %d :face_with_open_eyes_and_hand_over_mouth:", parts[0], maxWidth))
			return
		}
		height, err := strconv.Atoi(parts[2])
		if err != nil || height > 30 {
			b.m.channelMessageSend(message.ChannelID, fmt.Sprintf("ope, bad parameters, for `%s [maxWidth maxHeight]` I need `maxHeight` to be an integer no greater than %d :face_with_open_eyes_and_hand_over_mouth:", parts[0], maxHeight))
			return
		}
		maxWidth, maxHeight = width, height
	}

	// download attachment
	filename := fmt.Sprintf("%s.png", uuid.New().String())
	if err := b.download(attachment.URL, filename); err != nil {
		slog.Error("failed to download attachment", slog.Any("error", err))
		b.m.channelMessageSend(message.ChannelID, ":x: sorry, i couldn't download your image :grimmace:")
		return
	}
	defer os.Remove(filename)

	ascii, err := asciify.Asciify(filename, maxWidth, maxHeight)
	if err != nil {
		slog.Error("failed to asciify attachment", slog.Any("error", err))
		b.m.channelMessageSend(message.ChannelID, fmt.Sprintf(":x: sorry, i couldn't %s that :grimmace:", parts[0]))
		return
	}
	if toFile {
		outFilename := fmt.Sprintf("%s.txt", filename[:strings.LastIndex(filename, ".")])
		if err := b.createTxt(outFilename, ascii); err != nil {
			b.m.channelMessageSend(message.ChannelID, ":x: sorry, i couldn't write that file :grimmace:")
		}
		defer os.Remove(outFilename)
		b.m.channelMessageSendWithFile(message.ChannelID, ":white_check_mark: asciifiled: :nerd:", outFilename)
	} else {
		b.m.channelMessageSend(message.ChannelID, fmt.Sprintf(":white_check_mark: asciified: :nerd:\n```%s```", ascii))
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
func (b *bot) interactionCreate(interaction *discordgo.InteractionCreate) {
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
	err = b.s.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
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
func (b *bot) registerCommands() {
	command := &discordgo.ApplicationCommand{
		Name:        b.name,
		Description: fmt.Sprintf("A friendly Discord bot named %s", b.name),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "command",
				Description: fmt.Sprintf("Get information about %s", b.name),
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  cmdHelp,
						Value: cmdHelp,
					},
				},
			},
		},
	}
	_, err := b.s.ApplicationCommandCreate(instance.id, "", command)
	if err != nil {
		slog.Error("failed to create slash command", slog.Any("error", err))
	}
}
