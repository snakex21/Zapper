const fs = require("fs");
const vm = require("vm");

function loadI18n() {
  let source = fs.readFileSync("app/web/i18n.js", "utf8");
  source = source.replace(
    "window.ZapperI18n = {",
    "window.__TEXT = TEXT; window.__PACKS = PACKS; window.ZapperI18n = {"
  );
  const window = { dispatchEvent() {} };
  const context = {
    window,
    document: {},
    Node: { TEXT_NODE: 3, ELEMENT_NODE: 1 },
    NodeFilter: {},
    MutationObserver: function MutationObserver() {},
    CustomEvent: function CustomEvent() {}
  };
  vm.createContext(context);
  vm.runInContext(source, context);
  return { text: window.__TEXT, packs: window.__PACKS };
}

function asciiJson(value) {
  return JSON.stringify(value).replace(/[\u0080-\uFFFF]/g, ch => `\\u${ch.charCodeAt(0).toString(16).padStart(4, "0")}`);
}

const { text, packs } = loadI18n();
const language = process.argv[2] || "";
if (!language) {
  process.stdout.write(asciiJson({ text, packs }));
  process.exit(0);
}
const pack = packs[language] || {};
const missing = {};
for (const [key, pair] of Object.entries(text)) {
  if (language === "en") continue;
  if (language === "pl") continue;
  if (!Object.prototype.hasOwnProperty.call(pack, key)) missing[key] = pair[1];
}
process.stdout.write(asciiJson({ language, missing, count: Object.keys(missing).length }));
