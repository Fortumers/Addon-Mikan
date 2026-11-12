package bot

import (
    "log"
    "os"
    "time"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/mikan-panel/sdk"
    "github.com/Fortumers/Addon-Mikan/config"
    "github.com/Fortumers/Addon-Mikan/metrics"
    "github.com/Fortumers/Addon-Mikan/storage"
)

type Bot struct {
    api   *tgbotapi.BotAPI
    cfg   *config.BotConfig
    store storage.Storage
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
    _, err := b.api.Send(reply)
    if err != nil {
        log.Printf("send start reply error: %v", err)
    }
    b.metr.IncBotMsg()
}

func (b *Bot) handleUpdate(msg *tgbotapi.Message) {
    // placeholder – could be used for force refresh
    b.handleStart(msg)
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
    // Simple echo of callback data – real logic will be expanded later.
    data := cb.Data
    // If data encodes a button id, we can toggle Selected state etc.
    // For now just answer with alert.
    answer := tgbotapi.NewCallback(cb.ID, "Clicked: "+data)
    if _, err := b.api.Request(answer); err != nil {
        log.Printf("callback answer error: %v", err)
    }
}

func (b *Bot) buildMessageForUser(userID int64) tgbotapi.MessageConfig {
    // Find the first menu section (could be extended to per‑user selection)
    if len(b.cfg.Menus) == 0 {
        // fallback text
        return tgbotapi.NewMessage(userID, "Меню пока не настроено.")
    }
    sec := b.cfg.Menus[0]
    // Build text
    msg := tgbotapi.NewMessage(userID, sec.Text)
    // Attach image if provided
    if sec.ImageURL != "" {
        // using PhotoShare for simplicity
        photo := tgbotapi.NewPhotoShare(userID, sec.ImageURL)
        photo.Caption = sec.Text
        // Build article‑style keyboard from section buttons
        kb := BuildArticleKeyboard(sec.ArticleButtons)
        photo.ReplyMarkup = kb
        return photo
    }
    // Otherwise attach inline keyboard to plain message
    kb := BuildArticleKeyboard(sec.ArticleButtons)
    msg.ReplyMarkup = kb
    return msg
}

// RegisterHandlers is a thin wrapper used from addon.Register to expose bot methods.
func RegisterHandlers(b *Bot) {
    // In this simplified version handlers are already attached via listenUpdates.
    // SDK could also expose events – left as placeholder.
    _ = b
}
