package app

import "github.com/gin-gonic/gin"

// ServeAdminPrototype returns a minimal browser-based prototype for the admin APIs.
func ServeAdminPrototype(c *gin.Context) {
	c.Data(200, "text/html; charset=utf-8", []byte(adminPrototypeHTML))
}

const adminPrototypeHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Gin Ninja Admin Prototype</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 0; background: #f6f7fb; color: #1f2937; }
    header, main { max-width: 1280px; margin: 0 auto; padding: 16px; }
    .panel { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 16px; margin-bottom: 16px; }
    .grid { display: grid; gap: 16px; grid-template-columns: 260px 1fr; }
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
    @media (max-width: 960px) {
      .grid, .detail-layout { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <h1>Gin Ninja Admin Prototype</h1>
    <p class="muted">A minimal metadata-driven admin UI for the example admin APIs.</p>
  </header>
  <main class="stack">
    <section class="panel stack">
      <label>JWT token
        <input id="token" placeholder="Paste a Bearer token from /api/v1/auth/login" autocomplete="off">
      </label>
      <p class="muted">Token is stored in localStorage and attached to every admin request automatically.</p>
      <div class="row-actions">
        <button id="loadResources" type="button">Load resources</button>
        <button id="clearToken" type="button" class="secondary">Clear token</button>
      </div>
      <pre id="status">Ready.</pre>
    </section>
    <section class="grid">
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
    const state = {
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
      token: document.getElementById('token'),
      clearToken: document.getElementById('clearToken'),
      status: document.getElementById('status'),
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
        setStatus('Restored saved token.');
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
      persistToken();
      const response = await fetch(path, { ...options, headers: requestHeaders(options) });
      const text = await response.text();
      let data = null;
      try { data = text ? JSON.parse(text) : null; } catch (_) { data = text; }
      if (!response.ok) {
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

    els.token.addEventListener('input', persistToken);
    els.clearToken.onclick = () => {
      els.token.value = '';
      persistToken();
      setStatus('Cleared saved token.');
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

    restoreToken();
    renderPagination();
    syncBulkActionState();
  </script>
</body>
</html>`
