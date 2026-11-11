package bot

import (
    "strings"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/user/mikan-subs-bot-addon/config"
)

// BuildArticleKeyboard собирает клавиатуру из []config.ArticleButton.
// Кнопки группируются по 2 в строке, поддерживаются эмодзи, состояние Disabled/Selected и цвет (цвет передаётся в callback‑data, UI‑панель может использовать его).
func BuildArticleKeyboard(btns []config.ArticleButton) tgbotapi.InlineKeyboardMarkup {
    const cols = 2
    rows := [][]tgbotapi.InlineKeyboardButton{}
    curRow := []tgbotapi.InlineKeyboardButton{}
    for i, b := range btns {
        // Пропускаем полностью скрытые кнопки (Disabled && пустой Label)
        if b.Disabled && strings.TrimSpace(b.Label) == "" {
            continue
        }
        label := b.Label
        if b.Emoji != "" {
            label = b.Emoji + " " + label
        }
        // Маркеры выбранности/отключённости
        if b.Selected {
            label = "✅ " + label
        } else if b.Disabled {
            label = "🚫 " + label
        }
        // Добавляем цвет в callback‑data, если задан
        cb := b.CallbackData
        if b.ColorHex != "" {
            // Формат: <data>|color=<hex>
            cb = cb + "|color=" + b.ColorHex
        }
        btn := tgbotapi.NewInlineKeyboardButtonData(label, cb)
        curRow = append(curRow, btn)
        if (i+1)%cols == 0 {
            rows = append(rows, curRow)
            curRow = []tgbotapi.InlineKeyboardButton{}
        }
    }
    if len(curRow) > 0 {
        rows = append(rows, curRow)
    }
    return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
