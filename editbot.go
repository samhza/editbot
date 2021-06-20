package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"go.samhza.com/editbot/interactions"
	"go.samhza.com/esammy/vedit"
)

type Bot struct {
	s *state.State
	i *interactions.Client
}

func NewBot(token string, appid discord.AppID) (*Bot, error) {
	b := new(Bot)
	var err error
	b.s, err = state.NewWithIntents("Bot "+token, gateway.IntentGuilds, gateway.IntentGuildMessages)
	if err != nil {
		return nil, err
	}
	b.s.AddHandler(b.handleInteraction)
	b.i = interactions.New(appid)
	return b, nil
}

func (bot *Bot) handleInteraction(e *gateway.InteractionCreateEvent) {
	switch e.Data.Name {
	case "edit":
		bot.s.RespondInteraction(e.ID, e.Token,
			api.InteractionResponse{Type: api.DeferredMessageInteractionWithSource})
		out, err := bot.edit(e)
		if err != nil {
			bot.i.EditInitial(e.Token,
				webhook.EditMessageData{Content: option.NewNullableString(err.Error())})
			return
		}
		bot.i.EditInitial(e.Token,
			webhook.EditMessageData{Files: []sendpart.File{{"out.mp4", out}}})
	}
}

func (bot *Bot) edit(e *gateway.InteractionCreateEvent) (*os.File, error) {
	args := new(vedit.Arguments)
	args.Parse(e.Data.Options[0].Value)
	media, err := bot.findMedia(e.ChannelID)
	if err != nil {
		return nil, err
	}
	var itype vedit.InputType
	switch media.Type {
	case mediaImage, mediaGIF, mediaGIFV:
		itype = vedit.InputImage
	case mediaVideo:
		itype = vedit.InputVideo
	}
	resp, err := http.Get(media.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	tmp, err := os.CreateTemp("", "esammy.*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	_, err = io.Copy(tmp, resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	out, err := vedit.Process(*args, itype, tmp)
	if err != nil {
		return nil, err
	}
	return out, err
}

func (b *Bot) RegisterCommands(appID discord.AppID, guildID discord.GuildID) error {
	commands, err := b.s.GuildCommands(appID, guildID)
	if err != nil {
		return err
	}

	for _, command := range commands {
		log.Println("Existing command", command.Name, "found.")
	}

	newCommands := []api.CreateCommandData{
		{
			Name:        "edit",
			Description: "edits a video",
			Options: []discord.CommandOption{
				{
					Name:        "edits",
					Type:        discord.StringOption,
					Required:    true,
					Description: "edits to run on the video",
				},
			},
		},
	}

	for _, command := range newCommands {
		_, err := b.s.CreateGuildCommand(appID, guildID, command)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func (b *Bot) Start(ctx context.Context) error {
	return b.s.Open(ctx)
}

func main() {
	appID := discord.AppID(mustSnowflakeEnv("APP_ID"))
	guildID := discord.GuildID(mustSnowflakeEnv("GUILD_ID"))

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("No $BOT_TOKEN given.")
	}
	bot, err := NewBot(token, appID)
	if err != nil {
		log.Fatalln("creating state:", err)
	}
	err = bot.RegisterCommands(appID, guildID)
	if err != nil {
		log.Fatalln("registering commands:", err)
	}
	err = bot.Start(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	select {}
}

func mustSnowflakeEnv(env string) discord.Snowflake {
	s, err := discord.ParseSnowflake(os.Getenv(env))
	if err != nil {
		log.Fatalf("Invalid snowflake for $%s: %v", env, err)
	}
	return s
}
