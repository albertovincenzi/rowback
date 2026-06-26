'use strict';

const $ = (id) => document.getElementById(id);

let format = 'insert';
let insertMode = 'insert';
let resultData = null;
let activeTab = 0;
let removed = {};   // tableName -> {rowIndex:true}
let edits = {};     // tableName -> {rowIndex:{colIndex:{token,display}}}

/* ── Bootstrap defaults from the server (no HTML string-templating). ── */
fetch('/api/config')
  .then((r) => (r.ok ? r.json() : null))
  .then((cfg) => {
    if (!cfg) return;
    if (cfg.defaultDump && !$('dump').value) $('dump').value = cfg.defaultDump;
    if (cfg.defaultQuery && !$('query').value) $('query').value = cfg.defaultQuery;
  })
  .catch(() => {});

/* splitTuple splits a raw VALUES tuple into its top-level field tokens,
   respecting single-quoted strings, backslash escapes, and nested parens —
   mirrors the server-side splitFields so edits splice in faithfully. */
function splitTuple(raw) {
  const out = [];
  let inStr = false, esc = false, depth = 0, start = 0;
  for (let i = 0; i < raw.length; i++) {
    const c = raw[i];
    if (inStr) {
      if (esc) esc = false;
      else if (c === '\\') esc = true;
      else if (c === "'") inStr = false;
      continue;
    }
    if (c === "'") inStr = true;
    else if (c === '(') depth++;
    else if (c === ')') depth--;
    else if (c === ',' && depth === 0) { out.push(raw.slice(start, i)); start = i + 1; }
  }
  out.push(raw.slice(start));
  return out;
}

/* rebuildRaw applies a row's per-column edits to its raw tuple. */
function rebuildRaw(tname, i, raw) {
  const ov = edits[tname] && edits[tname][i];
  if (!ov) return raw;
  const keys = Object.keys(ov);
  if (!keys.length) return raw;
  const toks = splitTuple(raw);
  keys.forEach((ci) => { if (ci < toks.length) toks[ci] = ov[ci].token; });
  return toks.join(',');
}

/* origQuoted reports whether column ci of a row was originally a quoted string. */
function origQuoted(raw, ci) {
  const toks = splitTuple(raw);
  return ci < toks.length && toks[ci].charAt(0) === "'";
}

