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
    body { font-family: system-ui, sans-serif; margin: 0; background: #f6f7fb; color: #1f2937; }
    header, main { max-width: 1280px; margin: 0 auto; padding: 16px; }
    .panel { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 16px; margin-bottom: 16px; }
    .grid { display: grid; gap: 16px; grid-template-columns: 260px 1fr; }
    .login-shell { display:grid; gap:16px; }
    .session-panel { position:relative; overflow:hidden; }
    .session-toolbar-copy { display:grid; gap:4px; }
    .login-marketing, .login-lead, .login-credentials { display:none; }
    .eyebrow { display:inline-flex; width:max-content; align-items:center; gap:6px; border-radius:999px; padding:6px 10px; background:#e0e7ff; color:#3730a3; font-size:12px; font-weight:700; letter-spacing:0.08em; text-transform:uppercase; }
    .login-marketing { align-content:start; }
    .login-marketing h2, .login-lead h2 { margin:0; line-height:1.1; }
    .login-marketing p, .login-lead p { margin:0; }
    .login-feature-list { display:grid; gap:12px; }
    .login-feature { display:grid; gap:4px; padding:16px 18px; background:rgba(255,255,255,0.08); border:1px solid rgba(191,219,254,0.18); border-radius:16px; }
    .login-feature strong { font-size:15px; }
    .login-credentials { gap:8px; padding:12px 14px; border-radius:12px; background:#f8fafc; border:1px solid #dbeafe; }
    .login-credentials code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size:13px; background:#fff; border:1px solid #dbeafe; border-radius:8px; padding:2px 6px; }
    .two-col { display: grid; gap: 16px; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); }
    .stack { display: grid; gap: 12px; }
    .toolbar { display:flex; gap:12px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    .row-actions { display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
    .filters { display:grid; gap:12px; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); }
    .inline-field { display:grid; gap:6px; font-size: 14px; }
    .field-help { font-size: 12px; color: #6b7280; }
    .relation-control { display:grid; gap:8px; }
    .relation-preview { display:grid; gap:6px; margin:0; padding:0; list-style:none; }
    .relation-preview li { font-size:12px; color:#374151; background:#f9fafb; border:1px solid #e5e7eb; border-radius:8px; padding:6px 8px; }
    .relation-preview mark { background:#fef3c7; padding:0; }
    .detail-layout { display:grid; gap:16px; grid-template-columns: minmax(0, 1.2fr) minmax(280px, 0.8fr); }
    .detail-card { border:1px solid #e5e7eb; border-radius:10px; padding:16px; background:#fff; }
    .detail-grid { display:grid; gap:10px; }
    .detail-row { display:grid; grid-template-columns: 180px 1fr; gap:12px; border-bottom:1px solid #f3f4f6; padding-bottom:10px; }
    .detail-row:last-child { border-bottom:none; padding-bottom:0; }
    .detail-label { font-size:12px; font-weight:600; color:#6b7280; text-transform:uppercase; letter-spacing:0.04em; }
    .detail-value { font-size:14px; word-break:break-word; }
    .badge { display:inline-flex; align-items:center; gap:6px; font-size:12px; background:#eef2ff; color:#3730a3; border-radius:999px; padding:6px 10px; }
    .bulk-edit-fields { display:grid; gap:12px; }
    .bulk-edit-field { border:1px solid #e5e7eb; border-radius:10px; padding:12px; background:#f9fafb; }
    .pagination-bar { display:flex; gap:12px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    .pagination-info { font-size:14px; color:#6b7280; }
    label { display:grid; gap:6px; font-size:14px; }
    input, select, textarea, button { font: inherit; padding: 10px 12px; border-radius: 8px; border: 1px solid #d1d5db; }
    textarea { min-height: 96px; }
    button { cursor: pointer; background: #111827; color: #fff; }
    button.secondary { background: #fff; color: #111827; }
    button.danger { background: #b91c1c; }
    button:disabled, input:disabled, select:disabled, textarea:disabled { opacity: 0.6; cursor: not-allowed; }
    ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
    li button { width: 100%; text-align: left; }
    table { width: 100%; border-collapse: collapse; }
    th, td { border-bottom: 1px solid #e5e7eb; padding: 8px; text-align: left; font-size: 14px; vertical-align: top; }
    pre { margin: 0; white-space: pre-wrap; word-break: break-word; background: #111827; color: #f9fafb; padding: 12px; border-radius: 8px; }
    .muted { color: #6b7280; font-size: 14px; }
    body.standalone-login-page { background:
      radial-gradient(circle at top left, rgba(59,130,246,0.18), transparent 36%),
      radial-gradient(circle at bottom right, rgba(99,102,241,0.22), transparent 34%),
      linear-gradient(180deg, #eef4ff 0%, #f8fafc 45%, #eef2ff 100%);
    }
    body.standalone-login-page header,
    body.standalone-login-page main { max-width: 1120px; padding: 24px; }
    body.standalone-login-page header { padding-top: 48px; }
    body.standalone-login-page .login-shell { gap:24px; grid-template-columns: minmax(0, 1.1fr) minmax(360px, 420px); align-items:stretch; }
    body.standalone-login-page .login-marketing { display:grid; gap:18px; color:#e5eefb; background:linear-gradient(135deg, #0f172a 0%, #1d4ed8 100%); border-radius:24px; padding:32px; box-shadow:0 24px 60px rgba(15, 23, 42, 0.22); }
    body.standalone-login-page .login-lead,
    body.standalone-login-page .login-credentials { display:grid; }
    body.standalone-login-page .session-panel { margin:0; padding:28px; border-color:#dbeafe; border-radius:24px; box-shadow:0 24px 60px rgba(15, 23, 42, 0.12); }
    body.standalone-login-page #sessionToolbar { display:none; }
    body.standalone-login-page #loginForm { grid-template-columns:1fr; gap:14px; }
    body.standalone-login-page #loginForm label { font-size:13px; font-weight:600; color:#475569; }
    body.standalone-login-page input { min-height:48px; border-color:#cbd5e1; background:#fff; }
    body.standalone-login-page button { min-height:48px; }
    body.standalone-login-page #loginButton { width:100%; background:linear-gradient(135deg, #111827 0%, #1d4ed8 100%); box-shadow:0 12px 24px rgba(29, 78, 216, 0.2); }
    body.standalone-login-page #status { background:#0f172a; border:1px solid #0f172a; }
    body.standalone-login-page #authBadge { background:#eff6ff; color:#1d4ed8; }
    @media (max-width: 960px) {
      .grid, .detail-layout, body.standalone-login-page .login-shell { grid-template-columns: 1fr; }
      body.standalone-login-page header,
      body.standalone-login-page main { padding: 16px; }
      body.standalone-login-page header { padding-top: 24px; }
    }
  </style>
</head>
<body>
  <header>
    <h1 id="pageTitle">Gin Ninja Admin</h1>
    <p id="pageIntro" class="muted">A metadata-driven admin UI for the example admin APIs.</p>
  </header>
  <main class="stack">
    <section class="login-shell">
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
      <div id="sessionToolbar" class="toolbar">
        <div>
          <h2 style="margin:0;">Admin session</h2>
          <p id="authDescription" class="muted">Sign in to access the example admin APIs.</p>
        </div>
        <span id="authBadge" class="badge">Logged out</span>
      </div>
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
      <div id="sessionActions" class="row-actions" hidden>
        <button id="loadResources" type="button">Load resources</button>
        <button id="clearToken" type="button" class="secondary">Logout</button>
      </div>
      <div id="manualTokenTools" class="stack">
        <label>JWT token
          <input id="token" placeholder="Paste a Bearer token from /api/v1/auth/login" autocomplete="off">
        </label>
        <p class="muted">Successful sign-in stores the JWT in localStorage and attaches it to every admin request automatically.</p>
      </div>
      <pre id="status">Ready.</pre>
      </section>
    </section>
    <section id="adminShell" class="grid" hidden>
      <aside class="panel">
        <h2>Resources</h2>
        <ul id="resources"></ul>
      </aside>
      <section class="stack">
        <section class="panel">
          <div class="toolbar">
            <div>
              <h2 id="resourceTitle">Select a resource</h2>
              <p id="resourcePath" class="muted"></p>
            </div>
            <span id="selectedCountBadge" class="badge">0 selected</span>
          </div>
          <div id="actions" class="muted"></div>
        </section>
        <section class="panel stack">
          <div class="toolbar">
            <div>
              <h3 style="margin:0;">Change form</h3>
              <p id="selectionHint" class="muted">Select a row to inspect and edit it.</p>
            </div>
            <button id="deleteRecord" class="danger" type="button">Delete</button>
          </div>
          <div class="detail-layout">
            <section class="stack">
              <div class="detail-card stack">
                <div class="toolbar">
                  <strong id="detailTitle">No record selected</strong>
                  <span id="detailObjectBadge" class="badge">Draft view</span>
                </div>
                <div id="detailFields" class="detail-grid">
                  <p class="muted">No record selected.</p>
                </div>
              </div>
              <div class="detail-card stack">
                <strong>Raw payload</strong>
                <pre id="detail">No record selected.</pre>
              </div>
            </section>
            <section class="detail-card stack">
              <div>
                <h4 style="margin:0 0 4px 0;">Edit record</h4>
                <p class="muted" id="editHint">Select a row to open the change form.</p>
              </div>
              <form id="updateForm" class="stack"></form>
            </section>
          </div>
        </section>
        <section class="two-col">
          <section class="panel">
            <h3>Create record</h3>
            <form id="createForm" class="stack"></form>
          </section>
          <section class="panel stack">
            <div class="toolbar">
              <div>
                <h3 style="margin:0;">Bulk edit</h3>
                <p id="bulkEditHint" class="muted">Select rows to apply shared updates.</p>
              </div>
              <button id="applyBulkEdit" type="submit" form="bulkEditForm">Apply to selected</button>
            </div>
            <form id="bulkEditForm" class="bulk-edit-fields"></form>
          </section>
        </section>
        <section class="panel stack">
          <div class="toolbar">
            <h3 style="margin:0;">List records</h3>
            <div class="row-actions">
              <input id="search" placeholder="Search current resource">
              <select id="sort"></select>
              <select id="pageSize">
                <option value="5">5 / page</option>
                <option value="10" selected>10 / page</option>
                <option value="20">20 / page</option>
                <option value="50">50 / page</option>
              </select>
              <button id="reloadList" class="secondary" type="button">Reload list</button>
              <button id="clearFilters" class="secondary" type="button">Clear filters</button>
              <button id="bulkDelete" class="danger" type="button">Bulk delete</button>
            </div>
          </div>
          <form id="filtersForm" class="filters"></form>
          <div class="pagination-bar">
            <div id="paginationInfo" class="pagination-info">Page 1 of 1</div>
            <div class="row-actions">
              <button id="prevPage" class="secondary" type="button">Previous</button>
              <button id="nextPage" class="secondary" type="button">Next</button>
            </div>
          </div>
          <div id="list"></div>
        </section>
      </section>
    </section>
  </main>
  <script>
    const apiBase = '/api/v1/admin';
    const tokenStorageKey = 'gin-ninja-admin-token';
    const flashStorageKey = 'gin-ninja-admin-flash';
    const adminPagePath = '/admin';
    const adminLoginPath = '/admin/login';
    const prototypePagePath = '/admin-prototype';
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
      pagination: { page: 1, size: 10, pages: 1, total: 0 }
    };

     const els = {
      loginForm: document.getElementById('loginForm'),
      loginEmail: document.getElementById('loginEmail'),
      loginPassword: document.getElementById('loginPassword'),
      token: document.getElementById('token'),
      manualTokenTools: document.getElementById('manualTokenTools'),
      clearToken: document.getElementById('clearToken'),
      authBadge: document.getElementById('authBadge'),
      authDescription: document.getElementById('authDescription'),
      sessionActions: document.getElementById('sessionActions'),
      status: document.getElementById('status'),
      pageTitle: document.getElementById('pageTitle'),
      pageIntro: document.getElementById('pageIntro'),
      adminShell: document.getElementById('adminShell'),
      resources: document.getElementById('resources'),
      resourceTitle: document.getElementById('resourceTitle'),
      resourcePath: document.getElementById('resourcePath'),
      actions: document.getElementById('actions'),
      selectedCountBadge: document.getElementById('selectedCountBadge'),
      detailTitle: document.getElementById('detailTitle'),
      detailObjectBadge: document.getElementById('detailObjectBadge'),
      detailFields: document.getElementById('detailFields'),
      createForm: document.getElementById('createForm'),
      updateForm: document.getElementById('updateForm'),
      bulkEditForm: document.getElementById('bulkEditForm'),
      applyBulkEdit: document.getElementById('applyBulkEdit'),
      bulkEditHint: document.getElementById('bulkEditHint'),
      editHint: document.getElementById('editHint'),
      filtersForm: document.getElementById('filtersForm'),
      sort: document.getElementById('sort'),
      pageSize: document.getElementById('pageSize'),
      paginationInfo: document.getElementById('paginationInfo'),
      prevPage: document.getElementById('prevPage'),
      nextPage: document.getElementById('nextPage'),
      list: document.getElementById('list'),
      detail: document.getElementById('detail'),
      selectionHint: document.getElementById('selectionHint'),
      loadResources: document.getElementById('loadResources'),
      reloadList: document.getElementById('reloadList'),
      clearFilters: document.getElementById('clearFilters'),
      deleteRecord: document.getElementById('deleteRecord'),
      bulkDelete: document.getElementById('bulkDelete'),
      search: document.getElementById('search')
    };

    function setStatus(value) {
      els.status.textContent = value;
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
        els.pageTitle.textContent = 'Gin Ninja Admin Login';
        els.pageIntro.textContent = 'Sign in to enter the example admin console.';
        return;
      }
      if (isStandaloneAdminPage()) {
        document.title = 'Gin Ninja Admin';
        els.pageTitle.textContent = 'Gin Ninja Admin';
        els.pageIntro.textContent = 'A standalone admin console backed by the example admin APIs.';
        return;
      }
      document.title = 'Gin Ninja Admin Prototype';
      els.pageTitle.textContent = 'Gin Ninja Admin Prototype';
      els.pageIntro.textContent = 'A minimal metadata-driven admin UI for the example admin APIs.';
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
      const standaloneAdminPage = isStandaloneAdminPage();
      const standaloneLoginPage = isStandaloneLoginPage();
      els.loginForm.hidden = standaloneAdminPage;
      els.sessionActions.hidden = true;
      els.manualTokenTools.hidden = standaloneLoginPage;
      els.adminShell.hidden = true;
      els.authBadge.textContent = 'Logged out';
      els.authDescription.textContent = standaloneAdminPage
        ? 'Redirecting to /admin/login…'
        : 'Sign in to access the example admin APIs.';
    }

    function renderSignedInState() {
      const standaloneLoginPage = isStandaloneLoginPage();
      els.loginForm.hidden = true;
      els.sessionActions.hidden = standaloneLoginPage;
      els.manualTokenTools.hidden = standaloneLoginPage;
      els.adminShell.hidden = standaloneLoginPage;
      els.authBadge.textContent = state.auth.name ? ('Signed in as ' + state.auth.name) : 'Authenticated';
      els.authDescription.textContent = state.auth.name
        ? ('JWT session ready for ' + state.auth.name + '.')
        : 'JWT session ready for the example admin APIs.';
    }

    function renderAuthState() {
      if (hasToken()) {
        renderSignedInState();
        if (isStandaloneLoginPage()) {
          redirectToAdmin('Restored saved token. Redirecting to /admin.');
        }
      } else {
        renderSignedOutState();
        if (isStandaloneAdminPage()) {
          redirectToLogin('Sign in to continue.');
        }
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
      els.resources.innerHTML = '';
      els.resourceTitle.textContent = 'Select a resource';
      els.resourcePath.textContent = '';
      els.actions.textContent = 'Sign in to load admin resources.';
      els.detailTitle.textContent = 'No record selected';
      els.detailObjectBadge.textContent = 'Draft view';
      els.detailFields.innerHTML = '<p class="muted">No record selected.</p>';
      els.detail.textContent = 'No record selected.';
      els.createForm.innerHTML = '<p class="muted">Sign in to create records.</p>';
      els.updateForm.innerHTML = '<p class="muted">Sign in to edit records.</p>';
      els.bulkEditForm.innerHTML = '<p class="muted">Sign in to apply bulk edits.</p>';
      els.filtersForm.innerHTML = '';
      els.sort.innerHTML = '';
      els.list.innerHTML = '<p class="muted">Sign in to browse records.</p>';
      els.selectionHint.textContent = 'Sign in to inspect and edit records.';
      els.editHint.textContent = 'Sign in to open the change form.';
      els.bulkEditHint.textContent = 'Sign in to apply shared updates.';
      renderPagination();
      syncBulkActionState();
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

    function renderResources() {
      els.resources.innerHTML = '';
      state.resources.forEach((resource) => {
        const li = document.createElement('li');
        const button = document.createElement('button');
        button.type = 'button';
        button.textContent = resource.label + ' (' + resource.name + ')';
        button.onclick = () => selectResource(resource);
        li.appendChild(button);
        els.resources.appendChild(li);
      });
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
      if (!items.length) {
        preview.innerHTML = '<li>No matching options.</li>';
        return;
      }
      preview.innerHTML = items.slice(0, 5).map((item) => '<li>' + highlightMatch(item.label, term) + '</li>').join('');
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

    function populateRelationSelect(select, items, selectedValue) {
      select.innerHTML = '';
      items.forEach((item) => {
        const option = document.createElement('option');
        option.value = String(item.value);
        option.textContent = item.label;
        if (String(selectedValue ?? '') === String(item.value)) {
          option.selected = true;
        }
        select.appendChild(option);
      });
      if (selectedValue != null && selectedValue !== '' && !Array.from(select.options).some((option) => option.value === String(selectedValue))) {
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

    function scheduleRelationSearch(field, scopeKey, searchInput, select, preview, selectedValue) {
      const key = relationStateKey(scopeKey, field);
      clearTimeout(state.relationTimers[key]);
      state.relationTimers[key] = setTimeout(async () => {
        try {
          const term = searchInput.value.trim();
          const items = await loadRelationOptions(field, term, 1, 8);
          populateRelationSelect(select, items, selectedValue);
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
        help.textContent = 'Debounced search updates /fields/' + field.name + '/options and highlights matches below.';
        wrapper.appendChild(searchInput);
        wrapper.appendChild(select);
        wrapper.appendChild(preview);
        wrapper.appendChild(help);
        const items = await loadRelationOptions(field, searchInput.value.trim(), 1, 8);
        populateRelationSelect(select, items, value);
        updateRelationPreview(preview, items, searchInput.value.trim());
        searchInput.addEventListener('input', () => {
          state.relationSearch[searchKey] = searchInput.value;
          scheduleRelationSearch(field, scopeKey, searchInput, select, preview, value);
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

    async function renderBulkEditForm() {
      els.bulkEditForm.innerHTML = '';
      const fields = state.meta?.update_fields || [];
      if (!fields.length) {
        els.bulkEditForm.innerHTML = '<p class="muted">Bulk edit is not available for this resource.</p>';
        return;
      }
      for (const name of fields) {
        const field = fieldMeta(name);
        if (!field) continue;
        const row = document.createElement('div');
        row.className = 'bulk-edit-field stack';
        const rowHeader = document.createElement('div');
        rowHeader.className = 'toolbar';
        const toggleLabel = document.createElement('label');
        toggleLabel.style.display = 'flex';
        toggleLabel.style.gap = '8px';
        toggleLabel.style.alignItems = 'center';
        const toggle = document.createElement('input');
        toggle.type = 'checkbox';
        toggle.name = '__apply__' + field.name;
        toggle.style.width = '16px';
        toggle.style.height = '16px';
        const text = document.createElement('span');
        text.textContent = 'Apply ' + field.label;
        toggleLabel.appendChild(toggle);
        toggleLabel.appendChild(text);
        const hint = document.createElement('span');
        hint.className = 'field-help';
        hint.textContent = 'Checked fields are sent to every selected row.';
        rowHeader.appendChild(toggleLabel);
        rowHeader.appendChild(hint);
        const control = await buildFieldControl(field, '', 'bulk');
        setControlDisabled(control, true);
        toggle.onchange = () => {
          setControlDisabled(control, !toggle.checked);
        };
        row.appendChild(rowHeader);
        row.appendChild(control);
        els.bulkEditForm.appendChild(row);
      }
    }

    function renderSelectedRecord() {
      els.deleteRecord.disabled = !state.selected || !hasAction('delete');
      els.detailFields.innerHTML = '';
      if (!state.selected) {
        els.selectionHint.textContent = 'Select a row to inspect and edit it.';
        els.detailTitle.textContent = 'No record selected';
        els.detailObjectBadge.textContent = 'Draft view';
        els.detail.textContent = 'No record selected.';
        els.detailFields.innerHTML = '<p class="muted">No record selected.</p>';
        return;
      }
      const record = state.selected.item || {};
      const recordID = recordPrimaryKey(record);
      els.selectionHint.textContent = 'Reviewing record #' + recordID + ' in a Django-style change form layout.';
      els.detailTitle.textContent = state.meta.label + ' #' + recordID;
      els.detailObjectBadge.textContent = 'Object detail';
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
      els.applyBulkEdit.disabled = count === 0 || !hasAction('update');
      els.bulkEditHint.textContent = count
        ? ('Applying changes to ' + count + ' selected record(s).')
        : 'Select rows to apply shared updates.';
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
      els.paginationInfo.textContent = 'Page ' + state.pagination.page + ' of ' + state.pagination.pages + ' · ' + state.pagination.total + ' total row(s)';
      els.prevPage.disabled = state.pagination.page <= 1;
      els.nextPage.disabled = state.pagination.page >= state.pagination.pages;
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
        els.list.innerHTML = '<p class="muted">No list fields available.</p>';
        return;
      }
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
      const actionHead = document.createElement('th');
      actionHead.textContent = 'Actions';
      headRow.appendChild(actionHead);
      fields.forEach((field) => {
        const th = document.createElement('th');
        th.textContent = field;
        headRow.appendChild(th);
      });
      thead.appendChild(headRow);
      table.appendChild(thead);
      const tbody = document.createElement('tbody');
      rows.forEach((row) => {
        const tr = document.createElement('tr');
        const id = recordPrimaryKey(row);
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
        const actionCell = document.createElement('td');
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'secondary';
        button.textContent = 'View';
        button.onclick = () => selectRecord(row);
        actionCell.appendChild(button);
        tr.appendChild(actionCell);
        fields.forEach((field) => {
          const td = document.createElement('td');
          td.textContent = formatValue(row[field]);
          tr.appendChild(td);
        });
        tbody.appendChild(tr);
      });
      table.appendChild(tbody);
      els.list.innerHTML = '';
      els.list.appendChild(table);
    }

    async function selectRecord(row) {
      try {
        const id = recordPrimaryKey(row);
        if (!id) {
          throw new Error('Selected row has no primary key.');
        }
        state.selected = await request(currentBasePath() + '/' + encodeURIComponent(String(id)));
        renderSelectedRecord();
        await renderUpdateForm();
        setStatus('Loaded record #' + id + '.');
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
        els.resourceTitle.textContent = state.meta.label;
        els.resourcePath.textContent = currentBasePath();
        els.actions.textContent = 'Actions: ' + (state.meta.actions || []).join(', ');
        renderSortOptions();
        renderFilterControls();
        await Promise.all([renderCreateForm(), renderUpdateForm(), renderBulkEditForm(), renderList()]);
        renderSelectedRecord();
        syncBulkActionState();
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
    els.reloadList.onclick = () => state.current && reloadListWithStatus('Reloaded list.', false).catch((error) => setStatus(String(error.message || error)));
    els.clearFilters.onclick = () => {
      if (!state.current) return;
      els.search.value = '';
      els.sort.value = '';
      Array.from(els.filtersForm.elements).forEach((element) => {
        if ('value' in element) element.value = '';
      });
      reloadListWithStatus('Cleared filters.', true).catch((error) => setStatus(String(error.message || error)));
    };
    els.filtersForm.onsubmit = (event) => {
      event.preventDefault();
      els.reloadList.click();
    };
    els.search.onkeydown = (event) => {
      if (event.key === 'Enter') {
        event.preventDefault();
        resetToFirstPage();
        els.reloadList.click();
      }
    };
    els.sort.onchange = () => {
      if (!state.current) return;
      resetToFirstPage();
      els.reloadList.click();
    };
    els.pageSize.onchange = () => {
      state.pagination.size = Number(els.pageSize.value || 10);
      reloadListWithStatus('Updated page size.', true).catch((error) => setStatus(String(error.message || error)));
    };
    els.filtersForm.onchange = () => {
      if (!state.current) return;
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
        await renderList();
        await selectRecord({ id: id });
        setStatus('Updated record #' + id + '.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };
    els.bulkEditForm.onsubmit = async (event) => {
      event.preventDefault();
      if (!state.current || !selectedIDs().length) return;
      const payload = formPayload(els.bulkEditForm);
      if (!Object.keys(payload).length) {
        setStatus('Choose at least one field before bulk editing.');
        return;
      }
      try {
        for (const id of selectedIDs()) {
          await request(currentBasePath() + '/' + encodeURIComponent(String(id)), {
            method: 'PUT',
            body: JSON.stringify(payload)
          });
        }
        if (state.selected && isSelectedForBulk(recordPrimaryKey(state.selected.item))) {
          await selectRecord({ id: recordPrimaryKey(state.selected.item) });
        }
        await renderBulkEditForm();
        await reloadListWithStatus('Bulk updated ' + selectedIDs().length + ' record(s).', false);
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };
    els.deleteRecord.onclick = async () => {
      if (!state.current || !state.selected) return;
      try {
        const id = recordPrimaryKey(state.selected.item);
        await request(currentBasePath() + '/' + encodeURIComponent(String(id)), { method: 'DELETE' });
        state.selected = null;
        setSelectedForBulk(id, false);
        renderSelectedRecord();
        await renderUpdateForm();
        await reloadListWithStatus('Deleted record #' + id + '.', false);
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
