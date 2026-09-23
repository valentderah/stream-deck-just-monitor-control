(function () {
  "use strict";

  var websocket = null;
  var pluginUUID = null;
  var actionUUID = null;
  var settings = {};
  var translations = {};
  var pickerMode = "single";
  var pickerKey = "monitorId";

  function getDefaults() {
    var defaults = {};
    document.querySelectorAll("[data-setting][data-default]").forEach(function (el) {
      var key = el.getAttribute("data-setting");
      var value = el.getAttribute("data-default");
      if (el.type === "checkbox") {
        defaults[key] = value === "true";
      } else if (el.type === "number" || el.classList.contains("port-select")) {
        defaults[key] = parseInt(value, 10) || 0;
      } else {
        defaults[key] = value;
      }
    });
    return defaults;
  }

  function updateConditionalVisibility() {
    document.querySelectorAll("[data-visible-when]").forEach(function (row) {
      var spec = row.getAttribute("data-visible-when");
      var eq = spec.indexOf("=");
      if (eq === -1) return;
      var key = spec.slice(0, eq);
      var values = spec.slice(eq + 1).split(",");
      var current = settings[key];
      row.style.display = values.indexOf(String(current)) === -1 ? "none" : "flex";
    });
  }

  function loadLocalization(lang) {
    return fetch("../" + lang + ".json")
      .then(function (res) {
        if (!res.ok) throw new Error(res.status);
        return res.json();
      })
      .then(function (data) {
        if (data.Localization) return data.Localization;
        throw new Error("No Localization key");
      })
      .catch(function () {
        if (lang === "en") return {};
        return fetch("../en.json")
          .then(function (res) { return res.json(); })
          .then(function (data) { return data.Localization || {}; })
          .catch(function () { return {}; });
      });
  }

  function applyLocalization(tr) {
    document.querySelectorAll("[data-i18n]").forEach(function (el) {
      var key = el.getAttribute("data-i18n");
      if (tr[key]) el.textContent = tr[key];
    });
  }

  function applySettings() {
    document.querySelectorAll("[data-setting]").forEach(function (el) {
      var key = el.getAttribute("data-setting");
      if (settings[key] === undefined) return;
      if (el.type === "checkbox") {
        el.checked = settings[key] === true || settings[key] === "true";
      } else {
        el.value = settings[key];
      }
    });
    updateConditionalVisibility();
    syncPickerFromSettings();
  }

  function bindSettingListeners() {
    document.querySelectorAll("[data-setting]").forEach(function (el) {
      el.addEventListener("change", function () {
        var key = el.getAttribute("data-setting");
        if (el.type === "checkbox") {
          settings[key] = el.checked;
        } else if (el.type === "number" || el.classList.contains("port-select")) {
          settings[key] = parseInt(el.value, 10) || 0;
        } else {
          settings[key] = el.value;
        }
        updateConditionalVisibility();
        sendSettings();
        if (key === "monitorId") requestRefreshRates();
      });
    });
  }

  function sendToPlugin(payload) {
    if (!websocket || websocket.readyState !== WebSocket.OPEN) return;
    websocket.send(JSON.stringify({
      event: "sendToPlugin",
      action: actionUUID,
      context: pluginUUID,
      payload: payload
    }));
  }

  function sendSettings() {
    if (!websocket || websocket.readyState !== WebSocket.OPEN) return;
    websocket.send(JSON.stringify({
      event: "setSettings",
      context: pluginUUID,
      payload: settings
    }));
  }

  function setPickerLoading() {
    var host = document.getElementById("monitorPicker");
    if (!host) return;
    host.classList.add("muted");
    host.innerHTML = "";
    var span = document.createElement("span");
    span.setAttribute("data-i18n", "Loading");
    span.textContent = (translations.Loading) || "Loading…";
    host.appendChild(span);
  }

  function formatMonitorLabel(m) {
    var primaryLabel = translations.Primary || "Primary";
    var num = m.displayNum || 1;
    var name = m.name || "Display";
    if (m.isPrimary) {
      return name + " (" + primaryLabel + ", " + num + ")";
    }
    return name + " (" + num + ")";
  }

  window.identifyMonitors = function () {
    sendToPlugin({ type: "identify" });
  };

  function updateMultiSelectTrigger(box, monitors) {
    var trigger = box.querySelector(".multiselect-trigger");
    if (!trigger) return;
    var ids = settings.monitorIds || [];
    if (ids.length === 0) {
      trigger.textContent = "—";
      return;
    }
    var labels = [];
    monitors.forEach(function (m) {
      if (ids.indexOf(m.id) !== -1) {
        labels.push(formatMonitorLabel(m));
      }
    });
    trigger.textContent = labels.length > 0 ? labels.join(", ") : "—";
    trigger.title = trigger.textContent;
  }

  function syncPickerFromSettings() {
    if (pickerMode === "multi") {
      var ids = settings.monitorIds || [];
      document.querySelectorAll("#monitorPicker input[type=checkbox]").forEach(function (cb) {
        cb.checked = ids.indexOf(cb.value) !== -1;
      });
    } else {
      var sel = document.querySelector("#monitorPicker select");
      if (sel && settings.monitorId) sel.value = settings.monitorId;
    }
  }

  function fillMonitors(monitors) {
    var host = document.getElementById("monitorPicker");
    if (!host) return;
    host.classList.remove("muted");
    host.innerHTML = "";
    if (!monitors || !monitors.length) {
      host.textContent = "—";
      return;
    }

    if (pickerMode === "multi") {
      if (!settings.monitorIds || settings.monitorIds.length === 0) {
        settings.monitorIds = monitors.map(function (m) { return m.id; });
        sendSettings();
      }

      var box = document.createElement("div");
      box.className = "multiselect-box";

      var trigger = document.createElement("div");
      trigger.className = "multiselect-trigger";
      box.appendChild(trigger);

      var dropdown = document.createElement("div");
      dropdown.className = "multiselect-dropdown";

      monitors.forEach(function (m) {
        var opt = document.createElement("label");
        opt.className = "multiselect-option";

        var cb = document.createElement("input");
        cb.type = "checkbox";
        cb.value = m.id;
        cb.checked = (settings.monitorIds || []).indexOf(m.id) !== -1;

        cb.addEventListener("change", function () {
          var ids = [];
          dropdown.querySelectorAll("input:checked").forEach(function (c) { ids.push(c.value); });
          settings.monitorIds = ids;
          updateMultiSelectTrigger(box, monitors);
          sendSettings();
        });

        var span = document.createElement("span");
        span.textContent = formatMonitorLabel(m);

        opt.appendChild(cb);
        opt.appendChild(span);
        dropdown.appendChild(opt);
      });

      box.appendChild(dropdown);

      trigger.addEventListener("click", function (e) {
        e.stopPropagation();
        var wasOpen = box.classList.contains("open");
        document.querySelectorAll(".multiselect-box.open").forEach(function (b) { b.classList.remove("open"); });
        if (!wasOpen) box.classList.add("open");
      });

      dropdown.addEventListener("click", function (e) {
        e.stopPropagation();
      });

      host.appendChild(box);
      updateMultiSelectTrigger(box, monitors);
    } else {
      var sel = document.createElement("select");
      monitors.forEach(function (m) {
        var opt = document.createElement("option");
        opt.value = m.id;
        opt.textContent = formatMonitorLabel(m);
        sel.appendChild(opt);
      });

      if (!settings.monitorId && monitors.length > 0) {
        settings.monitorId = monitors[0].id;
        sendSettings();
      }
      if (settings.monitorId) {
        sel.value = settings.monitorId;
      }

      sel.addEventListener("change", function () {
        settings.monitorId = sel.value;
        sendSettings();
        requestRefreshRates();
      });
      host.appendChild(sel);
      requestRefreshRates();
    }

    // Добавляем строчку с кнопкой Identify (без Windows Display Settings)
    var parentItem = host.closest(".sdpi-item");
    if (parentItem && !document.getElementById("monitorActionRow")) {
      var row = document.createElement("div");
      row.id = "monitorActionRow";
      row.className = "sdpi-item sdpi-item-hint";
      row.innerHTML =
        '<div class="sdpi-item-label"></div>' +
        '<div class="sdpi-item-value monitor-actions">' +
          '<a href="javascript:void(0)" class="link-btn" onclick="identifyMonitors()">' + (translations.Identify || "Identify") + '</a>' +
        '</div>';
      parentItem.parentNode.insertBefore(row, parentItem.nextSibling);
    }
  }

  // Закрытие выпадайки чекбоксов при клике в любое другое место
  document.addEventListener("click", function () {
    document.querySelectorAll(".multiselect-box.open").forEach(function (box) {
      box.classList.remove("open");
    });
  });

  function requestRefreshRates() {
    var rateSel = document.getElementById("refreshRates");
    if (!rateSel || !settings.monitorId) return;
    sendToPlugin({ type: "get_refresh_rates", monitorId: settings.monitorId });
  }

  function fillRefreshRates(rates) {
    var rateSel = document.getElementById("refreshRates");
    if (!rateSel) return;
    rateSel.innerHTML = "";
    (rates || []).forEach(function (r) {
      var opt = document.createElement("option");
      opt.value = r.numerator + "/" + (r.denominator || 1);
      opt.textContent = (Number(r.hz).toFixed(2)) + " Hz";
      opt.dataset.numerator = r.numerator;
      opt.dataset.denominator = r.denominator || 1;
      rateSel.appendChild(opt);
    });

    if (settings.numerator) {
      rateSel.value = settings.numerator + "/" + (settings.denominator || 1);
    } else if (rates && rates.length > 0) {
      settings.numerator = rates[0].numerator;
      settings.denominator = rates[0].denominator;
      sendSettings();
    }

    rateSel.onchange = function () {
      var opt = rateSel.options[rateSel.selectedIndex];
      if (!opt) return;
      settings.numerator = parseInt(opt.dataset.numerator, 10);
      settings.denominator = parseInt(opt.dataset.denominator, 10);
      sendSettings();
    };
  }

  window.initMonitorPicker = function (opts) {
    pickerMode = (opts && opts.mode) || "single";
    pickerKey = (opts && opts.settingKey) || (pickerMode === "multi" ? "monitorIds" : "monitorId");
  };

  window.connectElgatoStreamDeckSocket = function (port, uuid, event, info, actionInfo) {
    pluginUUID = uuid;
    var parsedInfo = JSON.parse(info);
    var parsedActionInfo = JSON.parse(actionInfo);
    actionUUID = parsedActionInfo.action;
    var lang = (parsedInfo.application && parsedInfo.application.language) || "en";
    settings = Object.assign({}, getDefaults(), parsedActionInfo.payload.settings || {});

    loadLocalization(lang).then(function (loaded) {
      translations = loaded;
      applyLocalization(translations);
      applySettings();
      document.body.style.visibility = "visible";
    });

    websocket = new WebSocket("ws://127.0.0.1:" + port);
    websocket.onopen = function () {
      websocket.send(JSON.stringify({ event: event, uuid: uuid }));
      setPickerLoading();
      sendToPlugin({ type: "get_monitors" });
    };
    websocket.onmessage = function (evt) {
      var data = JSON.parse(evt.data);
      if (data.event === "didReceiveSettings" && data.payload) {
        settings = Object.assign({}, getDefaults(), data.payload.settings || {});
        applySettings();
      }
      if (data.event === "sendToPropertyInspector" && data.payload) {
        if (data.payload.type === "monitors") fillMonitors(data.payload.monitors);
        if (data.payload.type === "refresh_rates") fillRefreshRates(data.payload.rates);
      }
    };
    bindSettingListeners();
  };
})();
