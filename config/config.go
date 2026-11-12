package config

import (
    "encoding/json"
    "os"
    "github.com/mikan-panel/sdk"
)

type ArticleButton struct {
    ID           string `json:"id"`
    Label        string `json:"label"`
    Emoji        string `json:"emoji,omitempty"`
    CallbackData string `json:"callback_data"`
    Disabled     bool   `json:"disabled,omitempty"`
    Selected     bool   `json:"selected,omitempty"`
    ColorHex     string `json:"color_hex,omitempty"`
}

type MenuSection struct {
    ID            string          `json:"id"`
    Text          string          `json:"text"`
    ImageURL      string          `json:"image_url,omitempty"`
    ArticleButtons []ArticleButton `json:"article_buttons,omitempty"`
}

type BotConfig struct {
    Menus []MenuSection `json:"menus"`
    // остальные настройки (авто‑сообщения, рассылка и пр.) могут быть добавлены позже
}

// LoadConfig reads the JSON configuration from the panel storage.
func LoadConfig(stg sdk.Storage) (*BotConfig, error) {
    data, err := stg.Get("config.json")
    if err != nil {
        // если конфиг отсутствует – возвращаем пустой объект
        if os.IsNotExist(err) {
            return &BotConfig{}, nil
        }
        return nil, err
    }
    var cfg BotConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

// SaveConfig writes the configuration back to storage.
func SaveConfig(stg sdk.Storage, cfg *BotConfig) error {
    b, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return stg.Set("config.json", b)
}
