package app

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed ui/index.html ui/app.css ui/app.js
var uiFS embed.FS

type broadcastState struct {
	Running bool   `json:"running"`
	Sent    int    `json:"sent"`
	Failed  int    `json:"failed"`
	Total   int    `json:"total"`
	LastErr string `json:"last_error,omitempty"`
}

// App is the protocol-2 HTTP addon.
type App struct {
	id      string
	version string
	token   string
	listen  string
	panel   Panel
	ctx     context.Context

	mu       sync.Mutex
	settings Settings
	chats    []Chat
	dirty    bool
	storeErr string
	botUser  string
	botErr   string
	bc       broadcastState
}

type rpcCall struct {
	Method string          `json:"method"`
	Path   string          `json:"path"`
	Body   json.RawMessage `json:"body"`
	Admin  struct {
		Lang string `json:"lang"`
	} `json:"admin"`
}

// Run loads saved settings, serves /v2 and /ui, and polls the panel's Bot API proxy.
func Run(ctx context.Context) error {
	version := Version
	if version == "" {
		version = "dev"
	}
	listen := os.Getenv("MIKAN_ADAPTER_LISTEN")
	if listen == "" {
		listen = ":8080"
	}
	a := &App{
		id:       AddonID,
		version:  version,
		token:    os.Getenv("MIKAN_ADAPTER_TOKEN"),
		listen:   listen,
		ctx:      ctx,
		settings: DefaultSettings(),
		panel: Panel{
			BaseURL:  os.Getenv("MIKAN_PANEL_URL"),
			Token:    os.Getenv("MIKAN_PANEL_TOKEN"),
			BotURL:   os.Getenv("MIKAN_BOT_API_URL"),
			BotToken: os.Getenv("MIKAN_BOT_TOKEN"),
			HTTP:     &http.Client{Timeout: 45 * time.Second},
		},
	}
	a.load(ctx)
	go a.runBot(ctx)
	go a.flushLoop(ctx)

	srv := &http.Server{
		Addr:              a.listen,
		Handler:           a.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		log.Printf("listen %s id=%s version=%s", a.listen, a.id, a.version)
		errc <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shut)
		a.flush(context.Background())
		return nil
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/info", a.handleInfo)
	mux.HandleFunc("GET /v2/health", a.handleHealth)
	mux.HandleFunc("POST /v2/ui/rpc", a.handleRPC)
	mux.HandleFunc("GET /ui/", a.handleUI)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.token != "" && r.Header.Get("Authorization") != "Bearer "+a.token {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "unauthorized"})
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *App) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       a.id,
		"protocol": 2,
		"version":  a.version,
		"kind":     "bot",
		"mode":     "http",
		"name": map[string]string{
			"ru": "Бот подписчиков (Fortumers)",
			"en": "Subscribers bot (Fortumers)",
		},
		"description": map[string]string{
			"ru": "Меню, автоответ и рассылка. Настройки — на странице аддона.",
			"en": "Menu, auto-reply and broadcasts. Settings live on the addon page.",
		},
		"ui": map[string]any{
			"entry": "/ui/index.html",
			"title": map[string]string{"ru": "Бот подписчиков", "en": "Subscribers bot"},
		},
	})
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleRPC(w http.ResponseWriter, r *http.Request) {
	var call rpcCall
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&call); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad json"})
		return
	}
	switch call.Method + " " + call.Path {
	case "GET /status":
		writeJSON(w, http.StatusOK, a.status(call.Admin.Lang))
	case "PUT /settings":
		var s Settings
		if len(call.Body) == 0 || string(call.Body) == "null" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "пустые настройки"})
			return
		}
		if err := json.Unmarshal(call.Body, &s); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "настройки не читаются"})
			return
		}
		if err := s.Normalize(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := a.saveSettings(r.Context(), s); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "не удалось сохранить настройки"})
			return
		}
		writeJSON(w, http.StatusOK, a.status(call.Admin.Lang))
	case "POST /broadcast":
		var body struct {
			Text string `json:"text"`
		}
		if len(call.Body) > 0 && string(call.Body) != "null" {
			if err := json.Unmarshal(call.Body, &body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "текст не читается"})
				return
			}
		}
		body.Text = strings.TrimSpace(body.Text)
		if err := validBroadcast(body.Text); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		n, err := a.startBroadcast(body.Text)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		st := a.status(call.Admin.Lang)
		st["queued"] = n
		writeJSON(w, http.StatusOK, st)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown call"})
	}
}