/* sqlLiteral turns a user value into a SQL token (quoted+escaped, or verbatim). */
function sqlLiteral(val, quote) {
  if (!quote) return val;
  return "'" + val.replace(/\\/g, '\\\\').replace(/'/g, "\\'") + "'";
}

document.querySelectorAll('#fmt button').forEach((b) => {
  b.addEventListener('click', () => {
    document.querySelectorAll('#fmt button').forEach((x) => x.classList.remove('on'));
    b.classList.add('on');
    format = b.dataset.v;
    $('imodeWrap').classList.toggle('hidden', format !== 'insert');
    if (resultData) updateExpHint();
  });
});
$('imode').addEventListener('change', () => {
  insertMode = $('imode').value;
  if (resultData) updateExpHint();
});

const fmt = (n) => (n || 0).toLocaleString();

/* animateCount tweens a stat element's number for a lively, readable update. */
const counters = new WeakMap();
function setStat(el, value, suffix) {
  suffix = suffix || '';
  const from = counters.get(el) || 0;
  if (from === value) { el.textContent = fmt(value) + suffix; return; }
  if (window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    counters.set(el, value); el.textContent = fmt(value) + suffix; return;
  }
  const start = performance.now(), dur = 380;
  function step(now) {
    const t = Math.min(1, (now - start) / dur);
    const eased = 1 - Math.pow(1 - t, 3);
    const cur = Math.round(from + (value - from) * eased);
    el.textContent = fmt(cur) + suffix;
    if (t < 1) requestAnimationFrame(step);
    else counters.set(el, value);
  }
  requestAnimationFrame(step);
}

function renderCoverage(d) {
  const box = $('cov');
  box.innerHTML = '';
  let has = false;
  if (d.Coverage && d.Coverage.length) {
    has = true;
    d.Coverage.forEach((c) => {
      const row = document.createElement('div');
      row.className = 'c';
      const label = document.createElement('b');
      label.textContent = c.Column + ':';
      row.appendChild(label);
      const val = document.createElement('span');
      if (c.Found >= c.Total) {
        val.className = 'ok';
        val.textContent = c.Found + '/' + c.Total + ' found';
      } else {
        val.className = c.Found === 0 ? 'bad' : 'warn';
        val.textContent = c.Found + '/' + c.Total;
        row.appendChild(val);
        const miss = document.createElement('span');
        miss.className = 'miss';
        miss.textContent = '— missing: ' + ((c.Missing || []).join(', '));
        row.appendChild(miss);
        box.appendChild(row);
        return;
      }
      row.appendChild(val);
      box.appendChild(row);
    });
  }
  if (d.RelatedRows > 0) {
    has = true;
    const rel = document.createElement('div');
    rel.className = 'rel';
    rel.textContent = '+ ' + fmt(d.RelatedRows) + ' related row(s) across ' + fmt(d.RelatedTbls) + ' table(s)';
    box.appendChild(rel);
  }
  box.classList.toggle('hidden', !has);
}

function apply(d) {
  const pct = d.pct || 0;
  $('s-pct').textContent = pct.toFixed(1) + '%';
  $('barfill').style.width = Math.min(100, pct) + '%';
  $('bar').setAttribute('aria-valuenow', Math.round(Math.min(100, pct)));
  setStat($('s-sub'), d.SubResolved | 0);
  setStat($('s-main'), d.MainScanned | 0);
  setStat($('s-match'), d.Matched | 0);
  document.querySelector('.stat.match').classList.toggle('live', (d.Matched | 0) > 0 && !d.Done);
  const st = $('status');
  if (d.err) {
    st.className = 'status err';
    st.textContent = d.err;
    finishRun('Run again');
  } else if (d.Done) {
    st.className = 'status ok';
    st.textContent = d.Message || ('Done — ' + fmt(d.Matched) + ' row(s).');
    if (d.Matched > 0) $('dl').classList.remove('hidden');
    renderCoverage(d);
    finishRun('Run again');
    loadResult();
  } else {
    st.className = 'status';
    let txt;
    if (d.Phase === 'indexing') txt = 'Building one-time index… ' + pct.toFixed(1) + '%';
    else if (d.Phase === 'schema') txt = 'Reading schema (foreign keys)…';
    else if (d.Phase === 'related') {
      txt = (d.Message || 'Scanning related tables') + ' … ' + pct.toFixed(1) + '%' +
        (d.RelatedRows > 0 ? (' — ' + fmt(d.RelatedRows) + ' related row(s) so far') : '');
    } else {
      txt = 'Scanning (' + (d.Phase || '') + ') … ' + pct.toFixed(1) + '%';
    }
    st.textContent = txt;
  }
}

function finishRun(label) {
  const run = $('run');
  run.disabled = false;
  run.classList.remove('busy');
  run.querySelector('.run-label').textContent = label;
}

function loadResult() {
  fetch('/api/result')
    .then((r) => { if (!r.ok) throw new Error('no result'); return r.json(); })
    .then((data) => {
      resultData = data;
      removed = {};
      edits = {};
      (data.tables || []).forEach((t) => { removed[t.name] = {}; edits[t.name] = {}; });
      activeTab = 0;
      $('curate').classList.remove('hidden');
      $('wrap').classList.add('wide');
      updateExpHint();
      renderTabs();
      renderGrid();
    })
    .catch(() => { /* no preview available */ });
}

function renderTabs() {
  const tabs = $('tabs');
  tabs.innerHTML = '';
  (resultData.tables || []).forEach((t, i) => {
    const b = document.createElement('button');
    b.type = 'button';
    b.className = 'tab' + (i === activeTab ? ' on' : '');
    b.setAttribute('role', 'tab');
    b.setAttribute('aria-selected', i === activeTab ? 'true' : 'false');
    const name = document.createElement('span');
    name.textContent = t.name + ' (' + fmt(t.total) + ')';
    b.appendChild(name);
    if (t.relation) {
      b.title = t.relation;
      const rel = document.createElement('span');
      rel.className = 'rel';
      rel.textContent = t.relation;
      b.appendChild(rel);
    }
    b.addEventListener('click', () => { activeTab = i; $('filter').value = ''; renderTabs(); renderGrid(); });
    tabs.appendChild(b);
  });
}

const isRemoved = (name, i) => !!(removed[name] && removed[name][i]);

function renderGrid() {
  const t = resultData.tables[activeTab];
  const wrap = $('gridwrap');
  wrap.innerHTML = '';
  // populate the bulk-edit column selector for this table
  const bcol = $('bcol');
  bcol.innerHTML = '';
  (t.columns || []).forEach((c, ci) => {
    const o = document.createElement('option');
    o.value = ci; o.textContent = c;
    bcol.appendChild(o);
  });
  $('capnote').classList.toggle('hidden', !t.capped);
  if (t.capped) {
    $('capnote').textContent = 'Only the first ' + fmt(t.rows.length) + ' of ' + fmt(t.total) +
      ' rows are shown for preview; export covers the shown rows.';
  }
  if (!(t.rows || []).length) {
    const empty = document.createElement('div');
    empty.className = 'gridempty';
    empty.textContent = 'No rows in this table.';
    wrap.appendChild(empty);
    applyFilter();
    return;
  }
  const table = document.createElement('table');
  table.className = 'dt';
  const thead = document.createElement('thead');
  const hr = document.createElement('tr');
  const hx = document.createElement('th');
  hx.className = 'x';
  hr.appendChild(hx);
  (t.columns || []).forEach((c) => {
    const th = document.createElement('th');
    th.textContent = c;
    hr.appendChild(th);
  });
  thead.appendChild(hr);
  table.appendChild(thead);
  const tbody = document.createElement('tbody');
  (t.rows || []).forEach((row, i) => {
    const tr = document.createElement('tr');
    tr.dataset.i = i;
    if (isRemoved(t.name, i)) tr.className = 'gone';
    const tdx = document.createElement('td');
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'xbtn';
    btn.textContent = isRemoved(t.name, i) ? '↺' : '✕';
    btn.title = isRemoved(t.name, i) ? 'Restore row' : 'Remove row from export';
    btn.addEventListener('click', () => toggleRow(t.name, i, tr, btn));
    tdx.appendChild(btn);
    tr.appendChild(tdx);
    (t.columns || []).forEach((c, ci) => {
      const td = document.createElement('td');
      td.className = 'cell';
      const ov = edits[t.name] && edits[t.name][i] && edits[t.name][i][ci];
      const v = ov ? ov.display : ((row.cells && row.cells[ci] !== undefined) ? row.cells[ci] : '');
      if (ov) td.className = 'cell edited';
      td.textContent = v;
      td.title = v + '  (double-click to edit)';
      td.addEventListener('dblclick', () => startCellEdit(t.name, i, ci, row, td));
      tr.appendChild(td);
    });
    tbody.appendChild(tr);
  });
  table.appendChild(tbody);
  wrap.appendChild(table);
  applyFilter();
}

function toggleRow(name, i, tr, btn) {
  if (removed[name][i]) {
    delete removed[name][i];
    tr.classList.remove('gone');
    btn.textContent = '✕';
    btn.title = 'Remove row from export';
  } else {
    removed[name][i] = true;
    tr.classList.add('gone');
    btn.textContent = '↺';
    btn.title = 'Restore row';
  }
  applyFilter();
  updateExpHint();
}

function applyFilter() {
  const t = resultData.tables[activeTab];
  const q = $('filter').value.trim().toLowerCase();
  const rows = $('gridwrap').querySelectorAll('tbody tr');
  let shown = 0, rem = 0;
  rows.forEach((tr) => {
    const i = +tr.dataset.i;
    if (isRemoved(t.name, i)) rem++;
    let match = true;
    if (q) {
      const cells = t.rows[i].cells || [];
      match = cells.some((v) => String(v).toLowerCase().indexOf(q) >= 0);
    }
    tr.style.display = match ? '' : 'none';
    if (match) shown++;
  });
  $('fcount').textContent = 'showing ' + fmt(shown) + ' of ' + fmt(t.rows.length) + ' (' + fmt(rem) + ' removed)';
}
$('filter').addEventListener('input', applyFilter);

/* startCellEdit makes a single cell editable in place. */
function startCellEdit(name, i, ci, row, td) {
  const cur = td.textContent.replace(/\s+\(double-click to edit\)$/, '');
  const inp = document.createElement('input');
  inp.className = 'celledit';
  inp.value = cur;
  td.textContent = '';
  td.appendChild(inp);
  inp.focus();
  inp.select();
  let done = false;
  function commit() {
    if (done) return;
    done = true;
    const nv = inp.value;
    const quote = origQuoted(row.raw, ci);
    if (!edits[name][i]) edits[name][i] = {};
    edits[name][i][ci] = { token: sqlLiteral(nv, quote), display: nv };
    renderGrid();
    updateExpHint();
  }
  inp.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') { e.preventDefault(); commit(); }
    else if (e.key === 'Escape') { done = true; renderGrid(); }
  });
  inp.addEventListener('blur', commit);
}

