"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const core = require("../../assets/ui/js/inspector-core.js");

const fixture = JSON.parse(
  fs.readFileSync(path.join(__dirname, "..", "..", "testdata", "input-names.json"), "utf8")
);

const fallback = [
  { value: 15, label: "DisplayPort 1" },
  { value: 17, label: "HDMI 1" },
];
const monitorField = { key: "monitorId", kind: "monitor", label: "Monitors" };
const monitorsField = { key: "monitorIds", kind: "monitors", label: "Monitors" };
const modeField = {
  key: "mode", kind: "select", type: "string", label: "Mode", default: "direct", zeroIsUnset: true,
  options: [{ value: "direct", labelKey: "Direct" }, { value: "toggle", labelKey: "Toggle" }],
};
const portField = {
  key: "port", kind: "select", type: "int", label: "Port", default: 15, zeroIsUnset: true,
  source: "monitorInputs", fallback, visibleWhen: { key: "mode", values: ["direct"] },
};
const portBField = {
  key: "portB", kind: "select", type: "int", label: "PortB", default: 17, zeroIsUnset: true,
  source: "monitorInputs", fallback, visibleWhen: { key: "mode", values: ["toggle"] },
};
const levelField = { key: "level", kind: "number", label: "Value", default: 50, min: 0, max: 100 };
const rateField = { keys: ["numerator", "denominator"], kind: "refreshRate", label: "RefreshRate" };
const schema = { fields: [monitorField, modeField, portField, portBField, levelField] };

const monitors = [
  { id: "a", name: "Dell", displayNum: 1, isPrimary: true },
  { id: "b", name: "LG", displayNum: 2 },
];
const text = { primary: "Primary", disconnected: "Disconnected" };

test("mergeSettings applies defaults", () => {
  assert.deepEqual(core.mergeSettings(schema, {}), { mode: "direct", port: 15, portB: 17, level: 50 });
});

test("mergeSettings keeps saved and unknown keys", () => {
  assert.deepEqual(
    core.mergeSettings(schema, { mode: "toggle", level: 0, currentState: 1 }),
    { mode: "toggle", port: 15, portB: 17, level: 0, currentState: 1 }
  );
});

test("mergeSettings treats zero as unset where marked", () => {
  const merged = core.mergeSettings(schema, { mode: "", port: 0 });
  assert.equal(merged.mode, "direct");
  assert.equal(merged.port, 15);
});

test("mergeSettings reads a zero denominator as 1", () => {
  assert.deepEqual(
    core.mergeSettings({ fields: [rateField] }, { numerator: 60, denominator: 0 }),
    { numerator: 60, denominator: 1 }
  );
});

test("coerceValue converts selects by type", () => {
  assert.deepEqual(core.coerceValue(modeField, "toggle"), { valid: true, value: "toggle" });
  assert.deepEqual(core.coerceValue(portField, "17"), { valid: true, value: 17 });
  assert.equal(core.coerceValue(portField, "x").valid, false);
});

test("coerceValue validates and clamps numbers", () => {
  for (const raw of ["", "1.5", "abc"]) {
    assert.equal(core.coerceValue(levelField, raw).valid, false, raw);
  }
  assert.equal(core.coerceValue(levelField, "150").value, 100);
  assert.equal(core.coerceValue(levelField, "-5").value, 0);
  assert.equal(core.coerceValue(levelField, " 42 ").value, 42);
});

test("applyChange keeps unknown keys and does not mutate", () => {
  const settings = { mode: "direct", level: 50, currentState: 1 };
  const next = core.applyChange(settings, levelField, "70");
  assert.deepEqual(next, { mode: "direct", level: 70, currentState: 1 });
  assert.equal(settings.level, 50);
});

test("applyChange returns the same settings for invalid input", () => {
  const settings = { level: 50 };
  assert.equal(core.applyChange(settings, levelField, ""), settings);
});

test("applyChange writes both refresh rate keys", () => {
  assert.deepEqual(core.applyChange({}, rateField, "144/1"), { numerator: 144, denominator: 1 });
});

test("isVisible follows visibleWhen", () => {
  assert.equal(core.isVisible(portField, { mode: "direct" }), true);
  assert.equal(core.isVisible(portField, { mode: "toggle" }), false);
  assert.equal(core.isVisible(levelField, { mode: "toggle" }), true);
});

