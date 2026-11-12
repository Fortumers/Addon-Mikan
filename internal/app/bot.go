package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"
)

type tgUpdate struct {
	UpdateID int64       `json:"update_id"`
	Message  *tgMessage  `json:"message"`
	Callback *tgCallback `json:"callback_query"`
}

type tgMessage struct {
	Text string  `json:"text"`
	Chat tgChat  `json:"chat"`
	From *tgUser `json:"from"`
}

type tgChat struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type tgUser struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type tgCallback struct {
	ID      string     `json:"id"`
	Data    string     `json:"data"`
	From    tgUser     `json:"from"`
	Message *tgMessage `json:"message"`
}

func (a *App) runBot(ctx context.Context) {
	if !a.panel.botEnabled() {
		log.Printf("bot api is not configured; answering /v2 only")
		return
	}
	a.introduce(ctx)
	var offset int64
	var lastLog time.Time
	for {
		if ctx.Err() != nil {
			return
		}
		call, cancel := context.WithTimeout(ctx, 40*time.Second)
		var ups []tgUpdate
		err := a.panel.tg(call, "getUpdates", map[string]any{
			"offset":  offset,
			"timeout": 15,
			"limit":   50,
		}, &ups)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			a.setBot("", err)
			if time.Since(lastLog) > 30*time.Second {
				log.Printf("getUpdates: %v", err)
				lastLog = time.Now()
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}
		a.touchBot()
		for _, u := range ups {
			if u.UpdateID+1 > offset {
				offset = u.UpdateID + 1
			}
			a.handleUpdate(ctx, u)
		}
	}
}

func (a *App) introduce(ctx context.Context) {
	c, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_ = a.panel.tg(c, "deleteWebhook", map[string]any{}, nil)
	_ = a.panel.tg(c, "setMyCommands", map[string]any{
		"commands": []map[string]string{{
			"command":     "start",
			"description": "Открыть меню",
		}},
	}, nil)
	var me struct {
		Username string `json:"username"`
	}
	if err := a.panel.tg(c, "getMe", map[string]any{}, &me); err != nil {
		a.setBot("", err)
		log.Printf("getMe: %v", err)
		return
	}
	a.setBot(me.Username, nil)
	log.Printf("bot @%s", me.Username)
}

func (a *App) handleUpdate(ctx context.Context, u tgUpdate) {
	if u.Callback != nil {
		a.handleCallback(ctx, u.Callback)
		return
	}
	if u.Message == nil || u.Message.Chat.ID == 0 {
		return
	}
	msg := u.Message
	name := chatName(msg.Chat, msg.From)
	a.remember(msg.Chat.ID, name)
	s, _ := a.snapshot()
	text := strings.TrimSpace(msg.Text)
	cmd, _, _ := strings.Cut(text, " ")
	cmd = strings.ToLower(cmd)
	cmd, _, _ = strings.Cut(cmd, "@")
	if cmd == "/start" || text == "" {
		_ = a.send(ctx, msg.Chat.ID, s.Welcome, markup(s))
		return
	}
	reply := s.Autoreply
	if reply == "" {
		reply = s.Welcome
	}
	_ = a.send(ctx, msg.Chat.ID, reply, markup(s))
}

func (a *App) handleCallback(ctx context.Context, q *tgCallback) {
	c, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_ = a.panel.tg(c, "answerCallbackQuery", map[string]any{"callback_query_id": q.ID}, nil)
	if q.Message == nil {
		return
	}
	s, _ := a.snapshot()
	var n int
	if _, err := fmt.Sscanf(q.Data, "b%d", &n); err != nil || n < 0 || n >= len(s.Buttons) {
		return
	}
	b := s.Buttons[n]
	if b.Reply == "" {
		return
	}
	_ = a.send(ctx, q.Message.Chat.ID, b.Reply, nil)
}

func markup(s Settings) any {
	if len(s.Buttons) == 0 {
		return nil
	}
	rows := make([]any, 0, len(s.Buttons))
	for i, b := range s.Buttons {
		btn := map[string]any{"text": b.Text}
		if b.URL != "" {
			btn["url"] = b.URL
		} else {
			btn["callback_data"] = fmt.Sprintf("b%d", i)
		}
		rows = append(rows, []any{btn})
	}
	return map[string]any{"inline_keyboard": rows}
}

func (a *App) send(ctx context.Context, chat int64, text string, kb any) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	payload := map[string]any{"chat_id": chat, "text": text}
	if kb != nil {
		payload["reply_markup"] = kb
	}
	c, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return a.panel.tg(c, "sendMessage", payload, nil)
}

func chatName(c tgChat, from *tgUser) string {
	if c.Title != "" {
		return clip(c.Title, maxName)
	}
	name := strings.TrimSpace(c.FirstName + " " + c.LastName)
	user := c.Username
	if name == "" && from != nil {
		name = strings.TrimSpace(from.FirstName + " " + from.LastName)
		if user == "" {
			user = from.Username
		}
	}
	if user != "" {
		if name != "" {
			name += " @" + user
		} else {
			name = "@" + user
		}
	}
	if name == "" {
		name = fmt.Sprintf("%d", c.ID)
	}
	return clip(name, maxName)
}

func (a *App) broadcast(text string, chats []Chat) {
	defer func() {
		a.mu.Lock()
		a.bc.Running = false
		a.mu.Unlock()
	}()
	for _, c := range chats {
		if a.ctx.Err() != nil {
			return
		}
		err := a.send(a.ctx, c.ID, text, nil)
		a.mu.Lock()
		if err != nil {
			a.bc.Failed++
			a.bc.LastErr = err.Error()
		} else {
			a.bc.Sent++
		}
		a.mu.Unlock()
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(120 * time.Millisecond):
		}
	}
}

func validBroadcast(text string) error {
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > maxWelcome {
		return fmt.Errorf("текст рассылки пустой или длиннее %d символов", maxWelcome)
	}
	return nil
}
