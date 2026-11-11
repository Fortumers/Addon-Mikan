package bot

import (
    "fmt"
    "log"
    "os"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/user/mikan-subs-bot-addon/config"
    "github.com/user/mikan-subs-bot-addon/metrics"
    "github.com/user/mikan-subs-bot-addon/storage"
)

// isQuietHours returns true when the current time in Moscow is between 22:00 and 09:00.
func isQuietHours() bool {
    loc, err := time.LoadLocation("Europe/Moscow")
    if err != nil {
        loc = time.UTC
    }
    now := time.Now().In(loc)
    hour := now.Hour()
    return hour >= 22 || hour < 9
}

// sendMessage sends a Telegram message respecting quiet hours.
func sendMessage(api *tgbotapi.BotAPI, msg tgbotapi.Chattable) error {
    // If quiet hours – disable notification for supported message types.
    if isQuietHours() {
        switch v := msg.(type) {
        case tgbotapi.MessageConfig:
            v.DisableNotification = true
            _, err := api.Send(v)
            return err
        case tgbotapi.PhotoConfig:
            v.DisableNotification = true
            _, err := api.Send(v)
            return err
        default:
            // other types are sent as‑is.
        }
    }
    _, err := api.Send(msg)
    return err
}

type Bot struct {
    api   *tgbotapi.BotAPI
    cfg   *config.BotConfig
    store *storage.Storage
    metr  *metrics.Counters
    // channels for broadcast and auto‑messages (simplified)
    broadcastCh chan struct{}
}


// NewBot creates and starts the Telegram bot.
func NewBot(stg storage.Storage, cfg *config.BotConfig) (*Bot, error) {
    token := os.Getenv("MIKAN_PANEL_BOT_TOKEN")
    if token == "" {
        return nil, fmt.Errorf("environment variable MIKAN_PANEL_BOT_TOKEN not set")
    }
    api, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, err
    }
    b := &Bot{
        api:          api,
        cfg:          cfg,
        store:        stg,
        metr:         metrics.NewCounters(stg),
        broadcastCh:  make(chan struct{}, 1),
    }
    go b.listenUpdates()
    return b, nil
}


func (b *Bot) listenUpdates() {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 30
    updates := b.api.GetUpdatesChan(u)
    for update := range updates {
        if update.Message != nil && update.Message.IsCommand() {
            b.handleCommand(update.Message)
        } else if update.CallbackQuery != nil {
            b.handleCallback(update.CallbackQuery)
        }
    }
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
    switch msg.Command() {
    case "start":
        b.handleStart(msg)
    case "update":
        b.handleUpdate(msg)
    default:
        // ignore unknown commands
    }
}

func (b *Bot) handleStart(msg *tgbotapi.Message) {
    // ensure user is tracked in storage
    _ = b.store.AddUser(msg.From.ID)
    // Build menu based on configuration
    reply := b.buildMessageForUser(msg.From.ID)
    // Send respecting quiet hours
    if err := sendMessage(b.api, reply); err != nil {
        log.Printf("send start reply error: %v", err)
    }
    b.metr.IncBotMsg()
}

func (b *Bot) handleUpdate(msg *tgbotapi.Message) {
    // placeholder – could be used for force refresh
    b.handleStart(msg)
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
    data := cb.Data
    answer := tgbotapi.NewCallback(cb.ID, "Clicked: "+data)
    if _, err := b.api.Request(answer); err != nil {
        log.Printf("callback answer error: %v", err)
    }
}

// buildMessageForUser creates a Telegram message (or photo) with article‑style keyboard.
func (b *Bot) buildMessageForUser(userID int64) tgbotapi.Chattable {
    if len(b.cfg.Menus) == 0 {
        return tgbotapi.NewMessage(userID, "Меню пока не настроено.")
    }
    sec := b.cfg.Menus[0]
    // If image is set – return PhotoConfig with caption
    if sec.ImageURL != "" {
        photo := tgbotapi.NewPhotoShare(userID, sec.ImageURL)
        photo.Caption = sec.Text
        photo.ReplyMarkup = BuildArticleKeyboard(sec.ArticleButtons)
        return photo
    }
    // otherwise plain text with inline keyboard
    msg := tgbotapi.NewMessage(userID, sec.Text)
    msg.ReplyMarkup = BuildArticleKeyboard(sec.ArticleButtons)
    return msg
}

// RegisterHandlers is a thin wrapper used from addon.Register to expose bot methods.
func RegisterHandlers(b *Bot) {
    // Handlers already attached via listenUpdates.
    _ = b
}
