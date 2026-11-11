package bot

import (
    "testing"
    "github.com/user/mikan-subs-bot-addon/config"
)

func TestBuildArticleKeyboard(t *testing.T) {
    btns := []config.ArticleButton{
        {ID: "1", Label: "First", Emoji: "🔔", CallbackData: "cb1", Selected: true},
        {ID: "2", Label: "Second", CallbackData: "cb2", Disabled: true},
        {ID: "3", Label: "Third", CallbackData: "cb3"},
    }
    kb := BuildArticleKeyboard(btns)
    // Expect 2 rows (2 cols per row)
    if len(kb.InlineKeyboard) != 2 {
        t.Fatalf("expected 2 rows, got %d", len(kb.InlineKeyboard))
    }
    // First button should have ✅ prefix
    if kb.InlineKeyboard[0][0].Text != "✅ 🔔 First" {
        t.Fatalf("first button label mismatch: %s", kb.InlineKeyboard[0][0].Text)
    }
    // Second button should have 🚫 prefix
    if kb.InlineKeyboard[0][1].Text != "🚫 Second" {
        t.Fatalf("second button label mismatch: %s", kb.InlineKeyboard[0][1].Text)
    }
    // Third button should be in second row, plain label
    if kb.InlineKeyboard[1][0].Text != "Third" {
        t.Fatalf("third button label mismatch: %s", kb.InlineKeyboard[1][0].Text)
    }
}
