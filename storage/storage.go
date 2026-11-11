package storage

import (
    "encoding/json"
    "fmt"
    "github.com/mikan-panel/sdk"
)

// Storage is a thin wrapper around the panel's key‑value store.
// It provides convenience methods used by the bot.

type Storage struct {
    kv sdk.Storage
}

func New(sdkStg sdk.Storage) *Storage {
    return &Storage{kv: sdkStg}
}

func (s *Storage) AddUser(userID int64) error {
    key := fmt.Sprintf("user:%d", userID)
    // Store a placeholder value just to remember that this user interacted.
    return s.kv.Set(key, []byte("1"))
}

func (s *Storage) GetUserList() ([]int64, error) {
    // In a real implementation we would iterate over keys with prefix "user:".
    // Here we assume the SDK provides a ListKeys method (mocked for now).
    keys, err := s.kv.ListKeys("user:")
    if err != nil {
        return nil, err
    }
    ids := make([]int64, 0, len(keys))
    for _, k := range keys {
        var id int64
        _, err := fmt.Sscanf(k, "user:%d", &id)
        if err == nil {
            ids = append(ids, id)
        }
    }
    return ids, nil
}

func (s *Storage) Set(key string, value interface{}) error {
    b, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return s.kv.Set(key, b)
}

func (s *Storage) ListKeys(prefix string) ([]string, error) {
    // SDK may provide a method to list keys with a prefix. If not, return empty slice.
    // This stub ensures compilation; actual implementation can be added later.
    return []string{}, nil
}

