const fs = require("fs");
const vm = require("vm");

const expectedLanguages = [
  "en","pl","de","fr","es","it","pt","nl","sv","no","da","fi","cs","sk","hu","ro","tr","id","ms","vi",
  "ru","uk","bg","el","ar","he","hi","zh","ja","ko"
];
const expectedKeys = ["title","message","install","later","checking","upToDate","downloading","restarting","portableOnly"];
const source = fs.readFileSync("app/web/app.js", "utf8");
const match = source.match(/const UPDATE_TEXT = (\{[\s\S]*?\n\});\n\nfunction updateText/);
if (!match) {
  console.error("UPDATE I18N ERROR: cannot find UPDATE_TEXT in app/web/app.js");
  process.exit(1);
}
let data;
try {
  data = vm.runInNewContext(`(${match[1]})`);
} catch (error) {
  console.error("UPDATE I18N ERROR:", error.message);
  process.exit(1);
}

const placeholders = value => [...String(value).matchAll(/\{([A-Za-z0-9_]+)\}/g)].map(match => match[1]).sort().join(",");
const reference = data.en;
for (const code of expectedLanguages) {
  if (!data[code]) {
    console.error(`UPDATE I18N ERROR: missing language ${code}`);
    process.exit(1);
  }
  for (const key of expectedKeys) {
    if (!String(data[code][key] || "").trim()) {
      console.error(`UPDATE I18N ERROR: ${code}/${key} is empty`);
      process.exit(1);
    }
    if (placeholders(data[code][key]) !== placeholders(reference[key])) {
      console.error(`UPDATE I18N ERROR: ${code}/${key} placeholders differ from English`);
      process.exit(1);
    }
  }
}
const extra = Object.keys(data).filter(code => !expectedLanguages.includes(code));
if (extra.length) {
  console.error(`UPDATE I18N ERROR: unexpected languages: ${extra.join(", ")}`);
  process.exit(1);
}
console.log(`UPDATE I18N OK: ${expectedLanguages.length}/30 languages, ${expectedKeys.length} strings each`);
