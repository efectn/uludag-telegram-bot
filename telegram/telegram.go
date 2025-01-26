package telegram

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

type TelegramBot struct {
	client *http.Client
	Token  string
}

type ReplyMarkup struct {
	ForceReply bool
	Selective  bool
}

type MessageOptions struct {
	ChatID      string
	Text        string
	ParseMode   string
	ReplyMarkup ReplyMarkup
}

func NewTelegramBot(token string) *TelegramBot {
	return &TelegramBot{
		Token:  token,
		client: &http.Client{},
	}
}

func (t *TelegramBot) SendMessage(options MessageOptions) error {
	// Validate config fields
	if options.ParseMode == "" {
		options.ParseMode = "markdown"
	}

	if options.ChatID == "" {
		return errors.New("chat_id is required")
	}

	// Send message
	values := url.Values{}
	values.Add("chat_id", options.ChatID)
	values.Add("text", options.Text)
	values.Add("parse_mode", options.ParseMode)
	values.Add("force_reply", strconv.FormatBool(options.ReplyMarkup.ForceReply))

	_, err := t.client.Get("https://api.telegram.org/bot" + t.Token + "/sendMessage?" + values.Encode())
	if err != nil {
		return err
	}

	return nil
}
