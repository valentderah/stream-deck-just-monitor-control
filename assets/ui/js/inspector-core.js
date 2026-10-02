(function (root) {
  "use strict";

  function zeroValue(field) {
    return field.kind === "select" && field.type === "string" ? "" : 0;
  }

  function isUnset(field, value) {
    return value === undefined || value === null || value === zeroValue(field);
  }

  function mergeSettings(schema, saved) {
    var out = {};
    schema.fields.forEach(function (field) {
      if (field.default !== undefined) out[field.key] = field.default;
    });
    Object.keys(saved || {}).forEach(function (key) {
      out[key] = saved[key];
    });
    schema.fields.forEach(function (field) {
      if (field.zeroIsUnset && isUnset(field, out[field.key])) out[field.key] = field.default;
      if (field.kind === "refreshRate" && out.denominator === 0) out.denominator = 1;
    });
    return out;
  }

  function parseInteger(raw) {
    var text = String(raw).trim();
    if (!/^-?\d+$/.test(text)) return { valid: false };
    return { valid: true, value: parseInt(text, 10) };
  }

  function clamp(value, min, max) {
    if (typeof min === "number" && value < min) return min;
    if (typeof max === "number" && value > max) return max;
    return value;
  }

  function coerceValue(field, raw) {
    switch (field.kind) {
      case "monitor":
        return { valid: true, value: String(raw) };
      case "monitors":
        return { valid: true, value: raw.slice() };
      case "select":
        if (field.type === "string") return { valid: true, value: String(raw) };
        return parseInteger(raw);
      case "number": {
        var parsed = parseInteger(raw);
        if (!parsed.valid) return parsed;
        return { valid: true, value: clamp(parsed.value, field.min, field.max) };
      }
      case "refreshRate": {
        var rate = decodeRefreshRate(raw);
        return rate ? { valid: true, value: rate } : { valid: false };
      }
      default:
        throw new Error("unknown field kind: " + field.kind);
    }
  }

  function applyChange(settings, field, raw) {
    var result = coerceValue(field, raw);
    if (!result.valid) return settings;
    var next = Object.assign({}, settings);
    if (field.kind === "refreshRate") {
      next.numerator = result.value.numerator;
      next.denominator = result.value.denominator;
    } else {
      next[field.key] = result.value;
    }
    return next;
  }

  function isVisible(field, settings) {
    var rule = field.visibleWhen;
    if (!rule) return true;
    return rule.values.map(String).indexOf(String(settings[rule.key])) !== -1;
  }

  function formatMonitorLabel(monitor, primaryText) {
    var name = monitor.name || "Display";
    var num = monitor.displayNum || 1;
    if (monitor.isPrimary) return name + " (" + primaryText + ", " + num + ")";
    return name + " (" + num + ")";
  }

  function monitorOptions(monitors, selected, text) {
    var options = monitors.map(function (m) {
      return { value: m.id, label: formatMonitorLabel(m, text.primary), disconnected: false };
    });
    var ids = Array.isArray(selected) ? selected : selected ? [selected] : [];
    ids.forEach(function (id) {
      var present = options.some(function (o) { return o.value === id; });
      if (!present) options.push({ value: id, label: text.disconnected, disconnected: true });
    });
    return options;
  }

  function autoSelectMonitors(field, settings, monitors) {
    if (!monitors.length) return settings;
    if (field.kind === "monitor" && !settings[field.key]) {
      return applyChange(settings, field, monitors[0].id);
    }
    if (field.kind === "monitors" && !(settings[field.key] && settings[field.key].length)) {
      return applyChange(settings, field, monitors.map(function (m) { return m.id; }));
    }
    return settings;
  }

  function formatInputName(code) {
    var hex = Number(code).toString(16).toUpperCase();
    if (hex.length < 2) hex = "0" + hex;
    return "Input 0x" + hex;
  }

  function portOptions(field, monitor, saved) {
    var options = monitor && monitor.inputs && monitor.inputs.length
      ? monitor.inputs.map(function (p) { return { value: p.code, label: p.name }; })
      : field.fallback.map(function (o) { return { value: o.value, label: o.label }; });
    if (saved === undefined || saved === null) return options;
    if (options.some(function (o) { return o.value === saved; })) return options;
    var known = field.fallback.filter(function (o) { return o.value === saved; })[0];
    options.push({ value: saved, label: known ? known.label : formatInputName(saved) });
    return options;
  }

  function encodeRefreshRate(rate) {
    return rate.numerator + "/" + (rate.denominator || 1);
  }

  function decodeRefreshRate(value) {
    var match = /^(\d+)\/(\d+)$/.exec(String(value));
    if (!match) return null;
    var numerator = parseInt(match[1], 10);
    if (!numerator) return null;
    return { numerator: numerator, denominator: parseInt(match[2], 10) || 1 };
  }

  function formatHz(rate) {
    return (rate.numerator / (rate.denominator || 1)).toFixed(2) + " Hz";
  }

  function refreshRateOptions(rates, saved) {
    var options = rates.map(function (r) {
      return { value: encodeRefreshRate(r), label: formatHz(r) };
    });
    if (!saved || !saved.numerator) return options;
    var value = encodeRefreshRate(saved);
    if (!options.some(function (o) { return o.value === value; })) {
      options.push({ value: value, label: formatHz(saved) });
    }
    return options;
  }

  function acceptsRefreshRates(settings, payload) {
    return payload.monitorId === settings.monitorId;
  }

  var api = {
    mergeSettings: mergeSettings,
    coerceValue: coerceValue,
    applyChange: applyChange,
    isVisible: isVisible,
    formatMonitorLabel: formatMonitorLabel,
    monitorOptions: monitorOptions,
    autoSelectMonitors: autoSelectMonitors,
    formatInputName: formatInputName,
    portOptions: portOptions,
    encodeRefreshRate: encodeRefreshRate,
    decodeRefreshRate: decodeRefreshRate,
    refreshRateOptions: refreshRateOptions,
    acceptsRefreshRates: acceptsRefreshRates,
  };

  root.InspectorCore = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : this);