func (a *App) handleUI(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/ui/")
	switch name {
	case "index.html", "app.js", "app.css":
	default:
		http.NotFound(w, r)
		return
	}
	b, err := uiFS.ReadFile("ui/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch name {
	case "index.html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case "app.js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (a *App) status(lang string) map[string]any {
	if lang != "en" {
		lang = "ru"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.settings
	s.Buttons = append([]Button(nil), a.settings.Buttons...)
	return map[string]any{
		"lang":     lang,
		"version":  a.version,
		"settings": s,
		"chats":    len(a.chats),
		"bot": map[string]any{
			"ok":       a.botErr == "" && a.botUser != "",
			"username": a.botUser,
			"error":    a.botErr,
		},
		"store_error": a.storeErr,
		"broadcast":   a.bc,
	}
}

func (a *App) snapshot() (Settings, []Chat) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.settings
	s.Buttons = append([]Button(nil), a.settings.Buttons...)
	c := append([]Chat(nil), a.chats...)
	return s, c
}

func (a *App) setBot(username string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.botUser = username
	if err != nil {
		a.botErr = err.Error()
		return
	}
	a.botErr = ""
}

func (a *App) touchBot() {
	a.mu.Lock()
	a.botErr = ""
	a.mu.Unlock()
}

func (a *App) remember(id int64, name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now().Unix()
	for i := range a.chats {
		if a.chats[i].ID == id {
			if a.chats[i].Name != name {
				a.chats[i].Name = name
				a.dirty = true
			}
			a.chats[i].At = now
			return
		}
	}
	a.chats = append(a.chats, Chat{ID: id, Name: name, At: now})
	if len(a.chats) > maxChats {
		old := 0
		for i := range a.chats {
			if a.chats[i].At < a.chats[old].At {
				old = i
			}
		}
		a.chats = append(a.chats[:old], a.chats[old+1:]...)
	}
	a.dirty = true
}

func (a *App) saveSettings(ctx context.Context, s Settings) error {
	if a.panel.enabled() {
		b, err := json.Marshal(s)
		if err != nil {
			return err
		}
		if err := a.panel.kvPut(ctx, "settings", b); err != nil {
			return err
		}
	}
	a.mu.Lock()
	a.settings = s
	a.storeErr = ""
	a.mu.Unlock()
	return nil
}

func (a *App) load(ctx context.Context) {
	if !a.panel.enabled() {
		return
	}
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if b, err := a.panel.kvGet(c, "settings"); err == nil {
		var s Settings
		if json.Unmarshal(b, &s) == nil && s.Normalize() == nil {
			a.settings = s
		}
	} else if !errors.Is(err, errNotFound) {
		log.Printf("load settings: %v", err)
	}
	if b, err := a.panel.kvGet(c, "chats"); err == nil {
		var chats []Chat
		if json.Unmarshal(b, &chats) == nil {
			a.chats = chats
		}
	} else if !errors.Is(err, errNotFound) {
		log.Printf("load chats: %v", err)
	}
}

func (a *App) flushLoop(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			a.flush(context.Background())
			return
		case <-t.C:
			a.flush(ctx)
		}
	}
}

func (a *App) flush(ctx context.Context) {
	if !a.panel.enabled() {
		return
	}
	a.mu.Lock()
	if !a.dirty {
		a.mu.Unlock()
		return
	}
	chats := append([]Chat(nil), a.chats...)
	a.dirty = false
	a.mu.Unlock()
	b, err := json.Marshal(chats)
	if err != nil {
		return
	}
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := a.panel.kvPut(c, "chats", b); err != nil {
		a.mu.Lock()
		a.dirty = true
		a.storeErr = err.Error()
		a.mu.Unlock()
		log.Printf("save chats: %v", err)
	}
}

func (a *App) startBroadcast(text string) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.bc.Running {
		return 0, errors.New("рассылка уже идёт")
	}
	if len(a.chats) == 0 {
		return 0, errors.New("пока никто не нажал /start — рассылать некому")
	}
	chats := append([]Chat(nil), a.chats...)
	a.bc = broadcastState{Running: true, Total: len(chats)}
	go a.broadcast(text, chats)
	return len(chats), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, `{"error":"encode"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}