/* visibleRowIndices returns the row indices currently shown by the filter. */
function visibleRowIndices() {
  const idx = [];
  $('gridwrap').querySelectorAll('tbody tr').forEach((tr) => {
    if (tr.style.display !== 'none') idx.push(+tr.dataset.i);
  });
  return idx;
}

$('bapply').addEventListener('click', () => {
  if (!resultData) return;
  const t = resultData.tables[activeTab];
  const ci = $('bcol').value;
  if (ci === '' || ci === null) return;
  const token = sqlLiteral($('bval').value, $('bquote').checked);
  const disp = $('bval').value;
  const targets = $('bscope').value === 'filtered'
    ? visibleRowIndices()
    : (t.rows || []).map((_, i) => i);
  targets.forEach((i) => {
    if (!edits[t.name][i]) edits[t.name][i] = {};
    edits[t.name][i][ci] = { token: token, display: disp };
  });
  renderGrid();
  updateExpHint();
});
$('bclear').addEventListener('click', () => {
  if (!resultData) return;
  const t = resultData.tables[activeTab];
  edits[t.name] = {};
  renderGrid();
  updateExpHint();
});

function keptCount() {
  let n = 0;
  (resultData.tables || []).forEach((t) => {
    (t.rows || []).forEach((r, i) => { if (!isRemoved(t.name, i)) n++; });
  });
  return n;
}
function editedCells() {
  let n = 0;
  Object.keys(edits).forEach((k) => {
    const t = edits[k] || {};
    Object.keys(t).forEach((ri) => { n += Object.keys(t[ri] || {}).length; });
  });
  return n;
}
function updateExpHint() {
  if (!resultData) return;
  const verb = insertMode === 'ignore' ? 'INSERT IGNORE' : insertMode === 'replace' ? 'REPLACE' : 'INSERT';
  const ec = editedCells();
  $('exphint').textContent = fmt(keptCount()) + ' kept row(s) · ' +
    (format === 'insert' ? verb + ' statements' : 'raw rows') +
    (ec > 0 ? (' · ' + fmt(ec) + ' cell edit(s)') : '');
}

