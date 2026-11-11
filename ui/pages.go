package ui

import (
    "encoding/json"
    "fmt"
    "github.com/mikan-panel/sdk"
    "github.com/user/mikan-subs-bot-addon/config"
    "github.com/user/mikan-subs-bot-addon/storage"
)

// RegisterPages registers UI pages for the addon.
func RegisterPages() error {
    return sdk.RegisterPage(sdk.PageSpec{
        ID:    "article_buttons",
        Title: "Кнопки статей",
        Icon:  "🔘",
        Render: func(ctx sdk.PageContext) (sdk.HTML, error) {
            // Load configuration
            stg, err := sdk.GetStorage()
            if err != nil {
                return "", err
            }
            cfg, err := config.LoadConfig(stg)
            if err != nil {
                return "", err
            }
            // For simplicity we assume there is at least one menu section.
            var btns []config.ArticleButton
            if len(cfg.Menus) > 0 {
                btns = cfg.Menus[0].ArticleButtons
            }
            data, _ := json.Marshal(btns)
            // Render a simple placeholder UI – actual UI is implemented on the client side.
            html := fmt.Sprintf(`
                <div id="article-buttons-app" data-buttons='%s'></div>
                <script src="/static/article_buttons.js"></script>
            `, data)
            return sdk.HTML(html), nil
        },
    })
}
