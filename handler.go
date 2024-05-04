package slogslack

import (
	"context"
	"time"

	"log/slog"

	slogcommon "github.com/samber/slog-common"
	"github.com/slack-go/slack"
)

type Option struct {
	// log level (default: debug)
	Level slog.Leveler

	// slack webhook url
	WebhookURL string
	// slack bot token
	BotToken string
	// slack channel (default: webhook channel)
	Channel string
	// bot username (default: webhook username)
	Username string
	// bot emoji (default: webhook emoji)
	IconEmoji string
	// bot emoji (default: webhook emoji)
	IconURL string

	Timeout time.Duration // default: 10s

	// optional: customize Slack message builder
	Converter Converter

	// optional: see slog.HandlerOptions
	AddSource   bool
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

func (o Option) NewSlackHandler() slog.Handler {
	if o.Level == nil {
		o.Level = slog.LevelDebug
	}

	if o.WebhookURL == "" && o.BotToken == "" {
		panic("missing Slack webhook url and bot token")
	}

	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}

	if o.Converter == nil {
		o.Converter = DefaultConverter
	}

	return &SlackHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

var _ slog.Handler = (*SlackHandler)(nil)

type SlackHandler struct {
	option Option
	attrs  []slog.Attr
	groups []string
}

func (h *SlackHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *SlackHandler) Handle(ctx context.Context, record slog.Record) error {
	message := h.option.Converter(h.option.AddSource, h.option.ReplaceAttr, h.attrs, h.groups, &record)

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

	if ts := ContextThreadTimestamp(ctx); ts != "" {
		message.ThreadTimestamp = ts
		if ContextReplyBroadcast(ctx) {
			message.ReplyBroadcast = true
		}
	}

	go func() {
		_ = h.postMessage(message)
	}()

	return nil
}

func (h *SlackHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SlackHandler{
		option: h.option,
		attrs:  slogcommon.AppendAttrsToGroup(h.groups, h.attrs, attrs...),
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

func (h *SlackHandler) postMessage(message *slack.WebhookMessage) error {
	if h.option.WebhookURL != "" {
		return slack.PostWebhook(h.option.WebhookURL, message)
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.option.Timeout)
	defer cancel()

	options := []slack.MsgOption{
		slack.MsgOptionText(message.Text, true),
		slack.MsgOptionAttachments(message.Attachments...),
		slack.MsgOptionUsername(message.Username),
		slack.MsgOptionIconURL(message.IconURL),
		slack.MsgOptionIconEmoji(message.IconEmoji),
	}
	if message.ThreadTimestamp != "" {
		options = append(options, slack.MsgOptionTS(message.ThreadTimestamp))
	}
	if message.ReplyBroadcast {
		options = append(options, slack.MsgOptionBroadcast())
	}

	_, _, err := slack.New(h.option.BotToken).PostMessageContext(ctx, message.Channel, options...)
	return err
}
