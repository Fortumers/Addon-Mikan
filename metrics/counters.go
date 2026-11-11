package metrics

import (
    "encoding/json"
    "sync"
    "github.com/mikan-panel/sdk"
)
    "encoding/json"
    "sync"
    "github.com/mikan-panel/sdk"
)
    "sync"
    "github.com/mikan-panel/sdk"
)

type Counters struct {
    mu               sync.Mutex
    BotMsgSent       int64 `json:"bot_msg_sent"`
    AutoMsgSent      map[string]int64 `json:"auto_msg_sent"`
    BroadcastSent    int64 `json:"broadcast_sent"`
    ButtonClicks     map[string]int64 `json:"button_clicks"`
    storage          sdk.Storage
}

func NewCounters(stg sdk.Storage) *Counters {
    // Try to load existing counters, ignore errors (start fresh).
    var c Counters
    data, err := stg.Get("counters.json")
    if err == nil {
        _ = json.Unmarshal(data, &c)
    }
    c.storage = stg
    if c.AutoMsgSent == nil { c.AutoMsgSent = make(map[string]int64) }
    if c.ButtonClicks == nil { c.ButtonClicks = make(map[string]int64) }
    return &c
}

func (c *Counters) save() {
    c.mu.Lock()
    defer c.mu.Unlock()
    b, _ := json.MarshalIndent(c, "", "  ")
    _ = c.storage.Set("counters.json", b)
}

func (c *Counters) IncBotMsg() {
    c.mu.Lock()
    c.BotMsgSent++
    c.mu.Unlock()
    c.save()
}

func (c *Counters) IncAutoMsg(id string) {
    c.mu.Lock()
    c.AutoMsgSent[id]++
    c.mu.Unlock()
    c.save()
}

func (c *Counters) IncBroadcast() {
    c.mu.Lock()
    c.BroadcastSent++
    c.mu.Unlock()
    c.save()
}

func (c *Counters) IncButton(id string) {
    c.mu.Lock()
    c.ButtonClicks[id]++
    c.mu.Unlock()
    c.save()
}
