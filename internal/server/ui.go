package server

import (
	"net/http"
)

// uiHTML is the self-contained dashboard for Fence.
// Served at GET /ui — no build step, no external files.
const uiHTML = `<!DOCTYPE html><html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Fence — Stockyard</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital,wght@0,400;0,700;1,400&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
<style>:root{
  --bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;
  --rust:#c45d2c;--rust-light:#e8753a;--rust-dark:#8b3d1a;
  --leather:#a0845c;--leather-light:#c4a87a;
  --cream:#f0e6d3;--cream-dim:#bfb5a3;--cream-muted:#7a7060;
  --gold:#d4a843;--green:#5ba86e;--red:#c0392b;
  --font-serif:'Libre Baskerville',Georgia,serif;
  --font-mono:'JetBrains Mono',monospace;
}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--font-serif);min-height:100vh;overflow-x:hidden}
a{color:var(--rust-light);text-decoration:none}a:hover{color:var(--gold)}
.hdr{background:var(--bg2);border-bottom:2px solid var(--rust-dark);padding:.9rem 1.8rem;display:flex;align-items:center;justify-content:space-between;gap:1rem}
.hdr-left{display:flex;align-items:center;gap:1rem}
.hdr-brand{font-family:var(--font-mono);font-size:.75rem;color:var(--leather);letter-spacing:3px;text-transform:uppercase}
.hdr-title{font-family:var(--font-mono);font-size:1.1rem;color:var(--cream);letter-spacing:1px}
.badge{font-family:var(--font-mono);font-size:.6rem;padding:.2rem .6rem;letter-spacing:1px;text-transform:uppercase;border:1px solid}
.badge-free{color:var(--green);border-color:var(--green)}
.badge-pro{color:var(--gold);border-color:var(--gold)}
.badge-ok{color:var(--green);border-color:var(--green)}
.badge-err{color:var(--red);border-color:var(--red)}
.main{max-width:1000px;margin:0 auto;padding:2rem 1.5rem}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:1rem;margin-bottom:2rem}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:1.2rem 1.5rem}
.card-val{font-family:var(--font-mono);font-size:1.8rem;font-weight:700;color:var(--cream);display:block}
.card-lbl{font-family:var(--font-mono);font-size:.62rem;letter-spacing:2px;text-transform:uppercase;color:var(--leather);margin-top:.3rem}
.section{margin-bottom:2.5rem}
.section-title{font-family:var(--font-mono);font-size:.68rem;letter-spacing:3px;text-transform:uppercase;color:var(--rust-light);margin-bottom:.8rem;padding-bottom:.5rem;border-bottom:1px solid var(--bg3)}
table{width:100%;border-collapse:collapse;font-family:var(--font-mono);font-size:.78rem}
th{background:var(--bg3);padding:.5rem .8rem;text-align:left;color:var(--leather-light);font-weight:400;letter-spacing:1px;font-size:.65rem;text-transform:uppercase}
td{padding:.5rem .8rem;border-bottom:1px solid var(--bg3);color:var(--cream-dim);vertical-align:top;word-break:break-all}
tr:hover td{background:var(--bg2)}
.empty{color:var(--cream-muted);text-align:center;padding:2rem;font-style:italic}
.btn{font-family:var(--font-mono);font-size:.75rem;padding:.4rem 1rem;border:1px solid var(--leather);background:transparent;color:var(--cream);cursor:pointer;transition:all .2s}
.btn:hover{border-color:var(--rust-light);color:var(--rust-light)}
.btn-rust{border-color:var(--rust);color:var(--rust-light)}.btn-rust:hover{background:var(--rust);color:var(--cream)}
.pill{display:inline-block;font-family:var(--font-mono);font-size:.6rem;padding:.1rem .4rem;border-radius:2px;text-transform:uppercase}
.pill-get{background:#1a3a2a;color:var(--green)}.pill-post{background:#2a1f1a;color:var(--rust-light)}
.pill-del{background:#2a1a1a;color:var(--red)}.pill-ok{background:#1a3a2a;color:var(--green)}
.pill-err{background:#2a1a1a;color:var(--red)}
.mono{font-family:var(--font-mono);font-size:.78rem}
.lbl{font-family:var(--font-mono);font-size:.62rem;letter-spacing:1px;text-transform:uppercase;color:var(--leather)}
.upgrade{background:var(--bg2);border:1px solid var(--rust-dark);border-left:3px solid var(--rust);padding:.8rem 1.2rem;font-size:.82rem;color:var(--cream-dim);margin-bottom:1.5rem}
.upgrade a{color:var(--rust-light)}
pre{background:var(--bg3);padding:.8rem 1rem;font-family:var(--font-mono);font-size:.75rem;color:var(--cream-dim);overflow-x:auto;max-width:100%}
input,select{font-family:var(--font-mono);font-size:.78rem;background:var(--bg3);border:1px solid var(--bg3);color:var(--cream);padding:.4rem .7rem;outline:none}
input:focus,select:focus{border-color:var(--leather)}
.row{display:flex;gap:.8rem;align-items:flex-end;flex-wrap:wrap;margin-bottom:1rem}
.field{display:flex;flex-direction:column;gap:.3rem}
.sserow{padding:.4rem .8rem;border-bottom:1px solid var(--bg3);font-family:var(--font-mono);font-size:.72rem;color:var(--cream-dim);display:grid;grid-template-columns:120px 60px 1fr;gap:.5rem}
.sserow:nth-child(odd){background:var(--bg2)}
</style></head><body>
<div class="hdr">
  <div class="hdr-left">
    <svg viewBox="0 0 64 64" width="22" height="22" fill="none"><rect x="8" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="28" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="48" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="8" y="27" width="48" height="7" rx="2.5" fill="#c4a87a"/></svg>
    <span class="hdr-brand">Stockyard</span>
    <span class="hdr-title">Fence</span>
  </div>
  <div style="display:flex;gap:.8rem;align-items:center">
    <span id="tier-badge" class="badge badge-free">Free</span>
    <a href="/api/stats" class="lbl" style="color:var(--leather)">API</a>
    <a href="https://stockyard.dev/fence/" class="lbl" style="color:var(--leather)">Docs</a>
  </div>
</div>
<div class="main">

<div class="cards">
  <div class="card"><span class="card-val" id="s-vaults">—</span><span class="card-lbl">Vaults</span></div>
  <div class="card"><span class="card-val" id="s-keys">—</span><span class="card-lbl">Keys</span></div>
  <div class="card"><span class="card-val" id="s-members">—</span><span class="card-lbl">Members</span></div>
  <div class="card"><span class="card-val" id="s-tokens">—</span><span class="card-lbl">Active Tokens</span></div>
  <div class="card"><span class="card-val" id="s-access">—</span><span class="card-lbl">Total Accesses</span></div>
</div>

<div class="section">
  <div class="section-title">Vaults</div>
  <div class="row">
    <div class="field"><span class="lbl">Name</span><input id="vault-name" placeholder="production" style="width:160px"></div>
    <button class="btn btn-rust" onclick="createVault()">+ Create Vault</button>
  </div>
  <table><thead><tr><th>ID</th><th>Name</th><th>Keys</th><th>Created</th><th></th></tr></thead>
  <tbody id="vault-list"><tr><td colspan="5" class="empty">Loading...</td></tr></tbody></table>
</div>

<div class="section" id="vault-detail" style="display:none">
  <div class="section-title">Vault: <span id="active-vault-name" style="color:var(--cream)"></span></div>
  <div style="display:grid;grid-template-columns:1fr 1fr;gap:2rem">

    <div>
      <div class="lbl" style="margin-bottom:.6rem">Keys <span style="font-size:.62rem;color:var(--cream-muted)">(values never shown)</span></div>
      <div class="row">
        <div class="field"><span class="lbl">Name</span><input id="key-name" placeholder="openai" style="width:110px"></div>
        <div class="field"><span class="lbl">Value</span><input id="key-val" type="password" placeholder="sk-..." style="width:140px"></div>
        <div class="field"><span class="lbl">Provider</span><input id="key-prov" placeholder="openai" style="width:90px"></div>
        <button class="btn btn-rust" style="align-self:flex-end" onclick="storeKey()">+ Store</button>
      </div>
      <table><thead><tr><th>Name</th><th>Provider</th><th>Updated</th><th></th></tr></thead>
      <tbody id="key-list"><tr><td colspan="4" class="empty">No keys yet.</td></tr></tbody></table>
    </div>

    <div>
      <div class="lbl" style="margin-bottom:.6rem">Members</div>
      <div class="row">
        <div class="field"><span class="lbl">Username</span><input id="mbr-name" placeholder="alice" style="width:120px"></div>
        <button class="btn btn-rust" style="align-self:flex-end" onclick="addMember()">+ Add</button>
      </div>
      <div class="lbl" style="margin-bottom:.6rem">Tokens</div>
      <table><thead><tr><th>Member</th><th>Token Name</th><th>Created</th><th></th></tr></thead>
      <tbody id="token-list"><tr><td colspan="4" class="empty">Loading...</td></tr></tbody></table>
      <div id="new-token-banner" style="display:none;margin-top:.8rem;padding:.6rem .8rem;background:var(--bg3);border-left:3px solid var(--gold)">
        <span class="lbl" style="color:var(--gold)">Token (save it — shown once):</span><br>
        <span id="new-token-val" class="mono" style="color:var(--cream);word-break:break-all;font-size:.7rem"></span>
      </div>
    </div>
  </div>
</div>

</div>
<script>
let _timer=null;
function autoReload(fn,ms=8000){if(_timer)clearInterval(_timer);_timer=setInterval(fn,ms)}
function ts(s){if(!s)return'-';const d=new Date(s);return d.toLocaleString()}
function rel(s){if(!s)return'-';const d=new Date(s),n=new Date(),diff=Math.round((n-d)/1000);if(diff<60)return diff+'s ago';if(diff<3600)return Math.round(diff/60)+'m ago';return Math.round(diff/3600)+'h ago'}
function fmt(n){return n===undefined||n===null?'-':n.toLocaleString()}
function pill(m){const c={'GET':'pill-get','POST':'pill-post','DELETE':'pill-del'}[m]||'';return '<span class="pill '+c+'">'+m+'</span>'}
function status(s){const ok=s>=200&&s<300;return '<span class="pill '+(ok?'pill-ok':'pill-err')+'">'+s+'</span>'}

const API='/api';
let activeVaultId=null,activeVaultMembers=[];

async function loadStats(){
  const r=await(await fetch(API+'/stats')).json().catch(()=>({}));
  document.getElementById('s-vaults').textContent=fmt(r.vaults);
  document.getElementById('s-keys').textContent=fmt(r.keys);
  document.getElementById('s-members').textContent=fmt(r.members);
  document.getElementById('s-tokens').textContent=fmt(r.tokens);
  document.getElementById('s-access').textContent=fmt(r.accesses);
}

async function loadVaults(){
  const r=await(await fetch(API+'/vaults')).json().catch(()=>({vaults:[]}));
  const vs=r.vaults||[];
  document.getElementById('vault-list').innerHTML=vs.length?vs.map(v=>
    ` + "`" + `<tr>
      <td class="mono" style="font-size:.7rem;color:var(--leather-light)">${v.id}</td>
      <td style="color:var(--cream)">${v.name}</td>
      <td>${v.key_count||0}</td>
      <td>${rel(v.created_at)}</td>
      <td><button class="btn" onclick="openVault('${v.id}','${v.name}')">Open</button></td>
    </tr>` + "`" + `).join(''):'<tr><td colspan="5" class="empty">No vaults yet — create one above.</td></tr>';
}

async function openVault(id,name){
  activeVaultId=id;
  document.getElementById('vault-detail').style.display='block';
  document.getElementById('active-vault-name').textContent=name;
  await Promise.all([loadKeys(),loadTokens(),loadMembers()]);
}

async function loadKeys(){
  if(!activeVaultId)return;
  const r=await(await fetch(API+'/vaults/'+activeVaultId+'/keys')).json().catch(()=>({keys:[]}));
  const ks=r.keys||[];
  document.getElementById('key-list').innerHTML=ks.length?ks.map(k=>
    ` + "`" + `<tr>
      <td style="color:var(--cream)">${k.name}</td>
      <td class="mono" style="font-size:.72rem;color:var(--leather-light)">${k.provider||'—'}</td>
      <td>${rel(k.updated_at||k.created_at)}</td>
      <td>
        <button class="btn" style="font-size:.65rem;padding:.2rem .5rem" onclick="rotateKey('${k.id}','${k.name}')">Rotate</button>
        <button class="btn" style="font-size:.65rem;padding:.2rem .5rem;margin-left:.3rem" onclick="deleteKey('${k.id}')">Del</button>
      </td>
    </tr>` + "`" + `).join(''):'<tr><td colspan="4" class="empty">No keys stored yet.</td></tr>';
}

async function loadMembers(){
  if(!activeVaultId)return;
  const r=await(await fetch(API+'/vaults/'+activeVaultId+'/members')).json().catch(()=>({members:[]}));
  activeVaultMembers=r.members||[];
}

async function loadTokens(){
  if(!activeVaultId)return;
  const r=await(await fetch(API+'/vaults/'+activeVaultId+'/tokens')).json().catch(()=>({tokens:[]}));
  const ts=r.tokens||[];
  document.getElementById('token-list').innerHTML=ts.length?ts.map(t=>
    ` + "`" + `<tr>
      <td class="mono" style="font-size:.72rem">${t.member_id}</td>
      <td style="color:var(--cream)">${t.name||'—'}</td>
      <td>${rel(t.created_at)}</td>
      <td><button class="btn" style="font-size:.65rem;padding:.2rem .5rem" onclick="revokeToken('${t.id}')">Revoke</button></td>
    </tr>` + "`" + `).join(''):'<tr><td colspan="4" class="empty">No tokens issued.</td></tr>';
}

async function createVault(){
  const name=document.getElementById('vault-name').value.trim();
  if(!name)return;
  const r=await fetch(API+'/vaults',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name})}).catch(()=>null);
  if(!r)return;
  if(r.status===402){alert('Free tier: 2 vault limit. Upgrade to Pro at stockyard.dev/fence/');return;}
  document.getElementById('vault-name').value='';loadVaults();
}

async function storeKey(){
  if(!activeVaultId)return;
  const name=document.getElementById('key-name').value.trim();
  const value=document.getElementById('key-val').value;
  const provider=document.getElementById('key-prov').value.trim();
  if(!name||!value)return;
  const r=await fetch(API+'/vaults/'+activeVaultId+'/keys',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name,value,provider})}).catch(()=>null);
  if(!r)return;
  if(r.status===402){alert('Free tier: 10 key limit. Upgrade to Pro at stockyard.dev/fence/');return;}
  document.getElementById('key-name').value='';document.getElementById('key-val').value='';
  document.getElementById('key-prov').value='';loadKeys();
}

async function addMember(){
  if(!activeVaultId)return;
  const username=document.getElementById('mbr-name').value.trim();
  if(!username)return;
  const r=await fetch(API+'/vaults/'+activeVaultId+'/members',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username,role:'reader'})}).catch(()=>null);
  if(!r)return;
  if(r.status===402){alert('Free tier: 2 member limit. Upgrade to Pro at stockyard.dev/fence/');return;}
  const j=await r.json().catch(()=>({}));
  const mbr=j.member;
  if(mbr){
    document.getElementById('mbr-name').value='';
    await loadMembers();
    // auto-issue a token for the new member
    const tr=await fetch(API+'/vaults/'+activeVaultId+'/tokens',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({member_id:mbr.id,name:username+'-token'})}).catch(()=>null);
    if(tr){const tj=await tr.json().catch(()=>({}));if(tj.token){document.getElementById('new-token-val').textContent=tj.token;document.getElementById('new-token-banner').style.display='block';}}
    loadTokens();
  }
}

async function rotateKey(id,name){
  const val=prompt('New value for "'+name+'":');
  if(!val)return;
  await fetch(API+'/vaults/'+activeVaultId+'/keys/'+id+'/rotate',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({value:val})});
  loadKeys();
}
async function deleteKey(id){if(!confirm('Delete key?'))return;await fetch(API+'/vaults/'+activeVaultId+'/keys/'+id,{method:'DELETE'});loadKeys();}
async function revokeToken(id){if(!confirm('Revoke token?'))return;await fetch(API+'/vaults/'+activeVaultId+'/tokens/'+id,{method:'DELETE'});loadTokens();}

async function refresh(){await Promise.all([loadStats(),loadVaults()]);if(activeVaultId)await Promise.all([loadKeys(),loadTokens()]);}
refresh();autoReload(refresh,10000);
</script></body></html>`

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(uiHTML))
}
