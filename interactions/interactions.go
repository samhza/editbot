package interactions

import (
	"net/http"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/webhook"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/pkg/errors"
)

type Client struct {
	*httputil.Client
	ID discord.AppID
}

func New(appid discord.AppID) *Client {
	return &Client{httputil.NewClient(), appid}
}

func (c *Client) DeleteMessage(tok string, id discord.MessageID) error {
	return c.deleteMessage(tok, id.String())
}

func (c *Client) DeleteInitial(tok string) error {
	return c.deleteMessage(tok, "@original")
}

func (c *Client) deleteMessage(tok, id string) error {
	return c.FastRequest(http.MethodDelete, api.EndpointWebhooks+c.ID.String()+"/"+tok+"/messages/"+id)
}

func (c *Client) EditMessage(tok string, id discord.MessageID, data webhook.EditMessageData) (*discord.Message, error) {
	return c.editMessage(tok, id.String(), data)
}

func (c *Client) EditInitial(tok string, data webhook.EditMessageData) (*discord.Message, error) {
	return c.editMessage(tok, "@original", data)
}

func (c *Client) editMessage(tok, id string, data webhook.EditMessageData) (*discord.Message, error) {
	if data.AllowedMentions != nil {
		if err := data.AllowedMentions.Verify(); err != nil {
			return nil, errors.Wrap(err, "allowedMentions error")
		}
	}
	if data.Embeds != nil {
		sum := 0
		for _, e := range *data.Embeds {
			if err := e.Validate(); err != nil {
				return nil, errors.Wrap(err, "embed error")
			}
			sum += e.Length()
			if sum > 6000 {
				return nil, &discord.OverboundError{sum, 6000, "sum of text in embeds"}
			}
		}
	}
	var msg *discord.Message
	return msg, sendpart.PATCH(c.Client, data, &msg,
		api.EndpointWebhooks+c.ID.String()+"/"+tok+"/messages/"+id)
}
