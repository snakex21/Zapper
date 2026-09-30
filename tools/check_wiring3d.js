// Dependency-free interaction regression checks; not a browser/WebView2 test.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync('app/web/wiring3d.js', 'utf8');
let size = { width: 1180, height: 760 }, resize, calls = [], labels = [], captured = false;
const listeners = {}, classes = new Set();
const gradient = { addColorStop() {} };
const ctx = new Proxy({}, { get(target, key) {
  if (key === 'measureText') return text => ({ width: text.length * 6 });
  if (key === 'createLinearGradient' || key === 'createRadialGradient') return () => gradient;
  if (key === 'fillText') return (...args) => labels.push(args);
  return target[key] ?? ((...args) => { assert(args.filter(x => typeof x === 'number').every(Number.isFinite)); calls.push([key, ...args]); });
}, set(target, key, value) { target[key] = value; return true; } });
const button = value => ({ dataset: { wiringView: value }, attrs: {}, setAttribute(k, v) { this.attrs[k] = v; } });
const buttons = ['iso', 'top', 'rear', 'lcdRear', 'kyRear'].map(button), labelButton = button('labels');
const canvas = {
  width: 0, height: 0, getContext: () => ctx, getBoundingClientRect: () => size,
  addEventListener: (name, fn) => listeners[name] = fn,
  classList: { add: key => classes.add(key), remove: key => classes.delete(key) },
  setPointerCapture() { captured = true; }, releasePointerCapture() { captured = false; }
};
const window = { devicePixelRatio: 2, __wires: [], __renderedWires: [], __lines: [] };
vm.runInNewContext(source.replace('const q=raised.map(project);', 'const q=raised.map(project); window.__renderedWires.push(q);').replace('function line(points, color, width=3){','function line(points, color, width=3){ window.__lines.push({points,color});').replace('function wire(points, color, width=4){', 'function wire(points, color, width=4){ window.__wires.push({points,color});').replace('  render();\n})();', '  render();\n  window.__test = {project, state};\n})();'), { window, document: {
  getElementById: id => id === 'wiring3dLabels' ? labelButton : canvas,
  querySelectorAll: () => buttons
}, ResizeObserver: class { constructor(fn) { resize = fn; } observe() {} } });
// Terminal coordinates are electrical invariants: prettier cable arcs may only move interior control points.
assert.equal(window.__wires.length, 11);
const terminals = window.__wires.map(({points}) => [points[0], points[points.length-1]]);
assert.deepEqual(JSON.parse(JSON.stringify(terminals)), [
  [{x:-437,y:-18,z:59},{x:-140,y:-12,z:38}],
  [{x:-411,y:-18,z:59},{x:-140,y:-52,z:38}],
  [{x:-385,y:-18,z:59},{x:-140,y:-132,z:38}],
  [{x:-359,y:-18,z:59},{x:-140,y:-112,z:38}],
  [{x:457,y:49,z:62},{x:140,y:-52,z:38}],
  [{x:496,y:49,z:62},{x:-112,y:-52,z:38}],
  [{x:535,y:49,z:62},{x:140,y:-112,z:38}],
  [{x:574,y:49,z:62},{x:140,y:-92,z:38}],
  [{x:613,y:49,z:62},{x:140,y:-72,z:38}],
  [{x:140,y:-292,z:12},{x:420,y:-340,z:58}],
  [{x:-112,y:-12,z:38},{x:420,y:294,z:38}]
]);
for(let i=0;i<11;i++){
  for(const index of [0,terminals[i].length-1]){
    const expected=window.__test.project(terminals[i][index]);
    const route=window.__renderedWires[i],actual=route[index===0?0:route.length-1];
    assert.equal(actual.x,expected.x);assert.equal(actual.y,expected.y);
  }
}
for(const expected of [[{x:112,y:-212,z:8},{x:112,y:-230,z:28}],[{x:112,y:-274,z:28},{x:112,y:-292,z:8}]]){
  assert(window.__lines.some(({points})=>JSON.stringify(points)===JSON.stringify(expected)), 'resistor leads stay in i5 and i1');
}
assert.equal(window.__test.state.labels, false, 'default is an uncluttered overview');
function snapshot() { return JSON.stringify(calls); }
function render(view) { calls = []; labels = []; window.wiring3dView(view); return snapshot(); }
const iso = render('iso');
assert.equal(canvas.width, 2360); assert.equal(canvas.height, 1520);
assert.equal(render('iso'), iso, 'reset is deterministic');
for (const view of ['top', 'rear', 'lcdRear', 'kyRear']) {
  assert.notEqual(render(view), iso, `${view} changes projection`);
  assert.equal(buttons.find(b => b.dataset.wiringView === view).attrs['aria-pressed'], 'true');
}
for (const [view, point] of [['lcdRear', {x:-520,y:-80,z:125}], ['kyRear', {x:535,y:-55,z:130}]]) {
  render(view); const projected = window.__test.project(point);
  assert(Math.abs(projected.x-size.width/2)<.01 && Math.abs(projected.y-size.height/2)<.01, `${view} centers the selected module`);
}
assert.equal(render('invalid'), iso, 'unknown view uses safe default');
const event = props => ({ pointerId: 1, button: 0, clientX: 200, clientY: 200, preventDefault() {}, ...props });
listeners.pointerdown(event({})); assert(captured && classes.has('dragging'));
listeners.pointermove(event({ clientX: 270, clientY: 240 }));
listeners.pointercancel(event({})); assert(!captured && !classes.has('dragging'));
calls = []; listeners.pointermove(event({ clientX: 290 })); assert.equal(calls.length, 0, 'cancelled drag remains inactive');
listeners.pointerdown(event({ button: 2 })); listeners.pointermove(event({ clientX: 230 }));
listeners.lostpointercapture(event({})); assert(!classes.has('dragging'));
for (let i = 0; i < 30; i++) listeners.wheel(event({ deltaY: i < 15 ? -1 : 1 }));
for (const key of ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', '+', '-', '0']) listeners.keydown(event({ key }));
listeners.keydown(event({ key: 'ArrowRight', shiftKey: true }));
labels = []; window.wiring3dToggleLabels(); assert.equal(labelButton.attrs['aria-pressed'], 'true');
assert(labels.some(([text]) => text.includes('i5 · D9/H5')), 'details expose critical resistor coordinates');
labels = []; window.wiring3dToggleLabels(); assert.equal(labelButton.attrs['aria-pressed'], 'false');
assert(!labels.some(([text]) => text.includes('ARDUINO')), 'labels toggle hides component callouts');
for (const [width, height] of [[600, 500], [390, 500], [0, 0], [1180, 760]]) { size = { width, height }; resize(); }
assert.equal(render('iso'), iso, 'resize/reset restores original projection');
listeners.dblclick(event({})); assert.equal(buttons[0].attrs['aria-pressed'], 'true');
console.log('PASS wiring3d: 11 exact electrical terminal pairs, clean/detail modes, presets, reset, invalid view, rotate/pan, cancel/lost capture, zoom, keyboard, labels, resize, DPR');
