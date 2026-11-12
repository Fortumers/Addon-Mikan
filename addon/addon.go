package addon

import (
	"github.com/mikan-panel/sdk"
	"github.com/Fortumers/Addon-Mikan/bot"
	"github.com/Fortumers/Addon-Mikan/config"
)

// Register registers the addon with the Mikan panel.
func Register() error {
	// Define required permissions for the addon.
	perms := []sdk.Permission{
		sdk.PermissionBot,            // "Отдать бота"
		sdk.PermissionUsersRead,
		sdk.PermissionUsersWrite,
		sdk.PermissionAllUsers,
		sdk.PermissionPayments,
		sdk.PermissionEvents,
		sdk.PermissionStorage,
		sdk.PermissionPages,
	}

	// UI pages that will appear in the admin panel.
	pages := []sdk.PageSpec{{
		ID:    "menu",
		Title: "Меню",
		Icon:  "📋",
	}, {
		ID:    "articles",
		Title: "Статьи",
		Icon:  "📰",
	}, {
		ID:    "commands",
		Title: "Команды",
		Icon:  "⚙️",
	}, {
		ID:    "automsg",
		Title: "Автосообщения",
		Icon:  "🤖",
	}, {
		ID:    "broadcast",
		Title: "Рассылка",
		Icon:  "📣",
	}, {
		ID:    "texts",
		Title: "Тексты и экраны",
		Icon:  "✏️",
	}, {
		ID:    "icons",
		Title: "Иконки и цвета",
		Icon:  "🎨",
	}, {
		ID:    "article_buttons",
		Title: "Кнопки статей",
		Icon:  "🔘",
	}}

	// Callback for initializing the addon after registration.
	initFn := func() error {
		// Load configuration from the panel storage.
		stg, err := sdk.GetStorage()
		if err != nil {
			return err
		}
		cfg, err := config.LoadConfig(stg)
		if err != nil {
			return err
		}
		// Create and start the Telegram bot.
		b, err := bot.NewBot(stg, cfg)
		if err != nil {
			return err
		}
		// Register bot handlers with the SDK event bus.
		bot.RegisterHandlers(b)
		return nil
	}

	spec := sdk.AddonSpec{
		ID:          "mikan-subs-bot",
		Name:        "Telegram‑бот подписчиков",
		Version:     "0.1.0",
		Permissions: perms,
		Pages:       pages,
		Init:        initFn,
	}
	return sdk.RegisterAddon(spec)
}
