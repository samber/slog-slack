package slogslack

import (
	"context"

	"github.com/slack-go/slack"
	"golang.org/x/exp/slog"
)

type Option struct {
	// log level (default: debug)
	Level slog.Leveler

	// slack webhook url
	WebhookURL string
	// slack channel (default: webhook channel)
	Channel string
	// bot username (default: webhook username)
	Username string
	// bot emoji (default: webhook emoji)
	IconEmoji string
	// bot emoji (default: webhook emoji)
	IconURL string

	// optional: customize Slack message builder
	Converter Converter
}

func (o Option) NewSlackHandler() slog.Handler {
	if o.Level == nil {
		o.Level = slog.LevelDebug
	}

	if o.WebhookURL == "" {
		panic("missing Slack webhook url")
	}

	return &SlackHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

type SlackHandler struct {
	option Option
	attrs  []slog.Attr
	groups []string
}

func (h *SlackHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *SlackHandler) Handle(ctx context.Context, record slog.Record) error {
	converter := DefaultConverter
	if h.option.Converter != nil {
		converter = h.option.Converter
	}

	message := converter(h.attrs, &record)

	if h.option.Channel != "" {
		message.Channel = h.option.Channel
	}

	if h.option.Username != "" {
		message.Username = h.option.Username
	}

	if h.option.IconEmoji != "" {
		message.IconEmoji = h.option.IconEmoji
	}

	if h.option.IconURL != "" {
		message.IconURL = h.option.IconURL
	}

	return slack.PostWebhook(h.option.WebhookURL, message)
}

func (h *SlackHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SlackHandler{
		option: h.option,
		attrs:  appendAttrsToGroup(h.groups, h.attrs, attrs),
		groups: h.groups,
	}
}

func (h *SlackHandler) WithGroup(name string) slog.Handler {
	return &SlackHandler{
		option: h.option,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}
