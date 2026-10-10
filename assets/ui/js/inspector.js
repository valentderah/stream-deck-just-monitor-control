(function () {
  "use strict";

  var core = window.InspectorCore;
  var websocket = null;
  var pluginUUID = null;
  var actionUUID = null;
  var controller = "Keypad";
  var translations = {};
  var schema = null;
  var saved = {};
  var settings = {};
  var monitors = null;
  var rates = null;
  var controls = [];

  function t(key) {
    return translations[key] || key;
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

  function applyLocalization() {
    document.querySelectorAll("[data-i18n]").forEach(function (el) {
      var key = el.getAttribute("data-i18n");
      if (translations[key]) el.textContent = translations[key];
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

  function saveSettings() {
    if (!websocket || websocket.readyState !== WebSocket.OPEN) return;
    websocket.send(JSON.stringify({ event: "setSettings", context: pluginUUID, payload: settings }));
  }

  function commit(next) {
    if (next === settings) {
      refresh();
      return;
    }
    var monitorChanged = next.monitorId !== settings.monitorId;
    settings = next;
    saveSettings();
    if (monitorChanged) requestRefreshRates();
    refresh();
  }

  function fieldOfKind(kind) {
    return schema.fields.filter(function (f) { return f.kind === kind; })[0] || null;
  }

  function selectedMonitor() {
    if (!monitors) return null;
    return monitors.filter(function (m) { return m.id === settings.monitorId; })[0] || null;
  }

  function requestRefreshRates() {
    if (!schema || !fieldOfKind("refreshRate") || !settings.monitorId) return;
    rates = null;
    sendToPlugin({ type: "get_refresh_rates", monitorId: settings.monitorId });
  }

  function monitorText() {
    return { primary: t("Primary"), disconnected: t("Disconnected") };
  }

  function closeDropdowns() {
    document.querySelectorAll(".multiselect-box.open").forEach(function (box) {
      box.classList.remove("open");
    });
  }

  function makeRow(field) {
    var row = document.createElement("div");
    row.className = "sdpi-item";
    var label = document.createElement("label");
    label.className = "sdpi-item-label";
    label.textContent = t(field.label);
    row.appendChild(label);
    return row;
  }

  function showLoading(host) {
    host.className = "sdpi-item-value muted";
    host.textContent = t("Loading");
  }

  function fillSelect(select, options, value) {
    if (document.activeElement === select) return;
    select.innerHTML = "";
    options.forEach(function (o) {
      var opt = document.createElement("option");
      opt.value = String(o.value);
      opt.textContent = o.label;
      select.appendChild(opt);
    });
    select.value = value === undefined || value === null ? "" : String(value);
  }

  function selectControl(field, optionsFor, valueFor) {
    var row = makeRow(field);
    var select = document.createElement("select");
    select.className = "sdpi-item-value";
    select.addEventListener("change", function () {
      commit(core.applyChange(settings, field, select.value));
    });
    select.addEventListener("blur", function () { refresh(); });
    row.appendChild(select);
    return {
      row: row,
      refresh: function () { fillSelect(select, optionsFor(), valueFor()); }
    };
  }

  function staticSelectControl(field) {
    var options = field.options.map(function (o) {
      return { value: o.value, label: o.labelKey ? t(o.labelKey) : o.label };
    });
    return selectControl(field,
      function () { return options; },
      function () { return settings[field.key]; });
  }

  function portSelectControl(field) {
    return selectControl(field,
      function () { return core.portOptions(field, selectedMonitor(), settings[field.key]); },
      function () { return settings[field.key]; });
  }

  function refreshRateControl(field) {
    return selectControl(field,
      function () { return core.refreshRateOptions(rates || [], settings); },
      function () { return settings.numerator ? core.encodeRefreshRate(settings) : ""; });
  }

  function numberControl(field) {
    var row = makeRow(field);
    var input = document.createElement("input");
    input.className = "sdpi-item-value";
    input.type = "number";
    if (typeof field.min === "number") input.min = String(field.min);
    if (typeof field.max === "number") input.max = String(field.max);
    input.addEventListener("change", function () {
      commit(core.applyChange(settings, field, input.value));
      input.value = String(settings[field.key]);
    });
    row.appendChild(input);
    return {
      row: row,
      refresh: function () {
        if (document.activeElement === input) return;
        input.value = String(settings[field.key]);
      }
    };
  }

  function monitorControl(field) {
    var row = makeRow(field);
    var host = document.createElement("div");
    row.appendChild(host);
    var select = document.createElement("select");
    select.addEventListener("change", function () {
      commit(core.applyChange(settings, field, select.value));
    });
    select.addEventListener("blur", function () { refresh(); });
    return {
      row: row,
      refresh: function () {
        if (!monitors) {
          showLoading(host);
          return;
        }
        if (select.parentNode !== host) {
          host.className = "sdpi-item-value";
          host.textContent = "";
          host.appendChild(select);
        }
        fillSelect(select, core.monitorOptions(monitors, settings[field.key], monitorText()), settings[field.key]);
      }
    };
  }

  function monitorsControl(field) {
    var row = makeRow(field);
    var host = document.createElement("div");
    row.appendChild(host);
    var box = document.createElement("div");
    box.className = "multiselect-box";
    var trigger = document.createElement("div");
    trigger.className = "multiselect-trigger";
    var dropdown = document.createElement("div");
    dropdown.className = "multiselect-dropdown";
    box.appendChild(trigger);
    box.appendChild(dropdown);

    trigger.addEventListener("click", function (e) {
      e.stopPropagation();
      var wasOpen = box.classList.contains("open");
      closeDropdowns();
      if (!wasOpen) box.classList.add("open");
    });
    dropdown.addEventListener("click", function (e) { e.stopPropagation(); });

    function checkedIds() {
      var ids = [];
      dropdown.querySelectorAll("input:checked").forEach(function (cb) { ids.push(cb.value); });
      return ids;
    }

    function render(options) {
      var ids = settings[field.key] || [];
      dropdown.innerHTML = "";
      options.forEach(function (o) {
        var label = document.createElement("label");
        label.className = "multiselect-option";
        var cb = document.createElement("input");
        cb.type = "checkbox";
        cb.value = o.value;
        cb.checked = ids.indexOf(o.value) !== -1;
        cb.addEventListener("change", function () {
          commit(core.applyChange(settings, field, checkedIds()));
        });
        var span = document.createElement("span");
        span.textContent = o.label;
        label.appendChild(cb);
        label.appendChild(span);
        dropdown.appendChild(label);
      });
      var labels = options
        .filter(function (o) { return ids.indexOf(o.value) !== -1; })
        .map(function (o) { return o.label; });
      trigger.textContent = labels.length ? labels.join(", ") : "—";
      trigger.title = trigger.textContent;
    }

    return {
      row: row,
      refresh: function () {
        if (!monitors) {
          showLoading(host);
          return;
        }
        if (box.parentNode !== host) {
          host.className = "sdpi-item-value";
          host.textContent = "";
          host.appendChild(box);
        }
        render(core.monitorOptions(monitors, settings[field.key] || [], monitorText()));
      }
    };
  }

  function identifyRow() {
    var row = document.createElement("div");
    row.className = "sdpi-item sdpi-item-hint";
    var spacer = document.createElement("div");
    spacer.className = "sdpi-item-label";
    var value = document.createElement("div");
    value.className = "sdpi-item-value monitor-actions";
    var link = document.createElement("a");
    link.href = "javascript:void(0)";
    link.className = "link-btn";
    link.textContent = t("Identify");
    link.addEventListener("click", function () { sendToPlugin({ type: "identify" }); });
    value.appendChild(link);
    row.appendChild(spacer);
    row.appendChild(value);
    return row;
  }

  function controlFor(field) {
    switch (field.kind) {
      case "monitor":
        return monitorControl(field);
      case "monitors":
        return monitorsControl(field);
      case "select":
        return field.source ? portSelectControl(field) : staticSelectControl(field);
      case "number":
        return numberControl(field);
      case "refreshRate":
        return refreshRateControl(field);
      default:
        throw new Error("unknown field kind: " + field.kind);
    }
  }

  function build() {
    var host = document.getElementById("inspector");
    host.innerHTML = "";
    controls = schema.fields.map(function (field) {
      var control = controlFor(field);
      control.field = field;
      host.appendChild(control.row);
      if (field.kind === "monitor" || field.kind === "monitors") host.appendChild(identifyRow());
      return control;
    });
  }

  function refresh() {
    controls.forEach(function (control) {
      control.row.style.display = core.isVisible(control.field, settings) ? "flex" : "none";
      control.refresh();
    });
  }

  function applyMonitorDefaults() {
    if (!schema || !monitors) return;
    var next = settings;
    schema.fields.forEach(function (field) {
      next = core.autoSelectMonitors(field, next, monitors);
    });
    commit(next);
  }

  function onSchema(next) {
    schema = next;
    settings = core.mergeSettings(schema, saved);
    build();
    refresh();
    requestRefreshRates();
    applyMonitorDefaults();
  }

  function onMonitors(list) {
    monitors = list || [];
    applyMonitorDefaults();
  }

  function onRefreshRates(payload) {
    if (!schema || !core.acceptsRefreshRates(settings, payload)) return;
    rates = payload.rates || [];
    var field = fieldOfKind("refreshRate");
    if (field && !settings.numerator && rates.length) {
      commit(core.applyChange(settings, field, core.encodeRefreshRate(rates[0])));
      return;
    }
    refresh();
  }

  function onSettings(next) {
    saved = next || {};
    if (!schema) return;
    var previousMonitor = settings.monitorId;
    settings = core.mergeSettings(schema, saved);
    if (settings.monitorId !== previousMonitor) requestRefreshRates();
    refresh();
  }

  function handleMessage(data) {
    if (data.event === "didReceiveSettings" && data.payload) {
      onSettings(data.payload.settings);
      return;
    }
    if (data.event !== "sendToPropertyInspector" || !data.payload) return;
    switch (data.payload.type) {
      case "schema":
        onSchema(data.payload.schema);
        break;
      case "monitors":
        onMonitors(data.payload.monitors);
        break;
      case "refresh_rates":
        onRefreshRates(data.payload);
        break;
    }
  }

  document.addEventListener("click", closeDropdowns);

  window.connectElgatoStreamDeckSocket = function (port, uuid, event, info, actionInfo) {
    pluginUUID = uuid;
    var parsedInfo = JSON.parse(info);
    var parsedActionInfo = JSON.parse(actionInfo);
    actionUUID = parsedActionInfo.action;
    saved = parsedActionInfo.payload.settings || {};
    controller = parsedActionInfo.payload.controller || controller;
    var lang = (parsedInfo.application && parsedInfo.application.language) || "en";

    var ready = loadLocalization(lang).then(function (loaded) {
      translations = loaded;
      applyLocalization();
      document.body.style.visibility = "visible";
    });

    websocket = new WebSocket("ws://127.0.0.1:" + port);
    websocket.onopen = function () {
      websocket.send(JSON.stringify({ event: event, uuid: uuid }));
      // A dial is configured by a different schema than a key.
      sendToPlugin({ type: "get_inspector", controller: controller });
    };
    websocket.onmessage = function (evt) {
      var data = JSON.parse(evt.data);
      ready.then(function () { handleMessage(data); });
    };
  };
})();
