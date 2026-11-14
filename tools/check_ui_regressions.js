// Deterministic UI state tests. No WebView2 or physical device is required.
const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { execFileSync } = require("node:child_process");
const root = path.resolve(__dirname, "..");
const source = fs.readFileSync(path.join(root, "app/web/app.js"), "utf8");

function fixture() {
  const elements = new Map();
  const element = id => {
    if (!elements.has(id)) elements.set(id, {
      id, value: "", textContent: "", innerHTML: "", disabled: false, readOnly: false,
      dataset: {}, classList: { add() {}, remove() {}, toggle() {} }, handlers: {},
      addEventListener(type, fn) { this.handlers[type] = fn; },
      close() { this.closed = true; this.handlers.close?.(); },
      blur() {}, querySelectorAll() { return []; }
    });
    return elements.get(id);
  };
  const document = { getElementById: element, addEventListener() {}, querySelectorAll() { return []; } };
  const window = { addEventListener() {} };
  const context = vm.createContext({
    document, window, console, URL, Intl, Date,
    setTimeout() { return 1; }, clearTimeout() {}, setInterval() {}, clearInterval() {}
  });
  const run = js => vm.runInContext(js, context);
  run(source);
  run(`renderAll=()=>{}; renderDeviceStatus=()=>{}; toast=()=>{};
    clearError=()=>{}; showError=()=>{};
    window.zapperReportError=(context,error)=>{window.lastError=String(error)};
    bindStaticActions();`);
  const event = id => ({ currentTarget: element(id) });
  const edit = value => { element("ai-import-input").value = value; element("ai-import-input").handlers.input(); };
  return { element, window, run, event, edit };
}
function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const raw = '{"person_id":"A","phases":[]}';
const preview = { persons: [{ person_id: "A", person_name: "A", summary: [] }] };
const snapshot = { config: { profiles: [] }, persons: [{ id: "A", name: "A" }] };
async function validate(f) {
  f.edit(raw);
  f.window.apiPreviewAIProfile = async () => preview;
  await f.run('previewAIImport({currentTarget:document.getElementById("preview-ai-import")})');
}

for (const change of ["edit", "edit-back", "close"]) {
  test(`Pending AI preview is invalidated by ${change}`, async () => {
    const f = fixture(), response = deferred();
    f.edit(raw);
    f.window.apiPreviewAIProfile = () => response.promise;
    const pending = f.run('previewAIImport({currentTarget:document.getElementById("preview-ai-import")})');
    if (change === "close") f.element("ai-dialog").close();
    else { f.edit('{"person_id":"B"}'); if (change === "edit-back") f.edit(raw); }
    response.resolve(preview); await pending;
    assert.equal(f.run("validatedAIJSON"), "");
    assert.equal(f.element("apply-ai-import").disabled, true);
  });
}

test("Import captures approved data; later invalidation cannot break successful completion", async () => {
  const f = fixture(), response = deferred();
  await validate(f);
  let calls = 0, submitted;
  f.window.apiApplyAIProfile = value => { calls++; submitted = value; return response.promise; };
  f.run('armedDelete="apply-ai-profile"');
  const pending = f.run('applyAIImport({currentTarget:document.getElementById("apply-ai-import")})');
  assert.equal(f.element("ai-import-input").readOnly, true);
  assert.equal(f.element("preview-ai-import").disabled, true);
  await f.run('applyAIImport({currentTarget:document.getElementById("apply-ai-import")})');
  // A close/reopen or programmatic input event must not alter the captured payload.
  f.edit("new draft");
  response.resolve(snapshot); await pending;
  assert.equal(calls, 1);
  assert.equal(submitted, raw);
  assert.equal(f.window.lastError, undefined);
  assert.equal(f.element("ai-dialog").closed, true);
  assert.equal(f.element("apply-ai-import").disabled, true);
  assert.equal(f.element("ai-import-input").readOnly, false);
  assert.equal(f.element("ai-import-input").value, "new draft");
  assert.equal(f.run("selectedPersonID"), "A");
});

