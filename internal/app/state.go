package app

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// AddonID is the id the panel matches against /v2/info and the release file.
const AddonID = "fortumers-subs-bot"

// Version is overwritten at build time with the release tag.
var Version = "dev"

const (
	maxWelcome = 3500
	maxButtons = 12
	maxBtnText = 40
	maxReply   = 1000
	maxURL     = 300
	maxChats   = 800
	maxName    = 64
)

// Button is one row of the bot's inline keyboard.
// URL and Reply are exclusive: a link opens, a reply is sent back.
type Button struct {
	Text  string `json:"text"`
	URL   string `json:"url,omitempty"`
	Reply string `json:"reply,omitempty"`
}

// Settings is what the admin page stores.
type Settings struct {
	Welcome   string   `json:"welcome"`
	Autoreply string   `json:"autoreply"`
	Buttons   []Button `json:"buttons"`
}

// Chat is someone who wrote to the bot.
type Chat struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	At   int64  `json:"at"`
}

// DefaultSettings is used until the admin saves their own.
func DefaultSettings() Settings {
	return Settings{
		Welcome:   "Привет. Это бот подписчиков.\nВыберите кнопку или напишите сообщение — так вы попадёте в список рассылки.",
		Autoreply: "Сообщение получено. Если нужен ответ, дождитесь администратора.",
		Buttons: []Button{{
			Text:  "Что умеет бот",
			Reply: "Бот присылает новости от администратора панели и отвечает на кнопки меню.",
		}},
	}
}

// Normalize trims fields and rejects anything the bot must not send.
func (s *Settings) Normalize() error {
	s.Welcome = strings.TrimSpace(s.Welcome)
	s.Autoreply = strings.TrimSpace(s.Autoreply)
	if s.Welcome == "" {
		return errors.New("текст приветствия пустой")
	}
	if utf8.RuneCountInString(s.Welcome) > maxWelcome || utf8.RuneCountInString(s.Autoreply) > maxWelcome {
		return errors.New("текст длиннее 3500 символов")
	}
	if len(s.Buttons) > maxButtons {
		return errors.New("кнопок больше 12")
	}
	out := make([]Button, 0, len(s.Buttons))
	for _, b := range s.Buttons {
		b.Text = strings.TrimSpace(b.Text)
		b.URL = strings.TrimSpace(b.URL)
		b.Reply = strings.TrimSpace(b.Reply)
		if b.Text == "" {
			return errors.New("у кнопки нет подписи")
		}
		if utf8.RuneCountInString(b.Text) > maxBtnText {
			return errors.New("подпись кнопки длиннее 40 символов")
		}
		if b.URL != "" && b.Reply != "" {
			return errors.New("у кнопки и ссылка, и ответ — оставьте что-то одно")
		}
		if b.URL == "" && b.Reply == "" {
			return errors.New("кнопке нужна ссылка или текст ответа")
		}
		if b.URL != "" {
			if !strings.HasPrefix(b.URL, "https://") || strings.ContainsAny(b.URL, " \t\r\n") || len(b.URL) > maxURL {
				return errors.New("ссылка кнопки должна начинаться с https://")
			}
		}
		if utf8.RuneCountInString(b.Reply) > maxReply {
			return errors.New("ответ кнопки длиннее 1000 символов")
		}
		out = append(out, b)
	}
	s.Buttons = out
	return nil
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
