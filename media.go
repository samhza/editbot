package main

import (
	"errors"
	"mime"
	"path"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
)

type Media struct {
	URL    string
	Height int
	Width  int
	Type   mediaType
}

type mediaType int

const (
	mediaImage mediaType = iota
	mediaVideo
	mediaGIFV
	mediaGIF
)

func (b *Bot) findMedia(id discord.ChannelID) (*Media, error) {
	msgs, err := b.s.Messages(id, 25)
	if err != nil {
		return nil, err
	}
	var media *Media
	for _, m := range msgs {
		media = b.getMsgMedia(m)
		if media != nil {
			return media, nil
		}
	}
	return nil, errors.New("no media found")
}

func (b *Bot) getMsgMedia(m discord.Message) *Media {
	for _, at := range m.Attachments {
		if at.Height == 0 {
			continue
		}
		ext := path.Ext(at.Proxy)
		m := &Media{
			URL:    at.Proxy,
			Height: int(at.Height),
			Width:  int(at.Width),
			Type:   mediaTypeByExt(ext),
		}
		return m
	}
	for _, em := range m.Embeds {
		if em.Type == discord.VideoEmbed && em.Provider == nil {
			return &Media{
				URL:    em.Video.URL,
				Height: int(em.Video.Height),
				Width:  int(em.Video.Width),
				Type:   mediaVideo,
			}
		}
		if em.Type == discord.ImageEmbed {
			m := &Media{
				URL:    em.Thumbnail.Proxy,
				Height: int(em.Thumbnail.Height),
				Width:  int(em.Thumbnail.Width),
			}
			m.Type = mediaTypeByExt(path.Ext(m.URL))
			return m
		}
		if em.Type == discord.GIFVEmbed {
			m := &Media{
				Height: int(em.Video.Height),
				Width:  int(em.Video.Width),
				URL:    em.Video.URL,
				Type:   mediaGIFV,
			}
			return m
		}
	}
	return nil
}

func mediaTypeByExt(ext string) mediaType {
	mime := mime.TypeByExtension(ext)
	switch {
	case mime == "image/gif":
		return mediaGIF
	case strings.HasPrefix(mime, "video/"):
		return mediaVideo
	case strings.HasPrefix(mime, "image/"):
		return mediaImage
	}
	return mediaImage
}