test("Failed import unlocks controls and retains validated payload for retry", async () => {
  const f = fixture(); await validate(f);
  f.window.apiApplyAIProfile = async () => { throw new Error("disk full"); };
  f.run('armedDelete="apply-ai-profile"');
  await f.run('applyAIImport({currentTarget:document.getElementById("apply-ai-import")})');
  assert.match(f.window.lastError, /disk full/);
  assert.equal(f.run("validatedAIJSON"), raw);
  assert.equal(f.element("apply-ai-import").disabled, false);
  assert.equal(f.element("ai-import-input").readOnly, false);
  assert.equal(f.element("preview-ai-import").disabled, false);
});

for (const dirty of [true, false]) {
  test(`Rename preserves correct profile state (dirty=${dirty})`, async () => {
    const f = fixture();
    f.run(`snapshot={config:{profiles:[{id:"p",person_id:"A",name:"A",phases:[{name:"saved"}]}]},persons:[{id:"A",name:"A"}]};
      syncDraft(); editedPersonID="A";`);
    if (dirty) f.run('draftConfig.profiles[0].phases[0].name="unsaved";profilesDirty=true');
    f.element("person-edit-name").value = "renamed";
    f.window.apiUpdatePerson = async () => ({
      config: { profiles: [{ id: "p", person_id: "A", name: "renamed", phases: [{ name: "saved" }] }] },
      persons: [{ id: "A", name: "renamed" }]
    });
    await f.run('savePersonName({currentTarget:document.getElementById("confirm-person-edit")})');
    assert.equal(f.run("draftConfig.profiles[0].name"), "renamed");
    assert.equal(f.run("draftConfig.profiles[0].phases[0].name"), dirty ? "unsaved" : "saved");
    assert.equal(f.run("profilesDirty"), dirty);
    assert.equal(f.element("save-profiles").disabled, !dirty);
  });
}

test("Piped i18n introspection emits complete JSON", () => {
  const output = execFileSync(process.execPath, ["tools/i18n_introspect.js"], { cwd: root, encoding: "utf8" });
  const data = JSON.parse(output);
  assert.ok(Object.keys(data.text).length > 400);
  assert.ok(output.length > 65536);
});

// --- i18n: formy mnogie i komunikaty urządzenia z backendu -----------------
function loadI18nRuntime() {
  const i18nSource = fs.readFileSync(path.join(root, "app/web/i18n.js"), "utf8");
  const window = { dispatchEvent() {} };
  const context = vm.createContext({
    window, Intl, console,
    document: {},
    Node: { TEXT_NODE: 3, ELEMENT_NODE: 1 },
    NodeFilter: {},
    MutationObserver: function MutationObserver() {},
    CustomEvent: function CustomEvent() {},
    fetch: async url => {
      const match = String(url).match(/^\/locale\/ui\/([a-z]+)$/);
      const file = match && path.join(root, "locales", `ui.${match[1]}.json`);
      if (!file || !fs.existsSync(file)) return { ok: false, status: 404 };
      const data = JSON.parse(fs.readFileSync(file, "utf8").replace(/^\uFEFF/, ""));
      return { ok: true, json: async () => data };
    }
  });
  vm.runInContext(i18nSource, context);
  return window.ZapperI18n;
}

test("Step counter uses Intl.PluralRules forms (pl/en exact)", () => {
  const i18n = loadI18nRuntime();
  const pl = n => i18n.plural("countSteps", n, {}, "pl");
  assert.equal(pl(1), "1 krok");
  assert.equal(pl(2), "2 kroki");
  assert.equal(pl(4), "4 kroki");
  assert.equal(pl(5), "5 kroków");
  assert.equal(pl(12), "12 kroków");
  assert.equal(pl(22), "22 kroki");
  assert.equal(i18n.plural("countSteps", 1, {}, "en"), "1 step");
  assert.equal(i18n.plural("countSteps", 3, {}, "en"), "3 steps");
  assert.equal(i18n.plural("countDays", 1, {}, "pl"), "1 dzień");
  assert.equal(i18n.plural("countDays", 3, {}, "pl"), "3 dni");
  assert.equal(i18n.plural("countPeopleAdded", 1, {}, "pl"), "Dodano 1 osobę");
  assert.equal(i18n.plural("countPeopleAdded", 3, {}, "pl"), "Dodano 3 osoby");
  assert.equal(i18n.plural("countPeopleAdded", 5, {}, "pl"), "Dodano 5 osób");
  assert.equal(i18n.plural("countCatalogItems", 1, {}, "en"), "1 item in the catalog");
});

