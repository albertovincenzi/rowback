package app

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Rowback - partial restore</title>
<link rel="icon" type="image/svg+xml" href="data:image/svg+xml,%3Csvg%20xmlns%3D%27http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%27%20viewBox%3D%270%200%2048%2048%27%3E%3Crect%20width%3D%2748%27%20height%3D%2748%27%20rx%3D%2712%27%20fill%3D%27%2310161f%27%2F%3E%3Cpath%20d%3D%27M24%209%20A15%2015%200%201%200%2039%2024%27%20fill%3D%27none%27%20stroke%3D%27%233fb950%27%20stroke-width%3D%273.4%27%20stroke-linecap%3D%27round%27%2F%3E%3Cpath%20d%3D%27M24%204.6%20L18.6%209%20L24%2013.4%20Z%27%20fill%3D%27%233fb950%27%2F%3E%3Crect%20x%3D%2715%27%20y%3D%2719%27%20width%3D%2718%27%20height%3D%273.1%27%20rx%3D%271.55%27%20fill%3D%27%234f9dff%27%2F%3E%3Crect%20x%3D%2715%27%20y%3D%2723.9%27%20width%3D%2718%27%20height%3D%273.1%27%20rx%3D%271.55%27%20fill%3D%27%23e7edf3%27%2F%3E%3Crect%20x%3D%2715%27%20y%3D%2728.8%27%20width%3D%2712.5%27%20height%3D%273.1%27%20rx%3D%271.55%27%20fill%3D%27%233fb950%27%2F%3E%3C%2Fsvg%3E">
<style>
  :root{
    --bg:#0a0d12; --panel:#141a23; --panel-2:#1a2230; --panel-3:#0f1620; --line:#26303d; --line-2:#313d4d;
    --text:#eef3f8; --muted:#8c98a8; --faint:#5d6776;
    --accent:#4f9dff; --accent-2:#1f6feb; --accent-3:#7ad0ff;
    --ok:#3fb950; --ok-2:#2ea043; --warn:#d29922; --err:#f85149;
    --radius:16px; --radius-sm:10px;
    --mono:ui-monospace,SFMono-Regular,Menlo,monospace;
    --sans:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;
    --shadow:0 1px 0 rgba(255,255,255,.05) inset, 0 24px 50px -30px rgba(0,0,0,.9);
    --ring:0 0 0 3px rgba(79,157,255,.28);
    --ease:cubic-bezier(.16,1,.3,1);
  }
  *{box-sizing:border-box}
  html{scroll-behavior:smooth}
  body{margin:0;font:15px/1.55 var(--sans);color:var(--text);background:var(--bg);min-height:100vh;
       -webkit-font-smoothing:antialiased;text-rendering:optimizeLegibility}
  body::before{content:"";position:fixed;inset:0;z-index:-2;pointer-events:none;
       background:
         radial-gradient(1000px 540px at 82% -10%,rgba(31,111,235,.20),transparent 62%),
         radial-gradient(760px 520px at 6% 2%,rgba(63,185,80,.10),transparent 58%),
         linear-gradient(180deg,#0c1019,#0a0d12 42%)}
  body::after{content:"";position:fixed;inset:0;z-index:-1;pointer-events:none;opacity:.04;
       background-image:url("data:image/svg+xml,%3Csvg%20xmlns%3D%27http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%27%20width%3D%27170%27%20height%3D%27170%27%3E%3Cfilter%20id%3D%27n%27%3E%3CfeTurbulence%20type%3D%27fractalNoise%27%20baseFrequency%3D%270.9%27%20numOctaves%3D%272%27%20stitchTiles%3D%27stitch%27%2F%3E%3C%2Ffilter%3E%3Crect%20width%3D%27170%27%20height%3D%27170%27%20filter%3D%27url%28%23n%29%27%2F%3E%3C%2Fsvg%3E");background-size:170px 170px}
  ::selection{background:rgba(79,157,255,.32);color:#fff}
  code{font-family:var(--mono);background:rgba(255,255,255,.05);border:1px solid var(--line);
       padding:1px 6px;border-radius:6px;font-size:.85em;color:var(--accent-3)}
  a{color:var(--accent-3)}

  .wrap{max-width:820px;margin:0 auto;padding:0 20px 96px}
  .wrap.wide{max-width:1160px}

  /* HERO */
  .hero{position:relative;margin:0 -20px 32px;padding:62px 20px 40px;overflow:hidden;border-bottom:1px solid var(--line)}
  .hero .aurora{position:absolute;left:50%;top:-300px;width:940px;height:640px;margin-left:-470px;z-index:-1;
       border-radius:46%;filter:blur(72px);opacity:.5;
       background:conic-gradient(from 120deg,#1f6feb,#3fb950,#7ad0ff,#1f6feb);
       animation:spin 28s linear infinite;transform-origin:50% 50%}
  @keyframes spin{to{transform:rotate(360deg)}}
  .hero-inner{max-width:780px;margin:0 auto;position:relative}
  .brand{display:flex;align-items:center;gap:14px}
  .brand .mark{height:46px;width:46px;flex:0 0 auto;filter:drop-shadow(0 8px 22px rgba(63,185,80,.28))}
  .wordmark{font:800 30px/1 var(--sans);letter-spacing:-.035em}
  .wordmark .b{color:var(--ok)}
  .badge{margin-left:auto;font:600 10.5px var(--mono);letter-spacing:.12em;text-transform:uppercase;color:var(--muted);
       border:1px solid var(--line);padding:7px 11px;border-radius:999px;background:rgba(255,255,255,.025)}
  .tagline{margin:24px 0 0;font:700 clamp(22px,2.2vw,30px)/1.15 var(--sans);letter-spacing:-.02em;max-width:600px}
  .subtag{margin:12px 0 0;color:var(--muted);font-size:15px;max-width:600px}

  /* STEP CARDS */
  .step{position:relative;margin-bottom:18px;padding:24px 24px 22px;border-radius:var(--radius);
       background:linear-gradient(180deg,rgba(26,34,48,.72),rgba(15,22,32,.72));
       border:1px solid var(--line);box-shadow:var(--shadow);backdrop-filter:blur(10px);-webkit-backdrop-filter:blur(10px)}
  .step::before{content:attr(data-n);position:absolute;top:20px;right:22px;font:700 12px var(--mono);color:var(--muted);
       border:1px solid var(--line);background:rgba(255,255,255,.02);border-radius:999px;width:28px;height:28px;
       display:grid;place-items:center}
  .step h2{font:700 12px/1 var(--mono);margin:0 0 16px;text-transform:uppercase;letter-spacing:.14em;color:var(--accent);
       display:flex;align-items:center;gap:9px}
  .step h2::before{content:"";width:7px;height:7px;border-radius:2px;background:var(--accent);box-shadow:0 0 10px var(--accent)}

  label{display:block;font-size:12.5px;color:var(--muted);margin:14px 0 6px;letter-spacing:.01em}
  input,textarea,select{width:100%;background:var(--panel-3);border:1px solid var(--line);color:var(--text);
       border-radius:var(--radius-sm);padding:11px 13px;font-size:13.5px;font-family:var(--mono);resize:vertical;
       transition:border-color .18s var(--ease),box-shadow .18s var(--ease),background .18s var(--ease)}
  input::placeholder,textarea::placeholder{color:var(--faint)}
  select{appearance:none;cursor:pointer;
       background-image:linear-gradient(45deg,transparent 50%,var(--muted) 50%),linear-gradient(135deg,var(--muted) 50%,transparent 50%);
       background-position:calc(100% - 18px) 17px,calc(100% - 13px) 17px;background-size:5px 5px,5px 5px;background-repeat:no-repeat}
  textarea{min-height:128px;line-height:1.6;white-space:pre}
  input:hover,textarea:hover,select:hover{border-color:var(--line-2)}
  input:focus,textarea:focus,select:focus{outline:none;border-color:var(--accent);box-shadow:var(--ring);background:var(--bg)}
  .hint{font-size:12px;color:var(--muted);margin-top:7px;line-height:1.5}
  .row2{display:flex;gap:16px;flex-wrap:wrap}
  .row2>div{flex:1;min-width:200px}

  .seg{display:inline-flex;border:1px solid var(--line);border-radius:var(--radius-sm);overflow:hidden;margin-top:6px;background:var(--panel-3)}
  .seg button{background:transparent;color:var(--muted);border:0;padding:10px 16px;cursor:pointer;font:600 13px var(--sans);
       transition:color .18s,background .18s}
  .seg button:hover{color:var(--text)}
  .seg button.on{background:linear-gradient(180deg,var(--accent),var(--accent-2));color:#fff;box-shadow:0 6px 16px -8px var(--accent)}

  .check{display:flex;align-items:flex-start;gap:11px;margin-top:18px;cursor:pointer}
  .check input{width:18px;height:18px;flex:0 0 auto;margin-top:1px;accent-color:var(--accent);cursor:pointer}
  .check span{font-size:13.5px;color:var(--text)}
  .check span small{display:block;color:var(--muted);font-size:12px;margin-top:3px}

  button.run{appearance:none;border:0;cursor:pointer;font:700 15px/1 var(--sans);border-radius:12px;padding:14px 20px;
       background:linear-gradient(180deg,var(--accent),var(--accent-2));color:#fff;width:100%;letter-spacing:.01em;
       box-shadow:0 12px 28px -12px var(--accent),0 1px 0 rgba(255,255,255,.18) inset;
       transition:transform .12s var(--ease),box-shadow .2s var(--ease),filter .2s}
  button.run:hover{filter:brightness(1.06);box-shadow:0 16px 34px -12px var(--accent),0 1px 0 rgba(255,255,255,.22) inset}
  button.run:active{transform:translateY(1px)}
  button.run:disabled{opacity:.5;cursor:not-allowed;filter:none;box-shadow:none}

  .bar{height:10px;background:var(--panel-3);border:1px solid var(--line);border-radius:999px;overflow:hidden;margin:16px 0}
  .bar>i{display:block;height:100%;width:0;border-radius:999px;position:relative;overflow:hidden;
       background:linear-gradient(90deg,var(--accent),var(--accent-3));transition:width .35s var(--ease)}
  .bar>i::after{content:"";position:absolute;inset:0;transform:translateX(-100%);
       background:linear-gradient(90deg,transparent,rgba(255,255,255,.5),transparent);animation:shim 1.5s ease-in-out infinite}
  @keyframes shim{to{transform:translateX(100%)}}

  .grid{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}
  .stat{background:linear-gradient(180deg,var(--panel-2),var(--panel-3));border:1px solid var(--line);border-radius:12px;padding:13px 14px}
  .stat .k{font-size:10.5px;color:var(--muted);text-transform:uppercase;letter-spacing:.08em}
  .stat .v{font:700 22px var(--mono);margin-top:5px;letter-spacing:-.01em}
  .stat.match{border-color:rgba(63,185,80,.42);box-shadow:0 0 0 1px rgba(63,185,80,.12),0 0 30px -10px rgba(63,185,80,.55)}
  .stat.match .v{color:var(--ok);animation:matchpulse 2.6s ease-in-out infinite}
  @keyframes matchpulse{0%,100%{text-shadow:0 0 0 rgba(63,185,80,0)}50%{text-shadow:0 0 18px rgba(63,185,80,.6)}}

  .status{font-size:13px;color:var(--muted);margin:12px 0 0;min-height:18px}
  .status.err{color:var(--err)} .status.ok{color:var(--ok)}

  .dl{display:inline-block;margin-top:16px;background:linear-gradient(180deg,var(--ok),var(--ok-2));color:#04210d;text-decoration:none;
       font-weight:700;padding:11px 18px;border-radius:12px;box-shadow:0 12px 26px -14px var(--ok);
       transition:transform .12s var(--ease),filter .2s}
  .dl:hover{filter:brightness(1.06)} .dl:active{transform:translateY(1px)}
  .muted{color:var(--muted)} .hidden{display:none}

  .cov{margin-top:16px;display:flex;flex-direction:column;gap:6px;padding:14px 16px;border:1px solid var(--line);
       border-radius:12px;background:var(--panel-3)}
  .cov .c{font:13px var(--mono);display:flex;gap:8px;align-items:baseline}
  .cov .c b{font-weight:700}
  .cov .ok{color:var(--ok)} .cov .warn{color:var(--warn)} .cov .bad{color:var(--err)}
  .cov .miss{color:var(--muted)}
  .cov .rel{color:var(--accent);font:13px var(--mono);margin-top:2px}

  /* PREVIEW / CURATE */
  #curate:not(.hidden){animation:rise .55s var(--ease) both}
  @keyframes rise{from{opacity:0;transform:translateY(16px)}to{opacity:1;transform:none}}
  .tabs{display:flex;gap:6px;flex-wrap:wrap;margin-bottom:16px;border-bottom:1px solid var(--line)}
  .tab{background:transparent;border:1px solid transparent;border-bottom:0;color:var(--muted);
       padding:9px 14px;cursor:pointer;font:600 13px var(--mono);border-radius:10px 10px 0 0;position:relative;top:1px;
       transition:color .15s,background .15s}
  .tab:hover{color:var(--text);background:rgba(255,255,255,.03)}
  .tab.on{background:var(--panel-2);border-color:var(--line);color:var(--accent)}
  .tab .rel{display:block;font-weight:400;font-size:11px;color:var(--muted);margin-top:1px;max-width:240px;
       overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
  .filterbar{display:flex;gap:12px;align-items:center;margin-bottom:12px;flex-wrap:wrap}
  .filterbar input{flex:1;min-width:200px;margin:0}
  .filterbar .count{font:12px var(--mono);color:var(--muted);white-space:nowrap}
  .capnote{font-size:12px;color:var(--warn);margin-bottom:10px}
  .bulkbar{display:flex;gap:8px;align-items:center;margin-bottom:12px;flex-wrap:wrap;
       background:var(--panel-3);border:1px solid var(--line);border-radius:12px;padding:10px 12px}
  .bulkbar .bk{font:700 10.5px var(--mono);color:var(--accent);text-transform:uppercase;letter-spacing:.08em}
  .bulkbar select{width:auto;min-width:120px;margin:0;padding:7px 26px 7px 10px}
  .bulkbar input#bval{flex:1;min-width:140px;margin:0;padding:7px 10px}
  .bulkbar .eq{color:var(--muted);font:700 13px var(--mono)}
  .bulkbar .bq{display:flex;align-items:center;gap:5px;margin:0;font-size:12px;color:var(--muted);white-space:nowrap}
  .bulkbar .bq input{width:15px;height:15px;margin:0;accent-color:var(--accent)}
  .bbtn{appearance:none;border:1px solid var(--accent-2);background:linear-gradient(180deg,var(--accent),var(--accent-2));color:#fff;
       border-radius:9px;padding:7px 13px;cursor:pointer;font:600 12px var(--sans);white-space:nowrap;
       transition:transform .12s var(--ease),filter .18s}
  .bbtn.ghost{background:transparent;border-color:var(--line);color:var(--muted)}
  .bbtn:hover{filter:brightness(1.1)} .bbtn:active{transform:translateY(1px)}

  table.dt td.edited{color:#ffd27a;background:rgba(210,153,34,.14);box-shadow:inset 0 0 0 1px rgba(210,153,34,.35)}
  table.dt td.cell{cursor:text}
  input.celledit{width:100%;min-width:80px;margin:0;padding:3px 6px;font:12.5px var(--mono);
       background:#0a0f15;border:1px solid var(--accent);border-radius:6px;color:var(--text)}
  .gridwrap{overflow:auto;border:1px solid var(--line);border-radius:12px;max-height:480px;background:var(--panel-3)}
  table.dt{border-collapse:collapse;width:100%;font:12.5px var(--mono)}
  table.dt th,table.dt td{padding:8px 11px;text-align:left;border-bottom:1px solid var(--line);white-space:nowrap}
  table.dt thead th{position:sticky;top:0;background:rgba(16,22,32,.96);-webkit-backdrop-filter:blur(6px);backdrop-filter:blur(6px);
       color:var(--accent);z-index:2;text-transform:uppercase;letter-spacing:.06em;font-size:11px;font-weight:700;
       border-bottom:1px solid var(--line-2)}
  table.dt thead th.x{width:34px}
  table.dt tbody tr{transition:background .12s}
  table.dt tbody tr:nth-child(even){background:rgba(255,255,255,.02)}
  table.dt tbody tr:hover{background:rgba(79,157,255,.09)}
  table.dt tbody tr.gone td{color:var(--muted);text-decoration:line-through;opacity:.5}
  table.dt td.cell{max-width:320px;overflow:hidden;text-overflow:ellipsis}
  .xbtn{appearance:none;border:1px solid var(--line);background:var(--panel-2);color:var(--muted);
       width:23px;height:23px;border-radius:7px;cursor:pointer;font:700 12px/1 var(--mono);padding:0;
       transition:border-color .15s,color .15s}
  .xbtn:hover{border-color:var(--err);color:var(--err)}
  tr.gone .xbtn{border-color:var(--ok);color:var(--ok)}
  tr.gone .xbtn:hover{border-color:var(--accent);color:var(--accent)}
  .exprow{display:flex;gap:12px;align-items:center;margin-top:18px;flex-wrap:wrap}
  button.export{appearance:none;border:0;cursor:pointer;font:700 14px/1 var(--sans);border-radius:12px;padding:13px 20px;
       background:linear-gradient(180deg,var(--ok),var(--ok-2));color:#04210d;box-shadow:0 12px 26px -14px var(--ok);
       transition:transform .12s var(--ease),filter .18s}
  button.export:hover{filter:brightness(1.06)} button.export:active{transform:translateY(1px)}
  .dl.sec{background:var(--panel-2);color:var(--text);border:1px solid var(--line);margin-top:0;box-shadow:none}

  :focus-visible{outline:2px solid var(--accent);outline-offset:2px}
  @media (max-width:640px){.grid{grid-template-columns:repeat(2,1fr)}}
  @media (prefers-reduced-motion:reduce){
    *,*::before,*::after{animation:none!important;transition:none!important;scroll-behavior:auto!important}
  }
</style>
</head>
<body>
<div class="wrap" id="wrap">
  <header class="hero">
    <div class="aurora" aria-hidden="true"></div>
    <div class="hero-inner">
      <div class="brand">
        <svg class="mark" viewBox="0 0 48 48" role="img" aria-label="Rowback logo">
          <defs>
            <linearGradient id="rbBar" x1="0" y1="0" x2="1" y2="0">
              <stop offset="0" stop-color="#7ad0ff"/><stop offset="1" stop-color="#4f9dff"/>
            </linearGradient>
          </defs>
          <rect x="1.5" y="1.5" width="45" height="45" rx="13" fill="#10161f" stroke="#26303d"/>
          <path d="M24 9 A15 15 0 1 0 39 24" fill="none" stroke="#3fb950" stroke-width="3.4" stroke-linecap="round"/>
          <path d="M24 4.6 L18.6 9 L24 13.4 Z" fill="#3fb950"/>
          <rect x="15" y="19" width="18" height="3.1" rx="1.55" fill="url(#rbBar)"/>
          <rect x="15" y="23.9" width="18" height="3.1" rx="1.55" fill="#e7edf3"/>
          <rect x="15" y="28.8" width="12.5" height="3.1" rx="1.55" fill="#3fb950"/>
        </svg>
        <span class="wordmark">row<span class="b">back</span></span>
        <span class="badge">partial restore</span>
      </div>
      <h1 class="tagline">Surgically restore deleted rows from a massive SQL dump.</h1>
      <p class="subtag">Paste the query whose rows you want back. Rowback runs its <code>WHERE</code> against the dump and emits restore SQL.</p>
    </div>
  </header>

  <section class="step" data-n="1">
    <h2>Source dump</h2>
    <label for="dump">Path to mysqldump <code>.sql</code></label>
    <input id="dump" value="__DUMP__" spellcheck="false">
    <div class="hint">A one-time index is cached in <code>~/.rowback/indexes</code> (associated to this dump), so later queries seek straight to the tables they need.</div>
    <div class="row2">
      <div>
        <label for="outdir">Restore folder</label>
        <input id="outdir" placeholder="(defaults to the dump's folder)" spellcheck="false">
      </div>
      <div>
        <label for="outfile">Restore file name</label>
        <input id="outfile" value="restore_reservations.sql" spellcheck="false">
      </div>
    </div>
  </section>

  <section class="step" data-n="2">
    <h2>Query</h2>
    <label for="query">SQL <span class="muted">— DELETE / SELECT; the WHERE clause selects rows. Supports IN-lists, =, !=, and one IN (SELECT … WHERE …).</span></label>
    <textarea id="query" spellcheck="false">DELETE FROM orders
WHERE customer_id IN (SELECT id FROM customers WHERE region_id = 42)
  AND order_no IN ('A-1001','A-1002')</textarea>
    <div class="row2">
      <div>
        <label>Output format</label>
        <div class="seg" id="fmt">
          <button data-v="insert" class="on">INSERT statements</button>
          <button data-v="raw">Raw rows</button>
        </div>
      </div>
      <div id="imodeWrap">
        <label for="imode">Insert mode</label>
        <select id="imode">
          <option value="insert">INSERT</option>
          <option value="ignore">INSERT IGNORE</option>
          <option value="replace">REPLACE</option>
        </select>
      </div>
    </div>
    <label class="check" for="related">
      <input type="checkbox" id="related" checked>
      <span>Also extract linked rows
        <small>Walk children (transitively) that reference the matched rows and include them.</small>
      </span>
    </label>
  </section>

  <section class="step" data-n="3">
    <h2>Run &amp; restore</h2>
    <button class="run" id="run">Extract matching rows</button>
    <div class="bar"><i id="barfill"></i></div>
    <div class="grid">
      <div class="stat"><div class="k">Read</div><div class="v" id="s-pct">0%</div></div>
      <div class="stat"><div class="k">Sub resolved</div><div class="v" id="s-sub">0</div></div>
      <div class="stat"><div class="k">Rows scanned</div><div class="v" id="s-main">0</div></div>
      <div class="stat match"><div class="k">Matched</div><div class="v" id="s-match">0</div></div>
    </div>
    <p class="status" id="status"></p>
    <div class="cov hidden" id="cov"></div>
    <a class="dl hidden" id="dl" href="/api/download">↓ Download full restore SQL</a>
  </section>

  <section class="step hidden" data-n="4" id="curate">
    <h2>Preview &amp; curate</h2>
    <div class="tabs" id="tabs"></div>
    <div class="capnote hidden" id="capnote"></div>
    <div class="filterbar">
      <input id="filter" placeholder="Filter rows in this tab…" spellcheck="false">
      <span class="count" id="fcount"></span>
    </div>
    <div class="bulkbar">
      <span class="bk">Bulk edit</span>
      <select id="bcol"></select>
      <span class="eq">=</span>
      <input id="bval" placeholder="new value" spellcheck="false">
      <label class="bq"><input type="checkbox" id="bquote" checked> quote as text</label>
      <select id="bscope" title="Which rows the bulk edit affects">
        <option value="all">all rows</option>
        <option value="filtered">filtered rows only</option>
      </select>
      <button class="bbtn" id="bapply">Apply</button>
      <button class="bbtn ghost" id="bclear">Clear edits</button>
    </div>
    <div class="hint" style="margin:-6px 0 12px">Tip: double-click any cell to edit it individually.</div>
    <div class="gridwrap" id="gridwrap"></div>
    <div class="exprow">
      <button class="export" id="export">↓ Export kept rows</button>
      <a class="dl sec" id="dl2" href="/api/download">↓ Download full restore SQL (server file)</a>
      <span class="muted" id="exphint" style="font-size:12px"></span>
    </div>
  </section>
</div>

<script>
var $=function(id){return document.getElementById(id);};
` + "var BT='`';" + `
var format='insert';
var insertMode='insert';
var resultData=null;
var activeTab=0;
var removed={};      /* tableName -> {rowIndex:true} */
var edits={};        /* tableName -> {rowIndex:{colIndex:{token,display}}} */

/* splitTuple splits a raw VALUES tuple into its top-level field tokens,
   respecting single-quoted strings, backslash escapes, and nested parens —
   mirrors the server-side splitFields so edits splice in faithfully. */
function splitTuple(raw){
  var out=[],inStr=false,esc=false,depth=0,start=0;
  for(var i=0;i<raw.length;i++){
    var c=raw[i];
    if(inStr){
      if(esc){esc=false;}
      else if(c==='\\'){esc=true;}
      else if(c==="'"){inStr=false;}
      continue;
    }
    if(c==="'"){inStr=true;}
    else if(c==='('){depth++;}
    else if(c===')'){depth--;}
    else if(c===','&&depth===0){out.push(raw.slice(start,i));start=i+1;}
  }
  out.push(raw.slice(start));
  return out;
}

/* rebuildRaw applies a row's per-column edits to its raw tuple. */
function rebuildRaw(tname,i,raw){
  var ov=edits[tname]&&edits[tname][i];
  if(!ov)return raw;
  var keys=Object.keys(ov);
  if(!keys.length)return raw;
  var toks=splitTuple(raw);
  keys.forEach(function(ci){if(ci<toks.length)toks[ci]=ov[ci].token;});
  return toks.join(',');
}
/* origQuoted reports whether column ci of a row was originally a quoted string. */
function origQuoted(raw,ci){
  var toks=splitTuple(raw);
  return ci<toks.length && toks[ci].charAt(0)==="'";
}

/* sqlLiteral turns a user value into a SQL token (quoted+escaped, or verbatim). */
function sqlLiteral(val,quote){
  if(!quote)return val;
  return "'"+val.replace(/\\/g,'\\\\').replace(/'/g,"\\'")+"'";
}

document.querySelectorAll('#fmt button').forEach(function(b){
  b.addEventListener('click',function(){
    document.querySelectorAll('#fmt button').forEach(function(x){x.classList.remove('on');});
    b.classList.add('on');format=b.dataset.v;
    $('imodeWrap').classList.toggle('hidden',format!=='insert');
    if(resultData){updateExpHint();}
  });
});
$('imode').addEventListener('change',function(){insertMode=$('imode').value;if(resultData){updateExpHint();}});

function fmt(n){return (n||0).toLocaleString();}

function renderCoverage(d){
  var box=$('cov');box.innerHTML='';
  var has=false;
  if(d.Coverage&&d.Coverage.length){
    has=true;
    d.Coverage.forEach(function(c){
      var row=document.createElement('div');row.className='c';
      var label=document.createElement('b');label.textContent=c.Column+':';
      row.appendChild(label);
      var val=document.createElement('span');
      if(c.Found>=c.Total){
        val.className='ok';val.textContent=c.Found+'/'+c.Total+' found';
      }else{
        val.className=(c.Found===0?'bad':'warn');
        val.textContent=c.Found+'/'+c.Total;
        row.appendChild(val);
        var miss=document.createElement('span');miss.className='miss';
        miss.textContent='— missing: '+((c.Missing||[]).join(', '));
        row.appendChild(miss);
        box.appendChild(row);
        return;
      }
      row.appendChild(val);
      box.appendChild(row);
    });
  }
  if(d.RelatedRows>0){
    has=true;
    var rel=document.createElement('div');rel.className='rel';
    rel.textContent='+ '+fmt(d.RelatedRows)+' related row(s) across '+fmt(d.RelatedTbls)+' table(s)';
    box.appendChild(rel);
  }
  box.classList.toggle('hidden',!has);
}

function apply(d){
  $('s-pct').textContent=(d.pct||0).toFixed(1)+'%';
  $('barfill').style.width=Math.min(100,d.pct||0)+'%';
  $('s-sub').textContent=fmt(d.SubResolved);
  $('s-main').textContent=fmt(d.MainScanned);
  $('s-match').textContent=fmt(d.Matched);
  var st=$('status');
  if(d.err){
    st.className='status err';st.textContent=d.err;
    $('run').disabled=false;$('run').textContent='Run again';
  }else if(d.Done){
    st.className='status ok';
    st.textContent=d.Message||('Done — '+fmt(d.Matched)+' row(s).');
    if(d.Matched>0)$('dl').classList.remove('hidden');
    renderCoverage(d);
    $('run').disabled=false;$('run').textContent='Run again';
    loadResult();
  }else{
    st.className='status';
    var txt;
    if(d.Phase==='indexing'){txt='Building one-time index… '+(d.pct||0).toFixed(1)+'%';}
    else if(d.Phase==='schema'){txt='Reading schema (foreign keys)…';}
    else if(d.Phase==='related'){
      txt=(d.Message||'Scanning related tables')+' … '+(d.pct||0).toFixed(1)+'%'+
          (d.RelatedRows>0?(' — '+fmt(d.RelatedRows)+' related row(s) so far'):'');
    }
    else{txt='Scanning ('+(d.Phase||'')+') … '+(d.pct||0).toFixed(1)+'%';}
    st.textContent=txt;
  }
}

function loadResult(){
  fetch('/api/result').then(function(r){
    if(!r.ok)throw new Error('no result');
    return r.json();
  }).then(function(data){
    resultData=data;
    removed={};edits={};
    (data.tables||[]).forEach(function(t){removed[t.name]={};edits[t.name]={};});
    activeTab=0;
    $('curate').classList.remove('hidden');
    $('wrap').classList.add('wide');
    updateExpHint();
    renderTabs();
    renderGrid();
  }).catch(function(){/* no preview available */});
}

function renderTabs(){
  var tabs=$('tabs');tabs.innerHTML='';
  (resultData.tables||[]).forEach(function(t,i){
    var b=document.createElement('button');
    b.className='tab'+(i===activeTab?' on':'');
    var name=document.createElement('span');
    name.textContent=t.name+' ('+fmt(t.total)+')';
    b.appendChild(name);
    if(t.relation){
      b.title=t.relation;
      var rel=document.createElement('span');rel.className='rel';rel.textContent=t.relation;
      b.appendChild(rel);
    }
    b.addEventListener('click',function(){activeTab=i;$('filter').value='';renderTabs();renderGrid();});
    tabs.appendChild(b);
  });
}

function isRemoved(name,i){return !!(removed[name]&&removed[name][i]);}

function renderGrid(){
  var t=resultData.tables[activeTab];
  var wrap=$('gridwrap');wrap.innerHTML='';
  // populate the bulk-edit column selector for this table
  var bcol=$('bcol');bcol.innerHTML='';
  (t.columns||[]).forEach(function(c,ci){
    var o=document.createElement('option');o.value=ci;o.textContent=c;bcol.appendChild(o);
  });
  $('capnote').classList.toggle('hidden',!t.capped);
  if(t.capped){
    $('capnote').textContent='Only the first '+fmt(t.rows.length)+' of '+fmt(t.total)+' rows are shown for preview; export covers the shown rows.';
  }
  var table=document.createElement('table');table.className='dt';
  var thead=document.createElement('thead');
  var hr=document.createElement('tr');
  var hx=document.createElement('th');hx.className='x';hr.appendChild(hx);
  (t.columns||[]).forEach(function(c){
    var th=document.createElement('th');th.textContent=c;hr.appendChild(th);
  });
  thead.appendChild(hr);table.appendChild(thead);
  var tbody=document.createElement('tbody');
  (t.rows||[]).forEach(function(row,i){
    var tr=document.createElement('tr');tr.dataset.i=i;
    if(isRemoved(t.name,i))tr.className='gone';
    var tdx=document.createElement('td');
    var btn=document.createElement('button');btn.className='xbtn';
    btn.textContent=isRemoved(t.name,i)?'↺':'✕';
    btn.title=isRemoved(t.name,i)?'Restore row':'Remove row from export';
    btn.addEventListener('click',function(){toggleRow(t.name,i,tr,btn);});
    tdx.appendChild(btn);tr.appendChild(tdx);
    (t.columns||[]).forEach(function(c,ci){
      var td=document.createElement('td');td.className='cell';
      var ov=edits[t.name]&&edits[t.name][i]&&edits[t.name][i][ci];
      var v=ov?ov.display:((row.cells&&row.cells[ci]!==undefined)?row.cells[ci]:'');
      if(ov)td.className='cell edited';
      td.textContent=v;td.title=v+'  (double-click to edit)';
      td.addEventListener('dblclick',function(){startCellEdit(t.name,i,ci,row,td);});
      tr.appendChild(td);
    });
    tbody.appendChild(tr);
  });
  table.appendChild(tbody);wrap.appendChild(table);
  applyFilter();
}

function toggleRow(name,i,tr,btn){
  if(removed[name][i]){delete removed[name][i];tr.classList.remove('gone');btn.textContent='✕';btn.title='Remove row from export';}
  else{removed[name][i]=true;tr.classList.add('gone');btn.textContent='↺';btn.title='Restore row';}
  applyFilter();
  updateExpHint();
}

function applyFilter(){
  var t=resultData.tables[activeTab];
  var q=$('filter').value.trim().toLowerCase();
  var rows=$('gridwrap').querySelectorAll('tbody tr');
  var shown=0,rem=0;
  rows.forEach(function(tr){
    var i=+tr.dataset.i;
    if(isRemoved(t.name,i))rem++;
    var match=true;
    if(q){
      var cells=t.rows[i].cells||[];
      match=cells.some(function(v){return String(v).toLowerCase().indexOf(q)>=0;});
    }
    tr.style.display=match?'':'none';
    if(match)shown++;
  });
  $('fcount').textContent='showing '+fmt(shown)+' of '+fmt(t.rows.length)+' ('+fmt(rem)+' removed)';
}
$('filter').addEventListener('input',applyFilter);

/* startCellEdit makes a single cell editable in place. */
function startCellEdit(name,i,ci,row,td){
  var cur=td.textContent.replace(/\s+\(double-click to edit\)$/,'');
  var inp=document.createElement('input');inp.className='celledit';inp.value=cur;
  td.textContent='';td.appendChild(inp);inp.focus();inp.select();
  var done=false;
  function commit(){
    if(done)return;done=true;
    var nv=inp.value;
    var quote=origQuoted(row.raw,ci);
    if(!edits[name][i])edits[name][i]={};
    edits[name][i][ci]={token:sqlLiteral(nv,quote),display:nv};
    renderGrid();updateExpHint();
  }
  inp.addEventListener('keydown',function(e){
    if(e.key==='Enter'){e.preventDefault();commit();}
    else if(e.key==='Escape'){done=true;renderGrid();}
  });
  inp.addEventListener('blur',commit);
}

/* visibleRowIndices returns the row indices currently shown by the filter. */
function visibleRowIndices(){
  var idx=[];
  $('gridwrap').querySelectorAll('tbody tr').forEach(function(tr){
    if(tr.style.display!=='none')idx.push(+tr.dataset.i);
  });
  return idx;
}

$('bapply').addEventListener('click',function(){
  if(!resultData)return;
  var t=resultData.tables[activeTab];
  var ci=$('bcol').value;
  if(ci===''||ci===null)return;
  var token=sqlLiteral($('bval').value,$('bquote').checked);
  var disp=$('bval').value;
  var targets=$('bscope').value==='filtered'?visibleRowIndices():(t.rows||[]).map(function(_,i){return i;});
  targets.forEach(function(i){
    if(!edits[t.name][i])edits[t.name][i]={};
    edits[t.name][i][ci]={token:token,display:disp};
  });
  renderGrid();
  updateExpHint();
});
$('bclear').addEventListener('click',function(){
  if(!resultData)return;
  var t=resultData.tables[activeTab];
  edits[t.name]={};
  renderGrid();
  updateExpHint();
});

function keptCount(){
  var n=0;
  (resultData.tables||[]).forEach(function(t){
    (t.rows||[]).forEach(function(r,i){if(!isRemoved(t.name,i))n++;});
  });
  return n;
}
function editedCells(){
  var n=0;
  Object.keys(edits).forEach(function(k){
    var t=edits[k]||{};
    Object.keys(t).forEach(function(ri){n+=Object.keys(t[ri]||{}).length;});
  });
  return n;
}
function updateExpHint(){
  if(!resultData)return;
  var verb=insertMode==='ignore'?'INSERT IGNORE':insertMode==='replace'?'REPLACE':'INSERT';
  var ec=editedCells();
  $('exphint').textContent=fmt(keptCount())+' kept row(s) · '+(format==='insert'?verb+' statements':'raw rows')+
    (ec>0?(' · '+fmt(ec)+' cell edit(s)'):'');
}

function buildSQL(){
  var verb=insertMode==='ignore'?'INSERT IGNORE INTO':insertMode==='replace'?'REPLACE INTO':'INSERT INTO';
  var out=[];
  var isInsert=(format==='insert');
  if(isInsert){
    out.push('SET @OLD_FK=@@FOREIGN_KEY_CHECKS; SET FOREIGN_KEY_CHECKS=0;');
    out.push('');
  }
  (resultData.tables||[]).forEach(function(t){
    var kept=[];
    (t.rows||[]).forEach(function(r,i){if(!isRemoved(t.name,i))kept.push({r:r,i:i});});
    out.push('-- '+t.name+' ('+kept.length+' rows)');
    kept.forEach(function(k){
      var raw=rebuildRaw(t.name,k.i,k.r.raw);
      if(isInsert){out.push(verb+' '+BT+t.name+BT+' VALUES ('+raw+');');}
      else{out.push('('+raw+')');}
    });
    out.push('');
  });
  if(isInsert){out.push('SET FOREIGN_KEY_CHECKS=@OLD_FK;');}
  return out.join('\n');
}

$('export').addEventListener('click',function(){
  if(!resultData)return;
  var blob=new Blob([buildSQL()],{type:'application/sql'});
  var url=URL.createObjectURL(blob);
  var a=document.createElement('a');a.href=url;a.download='restore_curated.sql';
  document.body.appendChild(a);a.click();a.remove();
  setTimeout(function(){URL.revokeObjectURL(url);},1000);
});

var es;
function listen(){
  if(es)es.close();
  es=new EventSource('/api/events');
  es.onmessage=function(ev){try{apply(JSON.parse(ev.data));}catch(_){}};
  es.onerror=function(){};
}
$('run').addEventListener('click',function(){
  $('run').disabled=true;$('run').textContent='Running…';
  $('dl').classList.add('hidden');
  $('cov').classList.add('hidden');
  $('curate').classList.add('hidden');
  $('wrap').classList.remove('wide');
  resultData=null;
  $('status').className='status';$('status').textContent='Starting…';
  var fname=($('outfile').value.trim()||'restore_reservations.sql');
  var odir=$('outdir').value.trim();
  var outPath=odir?(odir.replace(/\/+$/,'')+'/'+fname):fname;
  var body={DumpPath:$('dump').value.trim(),Query:$('query').value,Format:format,
    InsertMode:insertMode,BatchSize:0,Related:$('related').checked,OutPath:outPath};
  fetch('/api/run',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})
    .then(function(r){if(!r.ok)return r.text().then(function(t){throw new Error(t);});})
    .then(function(){listen();})
    .catch(function(e){
      $('status').className='status err';$('status').textContent=String(e.message||e);
      $('run').disabled=false;$('run').textContent='Extract matching rows';
    });
});
listen();
</script>
</body>
</html>`