test("monitorOptions labels monitors", () => {
  assert.deepEqual(core.monitorOptions(monitors, "a", text), [
    { value: "a", label: "Dell (Primary, 1)", disconnected: false },
    { value: "b", label: "LG (2)", disconnected: false },
  ]);
});

test("monitorOptions keeps a disconnected single selection", () => {
  const options = core.monitorOptions(monitors, "gone", text);
  assert.deepEqual(options[2], { value: "gone", label: "Disconnected", disconnected: true });
});

test("monitorOptions keeps a disconnected id in a multi selection", () => {
  const options = core.monitorOptions(monitors, ["b", "gone"], text);
  assert.equal(options.length, 3);
  assert.deepEqual(options[2], { value: "gone", label: "Disconnected", disconnected: true });
});

test("autoSelectMonitors fills empty selections only", () => {
  assert.equal(core.autoSelectMonitors(monitorField, {}, monitors).monitorId, "a");
  assert.deepEqual(core.autoSelectMonitors(monitorsField, {}, monitors).monitorIds, ["a", "b"]);
  assert.deepEqual(core.autoSelectMonitors(monitorsField, { monitorIds: [] }, monitors).monitorIds, ["a", "b"]);
  const chosen = { monitorId: "b" };
  assert.equal(core.autoSelectMonitors(monitorField, chosen, monitors), chosen);
  const empty = {};
  assert.equal(core.autoSelectMonitors(monitorField, empty, []), empty);
});

test("portOptions uses the monitor inputs", () => {
  const monitor = { inputs: [{ code: 17, name: "HDMI 1" }, { code: 27, name: "USB-C / Type-C" }] };
  assert.deepEqual(core.portOptions(portField, monitor, 17), [
    { value: 17, label: "HDMI 1" },
    { value: 27, label: "USB-C / Type-C" },
  ]);
});

test("portOptions falls back without a monitor", () => {
  assert.deepEqual(core.portOptions(portField, null, 15), fallback);
});

test("portOptions keeps a saved code the monitor did not report", () => {
  const monitor = { inputs: [{ code: 27, name: "USB-C / Type-C" }] };
  assert.deepEqual(core.portOptions(portField, monitor, 15).at(-1), { value: 15, label: "DisplayPort 1" });
  assert.deepEqual(core.portOptions(portField, monitor, 32).at(-1), { value: 32, label: "Input 0x20" });
});

test("portOptions does not leak an extra option into another field", () => {
  const monitor = { inputs: [{ code: 27, name: "USB-C / Type-C" }] };
  core.portOptions(portField, monitor, 15);
  const values = core.portOptions(portBField, monitor, 17).map((o) => o.value);
  assert.deepEqual(values, [27, 17]);
});

test("formatInputName matches the shared fixture", () => {
  for (const c of fixture.unknown) {
    assert.equal(core.formatInputName(c.code), c.name);
  }
});

test("refresh rate values round trip", () => {
  assert.equal(core.encodeRefreshRate({ numerator: 60, denominator: 0 }), "60/1");
  assert.deepEqual(core.decodeRefreshRate("60/1"), { numerator: 60, denominator: 1 });
  assert.deepEqual(core.decodeRefreshRate("60/0"), { numerator: 60, denominator: 1 });
  assert.equal(core.decodeRefreshRate("x"), null);
  const rate = { numerator: 60000, denominator: 1001 };
  assert.deepEqual(core.decodeRefreshRate(core.encodeRefreshRate(rate)), rate);
});

test("refreshRateOptions keeps a saved rate missing from the list", () => {
  const rates = [{ numerator: 60, denominator: 1 }, { numerator: 144, denominator: 1 }];
  assert.equal(core.refreshRateOptions(rates, { numerator: 60, denominator: 1 }).length, 2);
  assert.deepEqual(core.refreshRateOptions(rates, { numerator: 75, denominator: 1 }).at(-1), {
    value: "75/1",
    label: "75.00 Hz",
  });
});

test("acceptsRefreshRates rejects stale responses", () => {
  assert.equal(core.acceptsRefreshRates({ monitorId: "a" }, { monitorId: "a" }), true);
  assert.equal(core.acceptsRefreshRates({ monitorId: "a" }, { monitorId: "b" }), false);
});