test("Plural counters distinguish forms in every language that has them", async () => {
  const i18n = loadI18nRuntime();
  const bases = ["countSteps", "countDays", "countPrograms", "countPhases", "countCatalogItems", "countPeopleAdded"];
  for (const code of Object.keys(i18n.languages)) {
    await i18n.loadLanguage(code);
    for (const base of bases) {
      for (const n of [0, 1, 2, 3, 5, 11, 12, 21, 22, 25, 101]) {
        const text = i18n.plural(base, n, {}, code);
        assert.ok(text.includes(String(n)), `${code}/${base}(${n}) must contain the number: ${text}`);
        assert.ok(!text.includes("{count}"), `${code}/${base}(${n}) left a placeholder: ${text}`);
      }
    }
  }
  // Języki, w których 1, 2 i 5 to trzy różne kategorie CLDR, muszą mieć trzy różne formy.
  for (const code of ["pl", "ru", "uk", "cs", "sk"]) {
    const forms = [1, 2, 5].map(n => i18n.plural("countSteps", n, {}, code).replace(/\d+/, "#"));
    assert.equal(new Set(forms).size, 3, `${code}: ${forms.join(" | ")}`);
  }
  for (const code of ["en", "de", "fr", "es", "it", "nl"]) {
    const forms = [1, 2].map(n => i18n.plural("countSteps", n, {}, code).replace(/\d+/, "#"));
    assert.notEqual(forms[0], forms[1], `${code}: singular and plural must differ`);
  }
});

test("Backend device messages are translated by key and by Polish error template", async () => {
  const i18n = loadI18nRuntime();
  await i18n.loadLanguage("de");
  assert.equal(i18n.phrase("devBoardReady", "pl"), "Płytka gotowa");
  assert.equal(i18n.phrase("devBoardReady", "en"), "Board ready");
  assert.notEqual(i18n.phrase("devBoardReady", "de"), "Płytka gotowa");
  // Odrzucone wywołanie apiDevice* zwraca polski tekst szablonu — tłumaczymy go z parametrami.
  const en = i18n.translateSource("nie można otworzyć COM7: Access is denied.", "en");
  assert.equal(en, "Cannot open COM7: Access is denied.");
  assert.equal(i18n.translateSource("krok 3 ma częstotliwość poza zakresem 0,1 Hz–4 MHz", "en"),
    "Step 3 has a frequency outside the 0.1 Hz–4 MHz range");
  const de = i18n.translateSource("płytka nie jest połączona", "de");
  assert.equal(de, i18n.phrase("devErrNotConnected", "de"));
  assert.ok(!/płytka/.test(de));
  assert.equal(i18n.translateSource("płytka nie jest połączona", "pl"), "płytka nie jest połączona");
});

test("Device view renders message_key/message_params instead of Polish text", () => {
  const f = fixture();
  const i18n = loadI18nRuntime();
  i18n.setLanguage = () => "en";
  f.window.ZapperI18n = i18n;
  f.run(`deviceStatus = {
    state: "error", connected: true, ready: true, port: "COM3",
    message: "Nie udało się wysłać sesji", message_key: "devSendFailed",
    last_error: 'błąd wysyłania "CLEAR": timeout', last_error_key: "devErrSendCommand",
    last_error_params: { command: '"CLEAR"', error: "timeout" }
  }`);
  assert.equal(f.run(`deviceMessageText(deviceStatus.message_key, deviceStatus.message_params, deviceStatus.message)`), "Could not send the session");
  assert.equal(f.run(`deviceMessageText(deviceStatus.last_error_key, deviceStatus.last_error_params, deviceStatus.last_error)`), 'Error sending "CLEAR": timeout');
  // Surowa linia protokołu (bez klucza) zostaje bez zmian.
  assert.equal(f.run(`deviceMessageText("", null, "STATE RUNNING 1 2 7830 1000")`), "STATE RUNNING 1 2 7830 1000");
  assert.equal(f.run(`uiPlural("countSteps", 1)`), "1 step");
  assert.equal(f.run(`uiPlural("countSteps", 4)`), "4 steps");
});
