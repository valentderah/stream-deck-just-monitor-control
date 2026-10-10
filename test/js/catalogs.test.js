const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const assets = path.join(__dirname, "..", "..", "assets");
const locales = ["en", "de", "es", "fr", "ja", "ko", "ru", "zh_CN", "zh_TW"];

function load(name) {
  return JSON.parse(fs.readFileSync(path.join(assets, name + ".json"), "utf8"));
}

function keyPaths(value, prefix) {
  if (value === null || typeof value !== "object") return [prefix];
  return Object.keys(value).flatMap(function (key) {
    return keyPaths(value[key], prefix ? prefix + "." + key : key);
  });
}

test("every catalog is valid JSON with the same keys as en", function () {
  const want = keyPaths(load("en"), "").sort();
  locales.forEach(function (locale) {
    assert.deepEqual(keyPaths(load(locale), "").sort(), want, locale);
  });
});

test("every dial action in the manifest has trigger descriptions in en", function () {
  const en = load("en");
  load("manifest").Actions.forEach(function (action) {
    if (!action.Encoder) return;
    assert.deepEqual(en[action.UUID].Encoder.TriggerDescription, action.Encoder.TriggerDescription, action.UUID);
  });
});
