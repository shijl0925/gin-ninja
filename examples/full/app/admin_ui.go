package app

import "github.com/gin-gonic/gin"

// ServeAdminPrototype returns the standalone admin demo shell used by /admin, /admin/login, and /admin-prototype.
func ServeAdminPrototype(c *gin.Context) {
	c.Data(200, "text/html; charset=utf-8", []byte(adminPrototypeHTML))
}

const adminPrototypeHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Gin Ninja Admin</title>
  <style>
    :root { color-scheme: light; }
    [hidden] { display:none !important; }
    * { box-sizing: border-box; }
    body { font-family: Inter, system-ui, sans-serif; margin: 0; background: #eef2f7; color: #0f172a; }
    .topbar, .app-main { max-width: 1480px; margin: 0 auto; padding: 20px 24px; }
    .topbar { display:flex; gap:20px; align-items:flex-start; justify-content:space-between; }
    .topbar-brand, .topbar-copy, .topbar-meta, .sidebar-heading { display:grid; gap:8px; }
    .topbar-copy h1, .sidebar-heading h2, .section-title { margin:0; }
    .topbar-copy p, .sidebar-heading p, .section-copy, .login-marketing p, .login-lead p { margin:0; }
    .topbar-meta { display:flex; justify-content:flex-end; align-items:center; min-width:0; }
    .app-main { display:grid; gap:20px; padding-top:0; }
    .panel { min-width:0; background:#fff; border:1px solid #dbe2ea; border-radius:18px; padding:20px; box-shadow:0 12px 30px rgba(15, 23, 42, 0.06); }
    .stack { display:grid; gap:16px; }
    .toolbar { display:flex; gap:12px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    .toolbar > *, .row-actions > *, .workspace-meta > *, .table-toolbar > *, .pagination-bar > * { min-width:0; }
    .row-actions { display:flex; gap:10px; align-items:center; flex-wrap:wrap; }
    .eyebrow { display:inline-flex; width:max-content; align-items:center; gap:6px; border-radius:999px; padding:6px 10px; background:#dbeafe; color:#1d4ed8; font-size:12px; font-weight:700; letter-spacing:0.08em; text-transform:uppercase; }
    .eyebrow.subtle { background:#e2e8f0; color:#334155; }
    .badge { display:inline-flex; align-items:center; gap:6px; font-size:12px; font-weight:600; background:#eef2ff; color:#3730a3; border-radius:999px; padding:7px 12px; }
    .status-banner { border-radius:16px; border:1px solid #dbe2ea; background:#fff; color:#334155; padding:14px 16px; font-size:14px; line-height:1.5; box-shadow:0 10px 24px rgba(15, 23, 42, 0.05); }
    .status-banner[data-tone="info"] { background:#eff6ff; border-color:#bfdbfe; color:#1d4ed8; }
    .status-banner[data-tone="success"] { background:#ecfdf5; border-color:#a7f3d0; color:#047857; }
    .status-banner[data-tone="danger"] { background:#fef2f2; border-color:#fecaca; color:#b91c1c; }
    .login-shell { display:grid; gap:20px; }
    .session-panel { position:relative; overflow:hidden; }
    .login-marketing, .login-lead, .login-credentials { display:none; }
    .login-marketing { align-content:start; }
    .login-marketing h2, .login-lead h2 { margin:0; line-height:1.1; }
    .login-feature-list { display:grid; gap:12px; }
    .login-feature { display:grid; gap:4px; padding:16px 18px; background:rgba(255,255,255,0.08); border:1px solid rgba(191,219,254,0.18); border-radius:16px; }
    .login-feature strong { font-size:15px; }
    .login-credentials { gap:8px; padding:12px 14px; border-radius:14px; background:#f8fafc; border:1px solid #dbeafe; }
    .login-credentials code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size:13px; background:#fff; border:1px solid #dbeafe; border-radius:8px; padding:2px 6px; }
    .admin-shell { display:grid; gap:20px; grid-template-columns: 300px minmax(0, 1fr); align-items:start; }
    .sidebar-panel { position:sticky; top:20px; }
    .nav-list { list-style:none; margin:0; padding:0; display:grid; gap:10px; }
    .nav-list li { margin:0; }
    .nav-link { width:100%; text-align:left; background:#f8fafc; color:#0f172a; border:1px solid #dbe2ea; padding:14px 16px; border-radius:14px; display:grid; gap:4px; font-weight:600; }
    .nav-link:hover { background:#eff6ff; border-color:#bfdbfe; }
    .nav-link.active { background:linear-gradient(135deg, #1d4ed8 0%, #3730a3 100%); border-color:#1d4ed8; color:#fff; box-shadow:0 14px 28px rgba(37, 99, 235, 0.22); }
    .workspace { min-width:0; }
    .workspace-header { display:flex; gap:12px 16px; align-items:center; justify-content:space-between; flex-wrap:wrap; padding:16px 18px; }
    .workspace-header-copy { display:grid; gap:6px; flex:1 1 380px; min-width:0; }
    .workspace-header-copy h2,
    .workspace-header-copy p { margin:0; }
    .workspace-header-copy h2 { font-size:clamp(1.55rem, 2vw, 1.9rem); line-height:1.1; }
    .workspace-path { display:inline-flex; width:max-content; max-width:100%; align-items:center; padding:0; font-size:12px; line-height:1.45; color:#64748b; }
    .workspace-meta { display:flex; gap:10px; align-items:center; justify-content:flex-end; flex:0 0 auto; margin-left:auto; }
    .content-grid { display:grid; gap:16px; grid-template-columns:minmax(0, 1fr); align-items:start; }
    .section-shell { display:grid; gap:14px; }
    .section-heading { display:grid; gap:6px; }
    .two-col { display:grid; gap:20px; grid-template-columns:repeat(auto-fit, minmax(240px, 1fr)); }
    .filters { display:grid; gap:12px; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); }
    .inline-field, .form-field { display:grid; gap:8px; font-size:14px; font-weight:600; color:#334155; }
    .field-help, .muted { font-size:13px; color:#64748b; }
    .relation-control { display:grid; gap:10px; }
    .relation-preview { display:grid; gap:6px; margin:0; padding:0; list-style:none; }
    .relation-preview li { font-size:12px; color:#334155; background:#fff; border:1px solid #e2e8f0; border-radius:10px; padding:8px 10px; }
    .relation-preview mark { background:#fef3c7; padding:0; }
    .detail-layout { display:grid; gap:16px; grid-template-columns:minmax(0, 1fr); align-items:start; }
    .content-grid > *, .content-grid form, .detail-layout > *, .detail-layout form, .bulk-edit-field { min-width:0; }
    .detail-card { border:1px solid #e2e8f0; border-radius:16px; padding:18px; background:#fff; }
    .detail-grid { display:grid; gap:10px; }
    .detail-row { display:grid; grid-template-columns: 160px 1fr; gap:12px; border-bottom:1px solid #edf2f7; padding-bottom:10px; }
    .detail-row:last-child { border-bottom:none; padding-bottom:0; }
    .detail-label { font-size:12px; font-weight:700; color:#64748b; text-transform:uppercase; letter-spacing:0.06em; }
    .detail-value { font-size:14px; word-break:break-word; color:#0f172a; }
    .bulk-edit-fields { display:grid; gap:12px; }
    .bulk-edit-field { border:1px solid #e2e8f0; border-radius:14px; padding:14px; background:#fff; }
    .table-toolbar, .pagination-bar { display:flex; gap:12px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    .table-toolbar .row-actions { flex:1 1 480px; }
    .table-toolbar input, .table-toolbar select { flex:1 1 180px; min-width:0; }
    .pagination-info { font-size:14px; color:#64748b; }
    .table-shell { overflow:auto; border:1px solid #e2e8f0; border-radius:16px; background:#fff; }
    .empty-state { border:1px dashed #cbd5e1; border-radius:16px; padding:28px 20px; background:#fff; color:#64748b; text-align:center; }
    .workspace-actions { display:flex; gap:8px; flex-wrap:wrap; justify-content:flex-end; }
    .workspace-actions button { padding-inline:14px; }
    .modal-overlay { position:fixed; inset:0; background:rgba(15, 23, 42, 0.56); display:grid; place-items:center; padding:24px; z-index:50; }
    .modal-dialog { width:min(720px, 100%); max-height:min(85vh, 920px); overflow:auto; border-radius:24px; border:1px solid #dbe2ea; background:#fff; box-shadow:0 30px 90px rgba(15, 23, 42, 0.28); }
    .modal-dialog.large { width:min(860px, 100%); }
    .modal-header { display:flex; gap:16px; align-items:flex-start; justify-content:space-between; flex-wrap:wrap; padding:24px 24px 0; }
    .modal-body { padding:0 24px 24px; }
    .modal-close { min-width:44px; min-height:44px; padding:0 14px; }
    body.modal-open { overflow:hidden; }
    label { display:grid; gap:8px; font-size:14px; font-weight:600; color:#334155; }
    input, select, textarea, button { font: inherit; padding: 11px 13px; border-radius: 12px; border: 1px solid #cbd5e1; background:#fff; color:#0f172a; transition:border-color 120ms ease, box-shadow 120ms ease, background 120ms ease; }
    input:focus, select:focus, textarea:focus { outline:none; border-color:#3b82f6; box-shadow:0 0 0 3px rgba(59, 130, 246, 0.18); }
    textarea { min-height: 112px; }
    button { cursor:pointer; background:linear-gradient(135deg, #0f172a 0%, #1d4ed8 100%); color:#fff; border-color:transparent; font-weight:600; }
    button.secondary { background:#fff; color:#0f172a; border-color:#cbd5e1; }
    button.danger { background:linear-gradient(135deg, #991b1b 0%, #dc2626 100%); }
    button:disabled, input:disabled, select:disabled, textarea:disabled { opacity:0.6; cursor:not-allowed; }
    table { width:100%; border-collapse:separate; border-spacing:0; min-width:720px; }
    th, td { border-bottom:1px solid #e2e8f0; padding:12px 14px; text-align:left; font-size:14px; vertical-align:top; }
    th { background:#f8fafc; color:#475569; font-size:12px; font-weight:700; text-transform:uppercase; letter-spacing:0.06em; }
    tbody tr:hover { background:#f8fbff; }
    tbody tr.row-selected { background:#eff6ff; }
    .action-cell { display:flex; gap:6px; align-items:center; white-space:nowrap; }
    .action-menu { position:relative; display:inline-block; }
    .action-menu-trigger { background:#fff; color:#0f172a; border:1px solid #cbd5e1; padding:6px 10px; font-size:13px; font-weight:600; border-radius:10px; cursor:pointer; line-height:1; }
    .action-menu-trigger:hover { background:#f1f5f9; border-color:#94a3b8; }
    .action-menu-list { display:none; position:absolute; right:0; top:calc(100% + 4px); min-width:130px; background:#fff; border:1px solid #dbe2ea; border-radius:12px; box-shadow:0 8px 24px rgba(15,23,42,0.12); z-index:100; overflow:hidden; }
    .action-menu-list.open { display:block; }
    .action-menu-item { display:block; width:100%; text-align:left; background:none; color:#0f172a; border:none; border-radius:0; padding:10px 14px; font-size:14px; font-weight:500; cursor:pointer; transition:background 80ms; }
    .action-menu-item:hover { background:#f1f5f9; }
    .action-menu-item:disabled { opacity:0.45; cursor:not-allowed; }
    .action-menu-divider { border:none; border-top:1px solid #e2e8f0; margin:4px 0; }
    .action-menu-item.danger { color:#dc2626; }
    .action-menu-item.danger:hover { background:#fef2f2; }
    .action-btn-view { background:#fff; color:#0f172a; border:1px solid #cbd5e1; padding:6px 12px; font-size:13px; font-weight:600; border-radius:10px; cursor:pointer; line-height:1; }
    .action-btn-view:hover { background:#f1f5f9; border-color:#94a3b8; }
    pre { margin:0; white-space:pre-wrap; word-break:break-word; background:#0f172a; color:#e2e8f0; padding:14px; border-radius:14px; }
    body.standalone-login-page { background:
      radial-gradient(circle at top left, rgba(59,130,246,0.18), transparent 36%),
      radial-gradient(circle at bottom right, rgba(99,102,241,0.22), transparent 34%),
      linear-gradient(180deg, #eef4ff 0%, #f8fafc 45%, #eef2ff 100%);
    }
    body.standalone-login-page .topbar,
    body.standalone-login-page .app-main { max-width:1120px; padding:24px; }
    body.standalone-login-page .topbar { padding-top:48px; }
    body.standalone-login-page .login-shell { gap:24px; grid-template-columns: minmax(0, 1.1fr) minmax(360px, 420px); align-items:stretch; }
    body.standalone-login-page .login-marketing { display:grid; gap:18px; color:#e5eefb; background:linear-gradient(135deg, #0f172a 0%, #1d4ed8 100%); border-radius:24px; padding:32px; box-shadow:0 24px 60px rgba(15, 23, 42, 0.22); }
    body.standalone-login-page .login-lead,
    body.standalone-login-page .login-credentials { display:grid; }
    body.standalone-login-page .session-panel { margin:0; padding:28px; border-color:#dbeafe; border-radius:24px; box-shadow:0 24px 60px rgba(15, 23, 42, 0.12); }
    body.standalone-login-page #loginForm { grid-template-columns:1fr; gap:14px; }
    body.standalone-login-page #loginForm label { font-size:13px; font-weight:600; color:#475569; }
    body.standalone-login-page input { min-height:48px; border-color:#cbd5e1; background:#fff; }
    body.standalone-login-page button { min-height:48px; }
    body.standalone-login-page #loginButton { width:100%; background:linear-gradient(135deg, #111827 0%, #1d4ed8 100%); box-shadow:0 12px 24px rgba(29, 78, 216, 0.2); }
    body.standalone-login-page #status { background:#0f172a; border-color:#0f172a; color:#e2e8f0; }
    @media (min-width: 1180px) {
      .detail-layout { grid-template-columns: minmax(0, 1.1fr) minmax(300px, 0.9fr); }
    }
    @media (max-width: 1380px) {
      .admin-shell { grid-template-columns:1fr; }
      .sidebar-panel { position:static; }
    }
    @media (max-width: 960px) {
      body.standalone-login-page .login-shell { grid-template-columns:1fr; }
      .topbar, .app-main, body.standalone-login-page .topbar, body.standalone-login-page .app-main { padding:16px; }
      body.standalone-login-page .topbar { padding-top:24px; }
      .topbar { flex-direction:column; }
      .topbar-meta { justify-content:flex-start; width:100%; }
      .table-toolbar .row-actions { flex-basis:100%; }
    }
  </style>
</head>
<body>
  <header class="topbar">
    <div class="topbar-brand">
      <span id="shellEyebrow" class="eyebrow">Admin Console</span>
      <div class="topbar-copy">
        <h1 id="pageTitle">Gin Ninja Admin</h1>
        <p id="pageIntro" class="muted">A metadata-driven admin UI for the example admin APIs.</p>
      </div>
    </div>
    <div class="topbar-meta">
      <div id="sessionActions" class="row-actions" hidden>
        <button id="loadResources" type="button" class="secondary">Refresh workspace</button>
        <button id="clearToken" type="button">Sign out</button>
      </div>
    </div>
  </header>
  <main class="app-main">
    <div id="status" class="status-banner" data-tone="neutral">Ready.</div>
    <section id="sessionShell" class="login-shell">
      <section class="login-marketing">
        <span class="eyebrow">Admin Console</span>
        <div class="stack">
          <h2>A cleaner sign-in for the standalone admin console.</h2>
          <p>Use a dedicated entrypoint to authenticate, then jump straight into the example back-office experience.</p>
        </div>
        <div class="login-feature-list">
          <div class="login-feature">
            <strong>Dedicated admin entrypoint</strong>
            <span>Keep login, redirects, and restored JWT sessions separate from the prototype page.</span>
          </div>
          <div class="login-feature">
            <strong>Metadata-driven operations</strong>
            <span>Explore resources, filters, relations, and bulk actions after sign-in.</span>
          </div>
          <div class="login-feature">
            <strong>Demo-friendly access</strong>
            <span>Use the seeded example account to try the full admin flow without extra setup.</span>
          </div>
        </div>
      </section>
      <section class="panel stack session-panel">
        <div class="login-lead">
          <span class="eyebrow">Secure Sign In</span>
          <h2>Welcome back</h2>
          <p class="muted">Authenticate with the seeded demo admin account to enter the standalone console.</p>
        </div>
        <div class="login-credentials">
          <strong>Demo credentials</strong>
          <div class="muted">Email: <code>alice@example.com</code></div>
          <div class="muted">Password: <code>password123</code></div>
        </div>
        <form id="loginForm" class="two-col">
          <label>Email
            <input id="loginEmail" type="email" placeholder="alice@example.com" autocomplete="username email">
          </label>
          <label>Password
            <input id="loginPassword" type="password" placeholder="password123" autocomplete="current-password">
          </label>
          <div class="row-actions">
            <button id="loginButton" type="submit">Sign in</button>
          </div>
        </form>
        <div id="manualTokenTools" class="stack">
          <label>JWT token
            <input id="token" placeholder="Paste a token from /api/v1/auth/login" autocomplete="off">
          </label>
          <p class="muted">Successful sign-in stores the JWT in localStorage and attaches it to every admin request automatically.</p>
        </div>
      </section>
    </section>
    <section id="adminShell" class="admin-shell" hidden>
      <aside>
        <section class="panel sidebar-panel stack">
          <div class="sidebar-heading">
            <span class="eyebrow subtle">Navigation</span>
            <div class="stack" style="gap:4px;">
              <h2>Resources</h2>
              <p class="muted">Choose a resource to manage records, filters, and bulk actions.</p>
            </div>
          </div>
          <ul id="resources" class="nav-list"></ul>
        </section>
      </aside>
      <section class="workspace stack">
        <section class="panel workspace-header">
          <div class="workspace-header-copy">
            <span class="eyebrow subtle">Admin Workspace</span>
            <h2 id="resourceTitle">Select a resource</h2>
            <p id="resourcePath" class="workspace-path muted">Sign in to open a resource workspace.</p>
          </div>
          <div class="workspace-meta">
            <div class="workspace-actions">
              <button id="openCreateModal" type="button">Create record</button>
            </div>
          </div>
        </section>
        <section class="content-grid">
          <section class="stack">
            <section class="panel section-shell">
              <div class="toolbar">
                <div class="section-heading">
                  <h3 class="section-title">Records</h3>
                  <p class="section-copy muted">Search, filter, sort, and bulk manage the current resource.</p>
                </div>
                <div class="row-actions">
                  <span id="selectedCountBadge" class="badge">0 selected</span>
                  <button id="reloadList" class="secondary" type="button">Refresh list</button>
                  <button id="clearFilters" class="secondary" type="button">Clear filters</button>
                  <button id="bulkDelete" class="danger" type="button">Bulk delete</button>
                </div>
              </div>
              <div class="table-toolbar">
                <div class="row-actions">
                  <input id="search" placeholder="Search current resource">
                  <select id="sort"></select>
                  <select id="pageSize">
                    <option value="5">5 / page</option>
                    <option value="10" selected>10 / page</option>
                    <option value="20">20 / page</option>
                    <option value="50">50 / page</option>
                  </select>
                </div>
                <div class="pagination-info" id="paginationInfo">Page 1 of 1</div>
              </div>
              <form id="filtersForm" class="filters"></form>
              <div class="pagination-bar">
                <div class="muted">Use filters to refine the current workspace.</div>
                <div class="row-actions">
                  <button id="prevPage" class="secondary" type="button">Previous</button>
                  <button id="nextPage" class="secondary" type="button">Next</button>
                </div>
              </div>
              <div id="list"></div>
            </section>
          </section>
        </section>
        <section id="createModal" class="modal-overlay" hidden>
          <div class="modal-dialog" role="dialog" aria-modal="true" aria-labelledby="createModalTitle">
            <div class="modal-header">
              <div class="section-heading">
                <h3 id="createModalTitle" class="section-title">Create record</h3>
                <p class="section-copy muted">Use the same admin layout to add a new record to the active resource.</p>
              </div>
              <button id="closeCreateModal" type="button" class="secondary modal-close" aria-label="Close create record dialog">Close</button>
            </div>
            <div class="modal-body">
              <form id="createForm" class="stack"></form>
            </div>
          </div>
        </section>
        <section id="recordModal" class="modal-overlay" hidden>
          <div class="modal-dialog large" role="dialog" aria-modal="true" aria-labelledby="recordModalTitle">
            <div class="modal-header">
              <div class="section-heading">
                <div class="row-actions">
                  <h3 id="recordModalTitle" class="section-title">Open record</h3>
                  <span id="detailObjectBadge" class="badge">Draft view</span>
                </div>
                <p class="section-copy muted">Inspect the selected record and review the reference payload in a focused dialog.</p>
              </div>
              <button id="closeRecordModal" type="button" class="secondary modal-close" aria-label="Close record dialog">Close</button>
            </div>
            <div class="modal-body">
              <div class="detail-layout">
                <section class="stack">
                  <div class="detail-card stack">
                    <div class="toolbar">
                      <strong id="detailTitle">No record selected</strong>
                    </div>
                    <div id="detailFields" class="detail-grid">
                      <p class="muted">No record selected.</p>
                    </div>
                  </div>
                  <div class="detail-card stack">
                    <strong>Reference payload</strong>
                    <pre id="detail">No record selected.</pre>
                  </div>
                </section>
              </div>
            </div>
          </div>
        </section>
        <section id="editModal" class="modal-overlay" hidden>
          <div class="modal-dialog" role="dialog" aria-modal="true" aria-labelledby="editModalTitle">
            <div class="modal-header">
              <div class="section-heading">
                <h3 id="editModalTitle" class="section-title">Edit record</h3>
                <p class="section-copy muted" id="editHint">Select a row to open the change form.</p>
              </div>
              <button id="closeEditModal" type="button" class="secondary modal-close" aria-label="Close edit record dialog">Close</button>
            </div>
            <div class="modal-body">
              <form id="updateForm" class="stack"></form>
            </div>
          </div>
        </section>
      </section>
    </section>
  <script>
    const apiBase = '/api/v1/admin';
    const tokenStorageKey = 'gin-ninja-admin-token';
    const flashStorageKey = 'gin-ninja-admin-flash';
    const adminPagePath = '/admin';
    const adminLoginPath = '/admin/login';
    const prototypePagePath = '/admin-prototype';
    const numericFieldPattern = /^-?\d+(?:\.\d+)?$/;
    const state = {
      auth: { name: '', userID: null },
      current: null,
      meta: null,
      resources: [],
      records: [],
      selected: null,
      bulkSelected: {},
      relationSearch: {},
      relationTimers: {},
      searchTimer: null,
      pagination: { page: 1, size: 10, pages: 1, total: 0 }
    };

     const els = {
      loginForm: document.getElementById('loginForm'),
      loginEmail: document.getElementById('loginEmail'),
      loginPassword: document.getElementById('loginPassword'),
      token: document.getElementById('token'),
      manualTokenTools: document.getElementById('manualTokenTools'),
      clearToken: document.getElementById('clearToken'),
      sessionActions: document.getElementById('sessionActions'),
      sessionShell: document.getElementById('sessionShell'),
      status: document.getElementById('status'),
      pageTitle: document.getElementById('pageTitle'),
      pageIntro: document.getElementById('pageIntro'),
      shellEyebrow: document.getElementById('shellEyebrow'),
      adminShell: document.getElementById('adminShell'),
      resources: document.getElementById('resources'),
      resourceTitle: document.getElementById('resourceTitle'),
      resourcePath: document.getElementById('resourcePath'),
      selectedCountBadge: document.getElementById('selectedCountBadge'),
      detailTitle: document.getElementById('detailTitle'),
      detailObjectBadge: document.getElementById('detailObjectBadge'),
      detailFields: document.getElementById('detailFields'),
      createForm: document.getElementById('createForm'),
      createModal: document.getElementById('createModal'),
      openCreateModal: document.getElementById('openCreateModal'),
      closeCreateModal: document.getElementById('closeCreateModal'),
      recordModal: document.getElementById('recordModal'),
      closeRecordModal: document.getElementById('closeRecordModal'),
      editModal: document.getElementById('editModal'),
      closeEditModal: document.getElementById('closeEditModal'),
      updateForm: document.getElementById('updateForm'),
      editHint: document.getElementById('editHint'),
      filtersForm: document.getElementById('filtersForm'),
      sort: document.getElementById('sort'),
      pageSize: document.getElementById('pageSize'),
      paginationInfo: document.getElementById('paginationInfo'),
      prevPage: document.getElementById('prevPage'),
      nextPage: document.getElementById('nextPage'),
      list: document.getElementById('list'),
      detail: document.getElementById('detail'),
      loadResources: document.getElementById('loadResources'),
      reloadList: document.getElementById('reloadList'),
      clearFilters: document.getElementById('clearFilters'),
      bulkDelete: document.getElementById('bulkDelete'),
      search: document.getElementById('search')
    };

    function inferStatusTone(value) {
      const message = String(value || '').toLowerCase();
      if (!message) return 'neutral';
      if (message.includes('expired') || message.includes('error') || message.includes('failed') || message.includes('did not') || message.includes('no primary key')) {
        return 'danger';
      }
      if (message.includes('signed in') || message.includes('created') || message.includes('updated') || message.includes('deleted') || message.includes('cleared') || message.includes('signed out')) {
        return 'success';
      }
      if (message.includes('loaded') || message.includes('redirect') || message.includes('ready') || message.includes('restored')) {
        return 'info';
      }
      return 'neutral';
    }

    function setStatus(value, tone) {
      els.status.textContent = value;
      els.status.dataset.tone = tone || inferStatusTone(value);
    }

    function currentPagePath() {
      return window.location.pathname || '';
    }

    function isStandaloneLoginPage() {
      return currentPagePath() === adminLoginPath;
    }

    function isStandaloneAdminPage() {
      return currentPagePath() === adminPagePath;
    }

    function isLegacyPrototypePage() {
      return currentPagePath() === prototypePagePath;
    }

    function rememberFlashMessage(value) {
      if (!value) return;
      sessionStorage.setItem(flashStorageKey, value);
    }

    function consumeFlashMessage() {
      const value = sessionStorage.getItem(flashStorageKey);
      if (value) {
        sessionStorage.removeItem(flashStorageKey);
      }
      return value;
    }

    function updatePageChrome() {
      document.body.classList.toggle('standalone-login-page', isStandaloneLoginPage());
      if (isStandaloneLoginPage()) {
        document.title = 'Gin Ninja Admin Login';
        els.shellEyebrow.textContent = 'Admin Login';
        els.pageTitle.textContent = 'Gin Ninja Admin Login';
        els.pageIntro.textContent = 'Sign in to enter the example admin console.';
        return;
      }
      if (isStandaloneAdminPage()) {
        document.title = 'Gin Ninja Admin';
        els.shellEyebrow.textContent = 'Admin Console';
        els.pageTitle.textContent = 'Gin Ninja Admin';
        els.pageIntro.textContent = 'An operations workspace for the example back-office experience.';
        return;
      }
      document.title = 'Gin Ninja Admin Prototype';
      els.shellEyebrow.textContent = 'Prototype Demo';
      els.pageTitle.textContent = 'Gin Ninja Admin Prototype';
      els.pageIntro.textContent = 'A sandboxed version of the metadata-driven admin workspace.';
    }

    function redirectToLogin(message) {
      if (isLegacyPrototypePage()) {
        if (message) setStatus(message);
        return;
      }
      rememberFlashMessage(message);
      if (!isStandaloneLoginPage()) {
        window.location.replace(adminLoginPath);
      }
    }

    function redirectToAdmin(message) {
      if (isLegacyPrototypePage()) {
        if (message) setStatus(message);
        return;
      }
      rememberFlashMessage(message);
      if (!isStandaloneAdminPage()) {
        window.location.replace(adminPagePath);
      }
    }

    function hasToken() {
      return !!els.token.value.trim();
    }

    function persistToken() {
      const token = els.token.value.trim();
      if (token) {
        localStorage.setItem(tokenStorageKey, token);
      } else {
        localStorage.removeItem(tokenStorageKey);
      }
    }

    function restoreToken() {
      const saved = localStorage.getItem(tokenStorageKey);
      if (saved) {
        els.token.value = saved;
        return true;
      }
      return false;
    }

     function renderSignedOutState() {
       closeAllModals();
       const standaloneAdminPage = isStandaloneAdminPage();
      els.loginForm.hidden = false;
      els.sessionShell.hidden = false;
      els.sessionActions.hidden = true;
      els.manualTokenTools.hidden = true;
      els.adminShell.hidden = true;
    }

    function renderSignedInState() {
      const standaloneLoginPage = isStandaloneLoginPage();
      els.loginForm.hidden = true;
      els.sessionActions.hidden = standaloneLoginPage;
      els.sessionShell.hidden = true;
      els.manualTokenTools.hidden = true;
      els.adminShell.hidden = standaloneLoginPage;
    }

    function renderAuthState() {
      if (hasToken()) {
        renderSignedInState();
        if (isStandaloneLoginPage()) {
          redirectToAdmin('Restored saved token. Redirecting to /admin.');
        }
      } else {
        renderSignedOutState();
      }
    }

    function resetAdminState() {
      state.auth = { name: '', userID: null };
      state.current = null;
      state.meta = null;
      state.resources = [];
      state.records = [];
      state.selected = null;
      state.bulkSelected = {};
      state.relationSearch = {};
      state.relationTimers = {};
      state.pagination = { page: 1, size: Number(els.pageSize.value || 10), pages: 1, total: 0 };
       renderResources();
       els.resourceTitle.textContent = 'Select a resource';
       els.resourcePath.textContent = 'Sign in to open a resource workspace.';
       els.detailTitle.textContent = 'No record selected';
       els.detailObjectBadge.textContent = 'Draft view';
       els.detailFields.innerHTML = '<p class="muted">No record selected.</p>';
      els.detail.textContent = 'No record selected.';
      els.createForm.innerHTML = '<p class="muted">Sign in to create records.</p>';
      els.updateForm.innerHTML = '<p class="muted">Sign in to edit records.</p>';
      els.filtersForm.innerHTML = '';
      els.sort.innerHTML = '';
      els.list.innerHTML = '<div class="empty-state">Sign in to browse records in the admin workspace.</div>';
       els.editHint.textContent = 'Sign in to open the change form.';
       renderPagination();
       syncBulkActionState();
      syncWorkspaceActionState();
    }

    function logout(message) {
      els.token.value = '';
      persistToken();
      resetAdminState();
      renderAuthState();
      els.loginPassword.value = '';
      if (isStandaloneAdminPage()) {
        redirectToLogin(message || 'Signed out of the admin console.');
        return;
      }
      if (message) {
        setStatus(message);
      }
    }

    function requestHeaders(options = {}) {
      const headers = new Headers(options.headers || {});
      if (!(options.body instanceof FormData)) {
        headers.set('Content-Type', 'application/json');
      }
      const token = els.token.value.trim();
      if (token) headers.set('Authorization', 'Bearer ' + token);
      return headers;
    }

    async function request(path, options = {}) {
      const { skipAuthRedirect, ...requestOptions } = options;
      persistToken();
      const response = await fetch(path, { ...requestOptions, headers: requestHeaders(requestOptions) });
      const text = await response.text();
      let data = null;
      try { data = text ? JSON.parse(text) : null; } catch (_) { data = text; }
      if (!response.ok) {
        if (response.status === 401 && !skipAuthRedirect) {
          logout('Session expired. Please sign in again.');
          throw new Error('Session expired. Please sign in again.');
        }
        throw new Error(typeof data === 'string' ? data : JSON.stringify(data, null, 2));
      }
      return data;
    }

    function currentBasePath() {
      return apiBase + '/resources' + state.current.path;
    }

    function hasAction(action) {
      return (state.meta?.actions || []).includes(action);
    }

    function recordPrimaryKey(record) {
      return record?.id;
    }

    function fieldMeta(name) {
      return (state.meta?.fields || []).find((field) => field.name === name);
    }

    function fieldValue(name) {
      return els.filtersForm.elements.namedItem(name);
    }

    function selectionKey(id) {
      return JSON.stringify(id);
    }

    function selectedIDs() {
      return Object.values(state.bulkSelected);
    }

    function isSelectedForBulk(id) {
      return Object.prototype.hasOwnProperty.call(state.bulkSelected, selectionKey(id));
    }

    function setSelectedForBulk(id, checked) {
      const key = selectionKey(id);
      if (checked) {
        state.bulkSelected[key] = id;
      } else {
        delete state.bulkSelected[key];
      }
    }

    function escapeHTML(value) {
      return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
    }

    function highlightMatch(label, term) {
      const source = String(label || '');
      const query = String(term || '').trim();
      if (!query) {
        return escapeHTML(source);
      }
      const lowerSource = source.toLowerCase();
      const lowerQuery = query.toLowerCase();
      const index = lowerSource.indexOf(lowerQuery);
      if (index === -1) {
        return escapeHTML(source);
      }
      return escapeHTML(source.slice(0, index)) + '<mark>' + escapeHTML(source.slice(index, index + query.length)) + '</mark>' + escapeHTML(source.slice(index + query.length));
    }

    function formatValue(value) {
      if (value == null) return '—';
      if (Array.isArray(value)) return value.length ? JSON.stringify(value) : '[]';
      if (typeof value === 'boolean') return value ? 'Yes' : 'No';
      if (typeof value === 'object') return JSON.stringify(value);
      return String(value);
    }

    function relationStateKey(scopeKey, field) {
      return scopeKey + ':' + field.name;
    }

    function resetQueryState() {
      state.bulkSelected = {};
      state.relationSearch = {};
      state.pagination = { page: 1, size: Number(els.pageSize.value || 10), pages: 1, total: 0 };
      els.search.value = '';
      els.sort.innerHTML = '';
      els.filtersForm.innerHTML = '';
    }

    function resetToFirstPage() {
      state.pagination.page = 1;
    }

    function cancelScheduledSearchReload() {
      if (state.searchTimer) {
        clearTimeout(state.searchTimer);
        state.searchTimer = null;
      }
    }

    function scheduleSearchReload() {
      cancelScheduledSearchReload();
      state.searchTimer = setTimeout(() => {
        state.searchTimer = null;
        if (!state.current) return;
        resetToFirstPage();
        els.reloadList.click();
      }, 300);
    }

    function renderResources() {
      els.resources.innerHTML = '';
      state.resources.forEach((resource) => {
        const li = document.createElement('li');
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'nav-link' + (state.current?.name === resource.name ? ' active' : '');
        button.textContent = resource.label + ' (' + resource.name + ')';
        button.onclick = () => selectResource(resource);
        li.appendChild(button);
        els.resources.appendChild(li);
      });
    }

     function openModal(modal) {
       if (!modal || modal.hidden) {
         if (modal) {
           modal.hidden = false;
        }
      }
       document.body.classList.add('modal-open');
     }

     function anyModalOpen() {
       return [els.createModal, els.recordModal, els.editModal].some((modal) => modal && !modal.hidden);
     }

     function closeModal(modal) {
       if (modal) {
         modal.hidden = true;
       }
       if (!anyModalOpen()) {
         document.body.classList.remove('modal-open');
       }
     }

     function closeAllModals() {
       [els.createModal, els.recordModal, els.editModal].forEach((modal) => closeModal(modal));
     }

     function syncWorkspaceActionState() {
       const createEnabled = Boolean(state.current && hasAction('create'));
       els.openCreateModal.disabled = !createEnabled;
     }

    function renderResourceSummary() {
      if (!state.current || !state.meta) {
        els.resourcePath.textContent = 'Sign in to open a resource workspace.';
        return;
      }
      if (isLegacyPrototypePage()) {
        els.resourcePath.textContent = currentBasePath();
        return;
      }
      els.resourcePath.textContent = 'Browse, inspect, and edit ' + state.meta.label + '.';
    }

    function renderSortOptions() {
      els.sort.innerHTML = '';
      const empty = document.createElement('option');
      empty.value = '';
      empty.textContent = 'Default sort';
      els.sort.appendChild(empty);
      (state.meta?.sort_fields || []).forEach((name) => {
        const asc = document.createElement('option');
        asc.value = name;
        asc.textContent = 'Sort by ' + name + ' ↑';
        els.sort.appendChild(asc);
        const desc = document.createElement('option');
        desc.value = '-' + name;
        desc.textContent = 'Sort by ' + name + ' ↓';
        els.sort.appendChild(desc);
      });
    }

    function buildFilterControl(field) {
      const wrapper = document.createElement('label');
      wrapper.className = 'inline-field';
      wrapper.textContent = field.label;
      let input;
      if (field.component === 'checkbox') {
        input = document.createElement('select');
        [['', 'Any'], ['true', 'True'], ['false', 'False']].forEach((pair) => {
          const option = document.createElement('option');
          option.value = pair[0];
          option.textContent = pair[1];
          input.appendChild(option);
        });
      } else if (field.component === 'number') {
        input = document.createElement('input');
        input.type = 'number';
      } else if (field.component === 'datetime') {
        input = document.createElement('input');
        input.type = 'datetime-local';
      } else {
        input = document.createElement('input');
        input.type = 'text';
      }
      input.name = field.name;
      input.placeholder = 'Filter by ' + field.label;
      wrapper.appendChild(input);
      els.filtersForm.appendChild(wrapper);
    }

    function renderFilterControls() {
      els.filtersForm.innerHTML = '';
      const filterFields = state.meta?.filter_fields || [];
      if (!filterFields.length) {
        els.filtersForm.innerHTML = '<p class="muted">No filters available for this resource.</p>';
        return;
      }
      filterFields.forEach((name) => {
        const field = fieldMeta(name);
        if (field) buildFilterControl(field);
      });
    }

    function updateRelationPreview(preview, items, term) {
      const normalizedTerm = term.trim();
      if (!normalizedTerm) {
        preview.hidden = true;
        preview.innerHTML = '';
        return;
      }
      preview.hidden = false;
      if (!items.length) {
        preview.innerHTML = '<li>No matching options.</li>';
        return;
      }
      preview.innerHTML = items.slice(0, 5).map((item) => '<li>' + highlightMatch(item.label, normalizedTerm) + '</li>').join('');
    }

    async function loadRelationOptions(field, search, page, size) {
      const params = new URLSearchParams();
      if (search) params.set('search', search);
      params.set('page', String(page || 1));
      params.set('size', String(size || 8));
      const query = params.toString();
      const options = await request(currentBasePath() + '/fields/' + field.name + '/options?' + query);
      return options.items || [];
    }

    function resolveRelationSelection(items, selectedValue, term) {
      if (selectedValue != null && selectedValue !== '') {
        return selectedValue;
      }
      const normalizedTerm = term.trim();
      if (!normalizedTerm) {
        return selectedValue;
      }
      const exactValueMatch = items.find((item) => String(item.value) === normalizedTerm);
      return exactValueMatch ? exactValueMatch.value : selectedValue;
    }

    function populateRelationSelect(select, items, selectedValue, placeholderLabel) {
      select.innerHTML = '';
      const hasSelection = selectedValue != null && selectedValue !== '';
      if (!hasSelection) {
        const option = document.createElement('option');
        option.value = '';
        option.textContent = 'Choose ' + placeholderLabel;
        option.selected = true;
        select.appendChild(option);
      }
      items.forEach((item) => {
        const option = document.createElement('option');
        option.value = String(item.value);
        option.textContent = item.label;
        if (hasSelection && String(selectedValue) === String(item.value)) {
          option.selected = true;
        }
        select.appendChild(option);
      });
      if (hasSelection && !Array.from(select.options).some((option) => option.value === String(selectedValue))) {
        const option = document.createElement('option');
        option.value = String(selectedValue);
        option.textContent = 'Selected: ' + String(selectedValue);
        option.selected = true;
        select.appendChild(option);
      }
    }

    function setControlDisabled(control, disabled) {
      if ('disabled' in control) {
        control.disabled = disabled;
      }
      control.querySelectorAll('input, select, textarea, button').forEach((element) => {
        element.disabled = disabled;
      });
    }

    function scheduleRelationSearch(field, scopeKey, searchInput, select, preview) {
      const key = relationStateKey(scopeKey, field);
      clearTimeout(state.relationTimers[key]);
      state.relationTimers[key] = setTimeout(async () => {
        try {
          const term = searchInput.value.trim();
          const items = await loadRelationOptions(field, term, 1, 8);
          const nextValue = resolveRelationSelection(items, select.value, term);
          populateRelationSelect(select, items, nextValue, field.label);
          updateRelationPreview(preview, items, term);
          setStatus('Loaded ' + items.length + ' relation option(s) for ' + field.name + '.');
        } catch (error) {
          setStatus(String(error.message || error));
        }
      }, 300);
    }

    async function buildFieldControl(field, value, scopeKey) {
      if (field.relation) {
        const wrapper = document.createElement('div');
        wrapper.className = 'relation-control';
        const searchKey = relationStateKey(scopeKey, field);
        const searchInput = document.createElement('input');
        searchInput.type = 'text';
        searchInput.placeholder = 'Search related ' + field.label;
        searchInput.value = state.relationSearch[searchKey] || '';
        const select = document.createElement('select');
        select.name = field.name;
        const preview = document.createElement('ul');
        preview.className = 'relation-preview';
        const help = document.createElement('div');
        help.className = 'field-help';
        help.textContent = 'Search related records and choose the best matching option for this field.';
        wrapper.appendChild(searchInput);
        wrapper.appendChild(select);
        wrapper.appendChild(preview);
        wrapper.appendChild(help);
        const items = await loadRelationOptions(field, searchInput.value.trim(), 1, 8);
        const nextValue = resolveRelationSelection(items, value, searchInput.value);
        populateRelationSelect(select, items, nextValue, field.label);
        updateRelationPreview(preview, items, searchInput.value.trim());
        searchInput.addEventListener('input', () => {
          state.relationSearch[searchKey] = searchInput.value;
          scheduleRelationSearch(field, scopeKey, searchInput, select, preview);
        });
        return wrapper;
      }
      if (field.component === 'checkbox') {
        const input = document.createElement('input');
        input.type = 'checkbox';
        input.name = field.name;
        input.checked = Boolean(value);
        return input;
      }
      if (field.component === 'array' || field.component === 'text' || field.component === 'textarea') {
        const input = document.createElement('textarea');
        input.name = field.name;
        input.value = field.component === 'array'
          ? (Array.isArray(value) ? JSON.stringify(value, null, 2) : (value ? JSON.stringify(value, null, 2) : ''))
          : (value == null ? '' : String(value));
        return input;
      }
      const input = document.createElement('input');
      input.name = field.name;
      input.type = ({ email: 'email', password: 'password', number: 'number', datetime: 'datetime-local' }[field.component]) || 'text';
      input.value = value == null ? '' : String(value);
      return input;
    }

    async function renderForm(target, fieldNames, mode, values, scopeKey) {
      target.innerHTML = '';
      if (!state.meta || !fieldNames.length) {
        target.innerHTML = '<p class="muted">' + mode + ' is not available for this resource.</p>';
        return;
      }
      for (const name of fieldNames) {
        const field = fieldMeta(name);
        if (!field) continue;
        const wrapper = document.createElement('label');
        wrapper.className = 'form-field';
        wrapper.textContent = field.label;
        const control = await buildFieldControl(field, values[name], scopeKey);
        wrapper.appendChild(control);
        target.appendChild(wrapper);
      }
      const submit = document.createElement('button');
      submit.type = 'submit';
      submit.textContent = mode === 'update' ? 'Update' : 'Create';
      target.appendChild(submit);
    }

    async function renderCreateForm() {
      await renderForm(els.createForm, state.meta?.create_fields || [], 'create', {}, 'create');
    }

    async function renderUpdateForm() {
      if (!state.selected) {
        els.updateForm.innerHTML = '<p class="muted">Select a row to edit it.</p>';
        els.editHint.textContent = 'Select a row to open the change form.';
        return;
      }
      els.editHint.textContent = 'Editing record #' + recordPrimaryKey(state.selected.item) + '.';
      await renderForm(els.updateForm, state.meta?.update_fields || [], 'update', state.selected.item || {}, 'update');
    }

     function renderSelectedRecord() {
       els.detailFields.innerHTML = '';
       if (!state.selected) {
         els.detailTitle.textContent = 'No record selected';
         els.detailObjectBadge.textContent = 'Draft view';
         els.detail.textContent = 'No record selected.';
        els.detailFields.innerHTML = '<p class="muted">No record selected.</p>';
        highlightSelectedRow();
        return;
       }
       const record = state.selected.item || {};
       const recordID = recordPrimaryKey(record);
       els.detailTitle.textContent = state.meta.label + ' #' + recordID;
       els.detailObjectBadge.textContent = 'Record overview';
       els.detail.textContent = JSON.stringify(record, null, 2);
      const detailFields = state.meta?.detail_fields || Object.keys(record);
      detailFields.forEach((name) => {
        const row = document.createElement('div');
        row.className = 'detail-row';
        const label = document.createElement('div');
        label.className = 'detail-label';
        label.textContent = fieldMeta(name)?.label || name;
        const value = document.createElement('div');
        value.className = 'detail-value';
        value.textContent = formatValue(record[name]);
        row.appendChild(label);
        row.appendChild(value);
        els.detailFields.appendChild(row);
      });
    }

    function syncBulkActionState() {
      const count = selectedIDs().length;
      els.selectedCountBadge.textContent = count + ' selected';
      els.bulkDelete.disabled = count === 0 || !hasAction('bulk_delete');
      syncWorkspaceActionState();
    }

    function formPayload(form) {
      const payload = {};
      const data = new FormData(form);
      for (const [key, value] of data.entries()) {
        const field = fieldMeta(key);
        if (!field) continue;
        if (field.component === 'password' && value === '') {
          continue;
        }
        if (field.component === 'number') {
          payload[key] = value === '' ? null : Number(value);
          continue;
        }
        if (field.component === 'array') {
          payload[key] = value ? JSON.parse(value) : [];
          continue;
        }
        if (field.relation) {
          if (value === '') {
            payload[key] = null;
            continue;
          }
          payload[key] = numericFieldPattern.test(value) ? Number(value) : value;
          continue;
        }
        payload[key] = value;
      }
      form.querySelectorAll('input[type=checkbox][name]').forEach((checkbox) => {
        if (!fieldMeta(checkbox.name) || checkbox.disabled) return;
        payload[checkbox.name] = checkbox.checked;
      });
      return payload;
    }

    function buildListQuery() {
      const params = new URLSearchParams();
      if (els.search.value.trim()) {
        params.set('search', els.search.value.trim());
      }
      if (els.sort.value) {
        params.set('sort', els.sort.value);
      }
      params.set('page', String(state.pagination.page));
      params.set('size', String(state.pagination.size));
      (state.meta?.filter_fields || []).forEach((name) => {
        const field = fieldValue(name);
        if (!field) return;
        const value = String(field.value || '').trim();
        if (value !== '') {
          params.set(name, value);
        }
      });
      return '?' + params.toString();
    }

    function renderPagination() {
      els.paginationInfo.textContent = 'Page ' + state.pagination.page + ' of ' + state.pagination.pages + ' · ' + state.pagination.total + ' record(s)';
      els.prevPage.disabled = state.pagination.page <= 1;
      els.nextPage.disabled = state.pagination.page >= state.pagination.pages;
    }

    function highlightSelectedRow() {
      const selectedID = state.selected ? String(recordPrimaryKey(state.selected.item)) : '';
      els.list.querySelectorAll('tbody tr[data-record-id]').forEach((row) => {
        row.classList.toggle('row-selected', row.dataset.recordId === selectedID);
      });
    }

    async function renderList() {
      if (!state.current) return;
      const data = await request(currentBasePath() + buildListQuery());
      const fields = state.meta?.list_fields || [];
      const rows = data.items || [];
      state.records = rows;
      state.pagination = {
        page: data.page || 1,
        size: data.size || Number(els.pageSize.value || 10),
        pages: data.pages || 1,
        total: data.total || 0
      };
      renderPagination();
      if (!fields.length) {
        els.list.innerHTML = '<div class="empty-state">No list fields are available for this resource.</div>';
        return;
      }
      if (!rows.length) {
        els.list.innerHTML = '<div class="empty-state">No records matched the current filters.</div>';
        return;
      }
      const tableShell = document.createElement('div');
      tableShell.className = 'table-shell';
      const table = document.createElement('table');
      const thead = document.createElement('thead');
      const headRow = document.createElement('tr');
      const bulkCell = document.createElement('th');
      const selectAll = document.createElement('input');
      selectAll.type = 'checkbox';
      selectAll.checked = rows.length > 0 && rows.every((row) => isSelectedForBulk(recordPrimaryKey(row)));
      selectAll.onchange = () => {
        rows.forEach((row) => setSelectedForBulk(recordPrimaryKey(row), selectAll.checked));
        syncBulkActionState();
        renderList().catch((error) => setStatus(String(error.message || error)));
      };
      bulkCell.appendChild(selectAll);
      headRow.appendChild(bulkCell);
      fields.forEach((field) => {
        const th = document.createElement('th');
        th.textContent = field;
        headRow.appendChild(th);
      });
      const actionHead = document.createElement('th');
      actionHead.textContent = 'Actions';
      headRow.appendChild(actionHead);
      thead.appendChild(headRow);
      table.appendChild(thead);
      const tbody = document.createElement('tbody');
      rows.forEach((row) => {
        const tr = document.createElement('tr');
        const id = recordPrimaryKey(row);
        tr.dataset.recordId = String(id);
        const checkCell = document.createElement('td');
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.checked = isSelectedForBulk(id);
        checkbox.onchange = () => {
          setSelectedForBulk(id, checkbox.checked);
          syncBulkActionState();
        };
        checkCell.appendChild(checkbox);
        tr.appendChild(checkCell);
        fields.forEach((field) => {
          const td = document.createElement('td');
          td.textContent = formatValue(row[field]);
          tr.appendChild(td);
        });
        const actionCell = document.createElement('td');
        const actionWrap = document.createElement('div');
        actionWrap.className = 'action-cell';
        // View button
        const openButton = document.createElement('button');
        openButton.type = 'button';
        openButton.className = 'action-btn-view';
        openButton.textContent = 'View';
        openButton.onclick = () => selectRecord(row, { openModal: 'record' });
        actionWrap.appendChild(openButton);
        // More (···) dropdown menu
        const menuWrap = document.createElement('div');
        menuWrap.className = 'action-menu';
        const trigger = document.createElement('button');
        trigger.type = 'button';
        trigger.className = 'action-menu-trigger';
        trigger.setAttribute('aria-label', 'More actions');
        trigger.textContent = '···';
        const menuList = document.createElement('div');
        menuList.className = 'action-menu-list';
        const editItem = document.createElement('button');
        editItem.type = 'button';
        editItem.className = 'action-menu-item';
        editItem.textContent = 'Edit';
        editItem.disabled = !hasAction('update');
        editItem.onclick = () => { menuList.classList.remove('open'); selectRecord(row, { openModal: 'edit' }); };
        menuList.appendChild(editItem);
        const divider = document.createElement('hr');
        divider.className = 'action-menu-divider';
        menuList.appendChild(divider);
        const deleteItem = document.createElement('button');
        deleteItem.type = 'button';
        deleteItem.className = 'action-menu-item danger';
        deleteItem.textContent = 'Delete';
        deleteItem.disabled = !hasAction('delete');
        deleteItem.onclick = () => { menuList.classList.remove('open'); deleteRecordByID(id); };
        menuList.appendChild(deleteItem);
        trigger.onclick = (e) => {
          e.stopPropagation();
          const isOpen = menuList.classList.contains('open');
          document.querySelectorAll('.action-menu-list.open').forEach(m => m.classList.remove('open'));
          if (!isOpen) menuList.classList.add('open');
        };
        menuWrap.appendChild(trigger);
        menuWrap.appendChild(menuList);
        actionWrap.appendChild(menuWrap);
        actionCell.appendChild(actionWrap);
        tr.appendChild(actionCell);
        tbody.appendChild(tr);
      });
      table.appendChild(tbody);
      tableShell.appendChild(table);
      els.list.innerHTML = '';
      els.list.appendChild(tableShell);
      highlightSelectedRow();
    }

     async function selectRecord(row, options = {}) {
       try {
         const id = recordPrimaryKey(row);
         if (!id) {
          throw new Error('Selected row has no primary key.');
        }
         state.selected = await request(currentBasePath() + '/' + encodeURIComponent(String(id)));
         renderSelectedRecord();
         await renderUpdateForm();
         highlightSelectedRow();
         syncWorkspaceActionState();
         if (options.openModal === 'record') {
           openModal(els.recordModal);
         }
         if (options.openModal === 'edit') {
           closeModal(els.recordModal);
           openModal(els.editModal);
         }
         setStatus('Loaded record #' + id + '.');
        } catch (error) {
          setStatus(String(error.message || error));
        }
      }

      async function deleteRecordByID(id) {
        if (!state.current || id == null) return;
        try {
         await request(currentBasePath() + '/' + encodeURIComponent(String(id)), { method: 'DELETE' });
         if (state.selected && String(recordPrimaryKey(state.selected.item)) === String(id)) {
           state.selected = null;
           renderSelectedRecord();
           await renderUpdateForm();
         }
         setSelectedForBulk(id, false);
         closeModal(els.recordModal);
         closeModal(els.editModal);
         await reloadListWithStatus('Deleted record #' + id + '.', false);
        } catch (error) {
          setStatus(String(error.message || error));
        }
      }

     async function reloadListWithStatus(message, resetPage) {
      if (resetPage) resetToFirstPage();
      await renderList();
      syncBulkActionState();
      if (message) setStatus(message);
    }

    async function loadResources() {
      if (!hasToken()) {
        renderAuthState();
        setStatus('Sign in before loading admin resources.');
        return;
      }
      try {
        const payload = await request(apiBase + '/resources');
        state.resources = payload.resources || [];
        renderResources();
        setStatus('Loaded ' + state.resources.length + ' resources.');
        if (state.resources.length) {
          await selectResource(state.resources[0]);
        }
      } catch (error) {
        setStatus(String(error.message || error));
      }
    }

    async function selectResource(resource) {
      state.current = resource;
      state.selected = null;
      resetQueryState();
      try {
        state.meta = await request(currentBasePath() + '/meta');
        renderResources();
        els.resourceTitle.textContent = state.meta.label;
        renderResourceSummary();
        renderSortOptions();
        renderFilterControls();
        await Promise.all([renderCreateForm(), renderUpdateForm(), renderList()]);
        renderSelectedRecord();
        syncBulkActionState();
        syncWorkspaceActionState();
        setStatus('Loaded resource ' + resource.name + '.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    }

    els.token.addEventListener('input', () => {
      persistToken();
      if (!hasToken()) {
        resetAdminState();
      }
      renderAuthState();
    });
    els.loginForm.onsubmit = async (event) => {
      event.preventDefault();
      try {
        const payload = await request('/api/v1/auth/login', {
          method: 'POST',
          body: JSON.stringify({
            email: els.loginEmail.value.trim(),
            password: els.loginPassword.value
          }),
          skipAuthRedirect: true
        });
        if (!payload || !payload.token) {
          throw new Error('Login response did not include a token.');
        }
        state.auth = {
          name: payload.name || '',
          userID: payload.user_id || payload.userID || null
        };
        els.token.value = payload.token;
        persistToken();
        els.loginPassword.value = '';
        renderAuthState();
        const successMessage = state.auth.name ? ('Signed in as ' + state.auth.name + '.') : 'Signed in successfully.';
        if (isStandaloneLoginPage()) {
          redirectToAdmin(successMessage);
          return;
        }
        setStatus(successMessage);
        await loadResources();
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };
    els.clearToken.onclick = () => {
      logout(isStandaloneAdminPage() ? 'Signed out of the admin console.' : 'Signed out of the admin prototype.');
    };
     els.loadResources.onclick = loadResources;
     els.openCreateModal.onclick = () => {
       if (els.openCreateModal.disabled) return;
       openModal(els.createModal);
     };
     els.closeCreateModal.onclick = () => closeModal(els.createModal);
     els.closeRecordModal.onclick = () => closeModal(els.recordModal);
     els.closeEditModal.onclick = () => closeModal(els.editModal);
     [els.createModal, els.recordModal, els.editModal].forEach((modal) => {
       modal.addEventListener('click', (event) => {
         if (event.target === modal) {
           closeModal(modal);
        }
      });
     });
     document.addEventListener('keydown', (event) => {
       if (event.key !== 'Escape') return;
       document.querySelectorAll('.action-menu-list.open').forEach(m => m.classList.remove('open'));
       closeAllModals();
     });
     document.addEventListener('click', () => {
       document.querySelectorAll('.action-menu-list.open').forEach(m => m.classList.remove('open'));
     });
    els.reloadList.onclick = () => state.current && reloadListWithStatus('Reloaded list.', false).catch((error) => setStatus(String(error.message || error)));
     els.clearFilters.onclick = () => {
       if (!state.current) return;
       cancelScheduledSearchReload();
       els.search.value = '';
       els.sort.value = '';
       Array.from(els.filtersForm.elements).forEach((element) => {
         if ('value' in element) element.value = '';
      });
      reloadListWithStatus('Cleared filters.', true).catch((error) => setStatus(String(error.message || error)));
    };
     els.filtersForm.onsubmit = (event) => {
       event.preventDefault();
       cancelScheduledSearchReload();
       els.reloadList.click();
     };
     els.search.addEventListener('input', () => {
       if (!state.current) return;
       scheduleSearchReload();
     });
     els.search.onkeydown = (event) => {
       if (event.key === 'Enter') {
         event.preventDefault();
         cancelScheduledSearchReload();
         resetToFirstPage();
         els.reloadList.click();
       }
     };
     els.sort.onchange = () => {
       if (!state.current) return;
       cancelScheduledSearchReload();
       resetToFirstPage();
       els.reloadList.click();
     };
    els.pageSize.onchange = () => {
      state.pagination.size = Number(els.pageSize.value || 10);
      reloadListWithStatus('Updated page size.', true).catch((error) => setStatus(String(error.message || error)));
    };
     els.filtersForm.onchange = () => {
       if (!state.current) return;
       cancelScheduledSearchReload();
       resetToFirstPage();
       els.reloadList.click();
     };
    els.prevPage.onclick = () => {
      if (state.pagination.page <= 1) return;
      state.pagination.page -= 1;
      reloadListWithStatus('Loaded previous page.', false).catch((error) => setStatus(String(error.message || error)));
    };
    els.nextPage.onclick = () => {
      if (state.pagination.page >= state.pagination.pages) return;
      state.pagination.page += 1;
      reloadListWithStatus('Loaded next page.', false).catch((error) => setStatus(String(error.message || error)));
    };
    els.createForm.onsubmit = async (event) => {
      event.preventDefault();
      if (!state.current) return;
      try {
        await request(currentBasePath(), {
          method: 'POST',
          body: JSON.stringify(formPayload(els.createForm))
        });
        await renderCreateForm();
        closeModal(els.createModal);
        await reloadListWithStatus('Created a new ' + state.current.name + ' record.', true);
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };
    els.updateForm.onsubmit = async (event) => {
      event.preventDefault();
      if (!state.current || !state.selected) return;
      try {
        const id = recordPrimaryKey(state.selected.item);
        await request(currentBasePath() + '/' + encodeURIComponent(String(id)), {
          method: 'PUT',
          body: JSON.stringify(formPayload(els.updateForm))
        });
        closeModal(els.editModal);
        await renderList();
        await selectRecord({ id: id });
        setStatus('Updated record #' + id + '.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };
    els.bulkDelete.onclick = async () => {
      if (!state.current || !selectedIDs().length) return;
      try {
        const ids = selectedIDs();
        const result = await request(currentBasePath() + '/bulk-delete', {
          method: 'POST',
          body: JSON.stringify({ ids: ids })
        });
        if (state.selected && isSelectedForBulk(recordPrimaryKey(state.selected.item))) {
          state.selected = null;
          renderSelectedRecord();
          await renderUpdateForm();
        }
        state.bulkSelected = {};
        await reloadListWithStatus('Bulk deleted ' + String(result.deleted || 0) + ' record(s).', false);
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };

    resetAdminState();
    updatePageChrome();
    const restoredToken = restoreToken();
    const flashMessage = consumeFlashMessage();
    renderAuthState();
    if (flashMessage) {
      setStatus(flashMessage);
    }
    if (restoredToken) {
      if (!isStandaloneLoginPage()) {
        if (!flashMessage) {
          setStatus('Restored saved token.');
        }
        loadResources().catch((error) => setStatus(String(error.message || error)));
      }
    } else {
      if (!flashMessage) {
        setStatus('Ready. Sign in to continue.');
      }
    }
  </script>
</body>
</html>`
