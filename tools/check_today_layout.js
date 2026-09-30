// Synthetic DOM/string regressions for the Today view. Not a WebView2 visual test.
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { execFileSync } = require('node:child_process');
const root = path.resolve(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'app/web/app.js'), 'utf8');
const text = JSON.parse(execFileSync(process.execPath, ['tools/i18n_introspect.js'], { cwd: root, encoding: 'utf8' })).text;
function fixture(plans = [], completions = {}) {
  const elements = new Map(), timers = new Map();
  let nextTimer = 0;
  const element = id => {
    if (!elements.has(id)) elements.set(id, {
      id, value: '', textContent: '', innerHTML: '', disabled: false, isConnected: true,
      dataset: {}, style: {}, handlers: {}, classList: { add() {}, remove() {}, toggle() {} },
      addEventListener(type, fn) { this.handlers[type] = fn; }, querySelectorAll() { return []; }
    });
    return elements.get(id);
  };
  const window = { addEventListener() {}, ZapperI18n: { locale: 'pl-PL', translateSource: source => source, t: key => text[key]?.[0] || key } };
  const context = vm.createContext({ window, console, URL, Intl, Date,
    document: { getElementById: element, addEventListener() {}, querySelectorAll() { return []; } },
    setTimeout(fn) { timers.set(++nextTimer, fn); return nextTimer; }, clearTimeout(id) { timers.delete(id); },
    setInterval() {}, clearInterval() {}
  });
  const run = code => vm.runInContext(code, context);
  run(source);
  const data = { today: plans, progress: { start_date: '2030-04-10', completions }, config: { profiles: [] }, profile_states: [], today_remaining_seconds: 420 };
  context.fixtureSnapshot = data;
  run(`snapshot=fixtureSnapshot; clearError=()=>{}; showError=()=>{}; toast=()=>{};
    renderDeviceStatus=()=>{}; renderHistory=()=>{}; renderSchedule=()=>{}; renderDeviceSessions=()=>{};
    renderAll=()=>renderToday(); bindStaticActions(); renderToday();`);
  return { run, window, element, data, timers, html: () => element('today-list').innerHTML,
    click(id, selector, button) { return element(id).handlers.click({ target: { closest: query => query === selector ? button : null } }); } };
}
const plan = (id, changes = {}) => ({ status: 'session', session_id: id, profile_id: 'sample', profile_name: 'Test person', program: 'Test program', phase_name: 'Test phase', planned_date: '2030-04-10', available: true, time: '7 min', ...changes });
function sections(html) {
  return { today: html.split('</section>')[0], overdue: html.split('</section>')[1] || '' };
}
function ids(html) { return [...html.matchAll(/data-session-done="([^"]+)"/g)].map(match => match[1]); }

test('Today comes first, retaining all rows and backend order within each section', () => {
  const f = fixture([plan('old-1', { overdue: true }), plan('old-2', { overdue: true, available: false }), plan('today-1'), plan('today-2')]);
  const s = sections(f.html());
  assert.deepEqual(ids(s.today), ['today-1', 'today-2']);
  assert.deepEqual(ids(s.overdue), ['old-1', 'old-2']);
  assert.match(s.today, /Na dziś/); assert.match(s.overdue, /Zaległe/);
  assert.match(s.overdue, /data-session-done="old-2"[^>]+disabled/);
  assert.match(s.overdue, /data-dismiss-session-group="old-2"/);
});
test('All overdue keeps a short empty Today group and every overdue item', () => {
  const f = fixture(Array.from({ length: 51 }, (_, i) => plan(`old-${i}`, { overdue: true })));
  const s = sections(f.html());
  assert.match(s.today, /Brak sesji na dzisiaj/); assert.equal(ids(s.overdue).length, 51);
});
test('Completed old sessions stay outside overdue and retain undo action', () => {
  const f = fixture([plan('done', { overdue: true }), plan('old', { overdue: true })], { done: { completed_at: '2030-04-10T09:00:00Z' } });
  const s = sections(f.html());
  assert.deepEqual(ids(s.today), ['done']); assert.deepEqual(ids(s.overdue), ['old']);
  assert.match(s.today, /data-done="true"/); assert.doesNotMatch(s.today, /data-dismiss-session-group/);
});
test('Future/not-started and rest status rows remain visible and non-interactive', () => {
  const f = fixture([plan('future', { status: 'waiting', available: false, planned_date: '2030-05-10' }), plan('rest', { status: 'rest' })]);
  assert.equal((f.html().match(/<article /g) || []).length, 2);
  assert.equal(ids(f.html()).length, 0); assert.doesNotMatch(f.html(), /overdue-sessions-heading/);
  assert.equal((f.html().match(/class="done-button" disabled/g) || []).length, 2);
});
test('No profiles and no overdue clears stale content after a repeated render', () => {
  const f = fixture([plan('old', { overdue: true })]);
  f.data.profile_states = [{ profile_id: 'sample', profile_name: 'Test person', overdue_count: 1 }];
  f.run('renderToday()'); assert.ok(f.element('overdue-actions').innerHTML);
  f.data.today = []; f.data.profile_states = []; f.run('renderToday()');
  assert.match(f.html(), /Nie ma jeszcze profili/); assert.equal(f.element('overdue-actions').innerHTML, '');
});
test('Shared explanation occurs once; every person retains correctly escaped dismissal target', () => {
  const f = fixture([plan('old', { overdue: true })]);
  f.data.profile_states = [{ profile_id: 'A"<>', profile_name: '<Test A>', overdue_count: 3 }, { profile_id: 'B', profile_name: 'Test B', overdue_count: 2 }];
  f.run('renderToday()'); const html = f.element('overdue-actions').innerHTML;
  assert.equal((html.match(/class="overdue-help"/g) || []).length, 1);
  assert.equal((html.match(/data-dismiss-overdue=/g) || []).length, 2);
  assert.match(html, /A&quot;&lt;&gt;/); assert.match(html, /&lt;Test A&gt;/);
  assert.match(html, /NIE zostaną zapisane jako wykonane/);
});
test('Partial repetition group keeps next part, count, blocked and paused state', () => {
  const group = { session_group_id: 'series', repeat_count: 3, overdue: true };
  const f = fixture([plan('series:1', { ...group, repeat_index: 1 }), plan('series:2', { ...group, repeat_index: 2, available: false, paused: true, remaining_seconds: 120 }), plan('series:3', { ...group, repeat_index: 3, available: false })], { 'series:1': {} });
  assert.deepEqual(ids(f.html()), ['series:2']); assert.match(f.html(), /1\/3/);
  assert.match(f.html(), /data-session-done="series:2"[^>]+disabled/);
  assert.match(f.html(), /Wstrzyman/);
});
test('Grouping depends on scheduler flags, not browser timezone or calendar date', () => {
  const previous = process.env.TZ;
  try {
    const outputs = ['UTC', 'America/Los_Angeles', 'Pacific/Auckland'].map(tz => {
      process.env.TZ = tz;
      return sections(fixture([plan('backend-today', { planned_date: '2030-04-09' }), plan('backend-overdue', { planned_date: '2030-04-10', overdue: true })]).html());
    });
    for (const output of outputs) {
      assert.deepEqual(ids(output.today), ['backend-today']); assert.deepEqual(ids(output.overdue), ['backend-overdue']);
    }
  } finally { if (previous === undefined) delete process.env.TZ; else process.env.TZ = previous; }
});
test('Rendering changes neither profile draft nor scheduling/completion data', () => {
  const f = fixture([plan('today')]); const before = JSON.stringify(f.data);
  f.run('profilesDirty=true; draftConfig={profiles:[{name:"unsaved draft"}]}; renderToday(); renderToday()');
  assert.equal(JSON.stringify(f.data), before); assert.equal(f.run('profilesDirty'), true);
  assert.equal(f.run('draftConfig.profiles[0].name'), 'unsaved draft');
});
for (const action of ['overdue', 'group']) {
  test(`${action} dismissal still requires two clicks, times out/cancels, and targets exact item`, async () => {
    const f = fixture([plan('old', { overdue: true })]);
    const button = f.element('test-button'); let calls = [];
    const selector = action === 'overdue' ? '[data-dismiss-overdue]' : '[data-dismiss-session-group]';
    const container = action === 'overdue' ? 'overdue-actions' : 'today-list';
    button.dataset = action === 'overdue' ? { dismissOverdue: 'profile-A' } : { dismissSessionGroup: 'old' };
    f.window[action === 'overdue' ? 'apiDismissOverdueSessions' : 'apiDismissSessionGroup'] = async id => { calls.push(id); return f.data; };
    await f.click(container, selector, button); assert.equal(calls.length, 0);
    for (const timer of [...f.timers.values()]) timer(); // Let confirmation expire without committing.
    await f.click(container, selector, button); assert.equal(calls.length, 0);
    await f.click(container, selector, button); assert.deepEqual(calls, [action === 'overdue' ? 'profile-A' : 'old']);
  });
}
test('Complete/undo keeps exact API semantics and blocks repeated click while pending', async () => {
  const f = fixture([plan('today')]), button = f.element('test-button');
  button.dataset = { sessionDone: 'today', done: 'false', outOfOrder: 'false' };
  let resolve, calls = [];
  f.window.apiSetSessionDone = (id, done) => { calls.push([id, done]); return new Promise(yes => { resolve = yes; }); };
  const pending = f.click('today-list', '[data-session-done]', button);
  await f.click('today-list', '[data-session-done]', button); assert.deepEqual(calls, [['today', true]]);
  resolve(f.data); await pending;
  button.dataset.done = 'true'; f.window.apiSetSessionDone = async (id, done) => { calls.push([id, done]); return f.data; };
  await f.click('today-list', '[data-session-done]', button); assert.deepEqual(calls[1], ['today', false]);
});
