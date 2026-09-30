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
