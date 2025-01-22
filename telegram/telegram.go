package telegram

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type TelegramBot struct {
	Token  string `json:"token"`
	client *http.Client
}

type ReplyMarkup struct {
	ForceReply bool `json:"force_reply"`
	Selective  bool `json:"selective"`
}

type MessageOptions struct {
	ChatID      string      `json:"chat_id"`
	Text        string      `json:"text"`
	ParseMode   string      `json:"parse_mode"`
	ReplyMarkup ReplyMarkup `json:"reply_markup"`
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

	// Encode special characters
	options.Text = strings.ReplaceAll(options.Text, "_", "\\_")
	options.Text = strings.ReplaceAll(options.Text, ".", ".")

	// Encode message
	options.Text = url.QueryEscape(options.Text)

	_, err := t.client.Get("https://api.telegram.org/bot" + t.Token + "/sendMessage?chat_id=" + options.ChatID + "&parse_mode=" + options.ParseMode + "&forceReply=" + strconv.FormatBool(options.ReplyMarkup.ForceReply) + "&text=" + options.Text)
	if err != nil {
		return err
	}

	return nil
}
