(function () {
  var waiting = new Map();
  var seq = 1;
  var lang = "ru";
  var poll;

  var copy = {
    ru: {
      title: "Бот подписчиков",
      lead: "Меню, автоответ и рассылка тем, кто написал боту.",
      chats: "Написали боту",
      user: "Имя бота",
      ver: "Версия",
      menu: "Сообщения и меню",
      welcome: "Приветствие на /start",
      auto: "Ответ на остальные сообщения",
      buttons: "Кнопки",
      add: "Добавить",
      hint: "Ссылка открывается в Telegram. Текст ответа бот присылает отдельным сообщением.",
      save: "Сохранить",
      preview: "Как видит человек",
      cast: "Рассылка",
      castHint: "Уходит всем, кто хотя бы раз написал этому боту.",
      send: "Отправить",
      ph: "Текст новости",
      textPh: "Подпись",
      valuePh: "https://… или ответ",
      link: "Ссылка",
      reply: "Ответ",
      remove: "Убрать",
      checking: "Проверка…",
      online: "Бот на связи",
      waiting: "Ждёт назначения бота",
      offline: "Бот не отвечает",
      saved: "Сохранено",
      sent: "Рассылка запущена",
      none: "Пока пусто",
      confirm: "Отправить это сообщение всем, кто писал боту?"
    },
    en: {
      title: "Subscribers bot",
      lead: "Menu, auto-reply and a broadcast to everyone who wrote to the bot.",
      chats: "Wrote to the bot",
      user: "Bot name",
      ver: "Version",
      menu: "Messages and menu",
      welcome: "Greeting on /start",
      auto: "Reply to other messages",
      buttons: "Buttons",
      add: "Add",
      hint: "A link opens in Telegram. A reply is sent back as a message.",
      save: "Save",
      preview: "What a person sees",
      cast: "Broadcast",
      castHint: "Goes to everyone who has written to this bot at least once.",
      send: "Send",
      ph: "News text",
      textPh: "Label",
      valuePh: "https://… or a reply",
      link: "Link",
      reply: "Reply",
      remove: "Remove",
      checking: "Checking…",
      online: "Bot is online",
      waiting: "Waiting to be assigned the bot",
      offline: "Bot is not answering",
      saved: "Saved",
      sent: "Broadcast started",
      none: "Nothing yet",
      confirm: "Send this message to everyone who wrote to the bot?"
    }
  };

  window.addEventListener("message", function (ev) {
    var m = ev.data;
    if (!m || m.type !== "rpc-result" || !waiting.has(m.id)) return;
    var job = waiting.get(m.id);
    waiting.delete(m.id);
    clearTimeout(job.timer);
    if (m.status >= 200 && m.status < 300) job.ok(m.body || {});
    else job.bad(m.body || { error: "status " + m.status });
  });

  function t(key) { return (copy[lang] || copy.ru)[key]; }

  function rpc(method, path, body) {
    return new Promise(function (ok, bad) {
      var id = "c" + seq++;
      var timer = setTimeout(function () {
        waiting.delete(id);
        bad({ error: "timeout" });
      }, 40000);
      waiting.set(id, { ok: ok, bad: bad, timer: timer });
      window.parent.postMessage({ type: "rpc", id: id, method: method, path: path, body: body }, "*");
    });
  }

  function toast(text) {
    var el = document.getElementById("toast");
    el.textContent = text;
    el.classList.add("show");
    clearTimeout(toast.timer);
    toast.timer = setTimeout(function () { el.classList.remove("show"); }, 2400);
  }

  function applyLang() {
    document.documentElement.lang = lang;
    document.getElementById("title").textContent = t("title");
    document.getElementById("lead").textContent = t("lead");
    document.getElementById("lbl-chats").textContent = t("chats");
    document.getElementById("lbl-user").textContent = t("user");
    document.getElementById("lbl-ver").textContent = t("ver");
    document.getElementById("lbl-menu").textContent = t("menu");
    document.getElementById("lbl-welcome").textContent = t("welcome");
    document.getElementById("lbl-auto").textContent = t("auto");
    document.getElementById("lbl-buttons").textContent = t("buttons");
    document.getElementById("add").textContent = t("add");
    document.getElementById("hint").textContent = t("hint");
    document.getElementById("save").textContent = t("save");
    document.getElementById("lbl-preview").textContent = t("preview");
    document.getElementById("lbl-cast").textContent = t("cast");
    document.getElementById("cast-hint").textContent = t("castHint");
    document.getElementById("cast-btn").textContent = t("send");
    document.getElementById("cast-text").placeholder = t("ph");
  }

  function row(button) {
    var wrap = document.createElement("div");
    wrap.className = "btn-row";
    var text = document.createElement("input");
    text.placeholder = t("textPh");
    text.maxLength = 40;
    text.value = button.text || "";
    var kind = document.createElement("select");
    var optLink = document.createElement("option");
    optLink.value = "url";
    optLink.textContent = t("link");
    var optReply = document.createElement("option");
    optReply.value = "reply";
    optReply.textContent = t("reply");
    kind.appendChild(optLink);
    kind.appendChild(optReply);
    kind.value = button.url ? "url" : "reply";
    var value = document.createElement("input");
    value.placeholder = t("valuePh");
    value.value = button.url || button.reply || "";
    var del = document.createElement("button");
    del.type = "button";
    del.className = "mini";
    del.textContent = t("remove");
    del.addEventListener("click", function () {
      wrap.remove();
      paintPreview();
    });
    [text, kind, value].forEach(function (el) {
      el.addEventListener("input", paintPreview);
      el.addEventListener("change", paintPreview);
    });
    wrap.appendChild(text);
    wrap.appendChild(kind);
    wrap.appendChild(value);
    wrap.appendChild(del);
    return wrap;
  }

  function readButtons() {
    return Array.prototype.map.call(document.querySelectorAll(".btn-row"), function (wrap) {
      var fields = wrap.querySelectorAll("input, select");
      var item = { text: fields[0].value.trim() };
      if (fields[1].value === "url") item.url = fields[2].value.trim();
      else item.reply = fields[2].value.trim();
      return item;
    });
  }

  function paintPreview() {
    document.getElementById("preview-text").textContent = document.getElementById("welcome").value || t("none");
    var box = document.getElementById("preview-keys");
    box.textContent = "";
    readButtons().forEach(function (b, i) {
      if (!b.text) return;
      var k = document.createElement("span");
      k.className = "key";
      k.style.animationDelay = (i * 0.15) + "s";
      k.textContent = b.text;
      box.appendChild(k);
    });
  }

  function paintStatus(state) {
    if (state.lang === "en" || state.lang === "ru") {
      if (state.lang !== lang) {
        lang = state.lang;
        applyLang();
      }
    }
    document.getElementById("chat-count").textContent = String(state.chats || 0);
    document.getElementById("ver").textContent = state.version || "—";
    var bot = state.bot || {};
    var dot = document.getElementById("dot");
    var label = document.getElementById("status-text");
    dot.className = "dot";
    if (bot.ok) {
      dot.classList.add("on");
      label.textContent = t("online");
      document.getElementById("bot-name").textContent = bot.username ? "@" + bot.username : "—";
    } else if (!bot.error) {
      label.textContent = t("waiting");
      document.getElementById("bot-name").textContent = "—";
    } else {
      dot.classList.add("off");
      label.textContent = t("offline");
      document.getElementById("bot-name").textContent = "—";
    }
    var bc = state.broadcast || {};
    var bar = document.getElementById("bar");
    if (bc.total) {
      bar.hidden = false;
      var done = (bc.sent || 0) + (bc.failed || 0);
      document.getElementById("bar-fill").style.width = Math.min(100, Math.round(done * 100 / bc.total)) + "%";
      document.getElementById("cast-note").textContent = done + " / " + bc.total + (bc.failed ? " · " + bc.failed : "");
    }
  }

  function fill(state) {
    paintStatus(state);
    var s = state.settings || { welcome: "", autoreply: "", buttons: [] };
    document.getElementById("welcome").value = s.welcome || "";
    document.getElementById("auto").value = s.autoreply || "";
    var box = document.getElementById("buttons");
    box.textContent = "";
    (s.buttons || []).forEach(function (b) { box.appendChild(row(b)); });
    paintPreview();
  }

  function refresh() {
    return rpc("GET", "/status").then(fill).catch(function (err) {
      document.getElementById("status-text").textContent = (err && err.error) || t("offline");
    });
  }

  document.getElementById("add").addEventListener("click", function () {
    document.getElementById("buttons").appendChild(row({ text: "", reply: "" }));
  });
  document.getElementById("welcome").addEventListener("input", paintPreview);
  document.getElementById("form").addEventListener("submit", function (ev) {
    ev.preventDefault();
    var note = document.getElementById("save-note");
    note.textContent = "…";
    rpc("PUT", "/settings", {
      welcome: document.getElementById("welcome").value,
      autoreply: document.getElementById("auto").value,
      buttons: readButtons()
    }).then(function (state) {
      fill(state);
      note.textContent = t("saved");
      toast(t("saved"));
    }).catch(function (err) {
      note.textContent = (err && err.error) || "error";
    });
  });
  document.getElementById("cast").addEventListener("submit", function (ev) {
    ev.preventDefault();
    var text = document.getElementById("cast-text").value.trim();
    if (!text) return;
    if (!window.confirm(t("confirm"))) return;
    var note = document.getElementById("cast-note");
    note.textContent = "…";
    rpc("POST", "/broadcast", { text: text }).then(function (state) {
      fill(state);
      note.textContent = t("sent");
      toast(t("sent"));
      document.getElementById("cast-text").value = "";
    }).catch(function (err) {
      note.textContent = (err && err.error) || "error";
    });
  });

  applyLang();
  refresh();
  poll = setInterval(function () {
    rpc("GET", "/status").then(paintStatus).catch(function () {});
  }, 4000);
  window.addEventListener("pagehide", function () { clearInterval(poll); });
})();