function buildSQL() {
  const verb = insertMode === 'ignore' ? 'INSERT IGNORE INTO' : insertMode === 'replace' ? 'REPLACE INTO' : 'INSERT INTO';
  const out = [];
  const isInsert = (format === 'insert');
  if (isInsert) {
    out.push('SET @OLD_FK=@@FOREIGN_KEY_CHECKS; SET FOREIGN_KEY_CHECKS=0;');
    out.push('');
  }
  (resultData.tables || []).forEach((t) => {
    const kept = [];
    (t.rows || []).forEach((r, i) => { if (!isRemoved(t.name, i)) kept.push({ r: r, i: i }); });
    out.push('-- ' + t.name + ' (' + kept.length + ' rows)');
    kept.forEach((k) => {
      const raw = rebuildRaw(t.name, k.i, k.r.raw);
      if (isInsert) out.push(verb + ' `' + t.name + '` VALUES (' + raw + ');');
      else out.push('(' + raw + ')');
    });
    out.push('');
  });
  if (isInsert) out.push('SET FOREIGN_KEY_CHECKS=@OLD_FK;');
  return out.join('\n');
}

$('export').addEventListener('click', () => {
  if (!resultData) return;
  const blob = new Blob([buildSQL()], { type: 'application/sql' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'restore_curated.sql';
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
});

let es;
function listen() {
  if (es) es.close();
  es = new EventSource('/api/events');
  es.onmessage = (ev) => { try { apply(JSON.parse(ev.data)); } catch (_) { /* ignore */ } };
  es.onerror = () => { /* auto-reconnects */ };
}

$('run').addEventListener('click', () => {
  const run = $('run');
  run.disabled = true;
  run.classList.add('busy');
  run.querySelector('.run-label').textContent = 'Running…';
  $('dl').classList.add('hidden');
  $('cov').classList.add('hidden');
  $('curate').classList.add('hidden');
  $('wrap').classList.remove('wide');
  resultData = null;
  $('status').className = 'status';
  $('status').textContent = 'Starting…';
  const fname = ($('outfile').value.trim() || 'restore_reservations.sql');
  const odir = $('outdir').value.trim();
  const outPath = odir ? (odir.replace(/\/+$/, '') + '/' + fname) : fname;
  const body = {
    DumpPath: $('dump').value.trim(),
    Query: $('query').value,
    Format: format,
    InsertMode: insertMode,
    BatchSize: 0,
    Related: $('related').checked,
    OutPath: outPath,
  };
  fetch('/api/run', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
    .then((r) => { if (!r.ok) return r.text().then((t) => { throw new Error(t); }); })
    .then(() => listen())
    .catch((e) => {
      $('status').className = 'status err';
      $('status').textContent = String(e.message || e);
      finishRun('Extract matching rows');
    });
});

listen();
