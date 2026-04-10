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
    header, main { max-width: 1200px; margin: 0 auto; padding: 16px; }
    .panel { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 16px; margin-bottom: 16px; }
    .grid { display: grid; gap: 16px; grid-template-columns: 260px 1fr; }
    .two-col { display: grid; gap: 16px; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); }
    .stack { display: grid; gap: 12px; }
    .toolbar { display:flex; gap:12px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    .row-actions { display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
    .filters { display:grid; gap:12px; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); }
    .inline-field { display:grid; gap:6px; font-size: 14px; }
    .field-help { font-size: 12px; color: #6b7280; }
    .relation-control { display:grid; gap:8px; }
    label { display: grid; gap: 6px; font-size: 14px; }
    input, select, textarea, button { font: inherit; padding: 10px 12px; border-radius: 8px; border: 1px solid #d1d5db; }
    textarea { min-height: 96px; }
    button { cursor: pointer; background: #111827; color: #fff; }
    button.secondary { background: #fff; color: #111827; }
    button.danger { background: #b91c1c; }
    button:disabled { opacity: 0.6; cursor: not-allowed; }
    ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
    li button { width: 100%; text-align: left; }
    table { width: 100%; border-collapse: collapse; }
    th, td { border-bottom: 1px solid #e5e7eb; padding: 8px; text-align: left; font-size: 14px; vertical-align: top; }
    pre { margin: 0; white-space: pre-wrap; word-break: break-word; background: #111827; color: #f9fafb; padding: 12px; border-radius: 8px; }
    .muted { color: #6b7280; font-size: 14px; }
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
          <h2 id="resourceTitle">Select a resource</h2>
          <p id="resourcePath" class="muted"></p>
          <div id="actions" class="muted"></div>
        </section>
        <section class="two-col">
          <section class="panel">
            <h3>Create record</h3>
            <form id="createForm" class="stack"></form>
          </section>
          <section class="panel">
            <div class="toolbar">
              <h3 style="margin:0;">Selected record</h3>
              <button id="deleteRecord" class="danger" type="button">Delete</button>
            </div>
            <p id="selectionHint" class="muted">Select a row to inspect and edit it.</p>
            <pre id="detail">No record selected.</pre>
            <h4>Edit record</h4>
            <form id="updateForm" class="stack"></form>
          </section>
        </section>
        <section class="panel stack">
          <div class="toolbar">
            <h3 style="margin:0;">List records</h3>
            <div class="row-actions">
              <input id="search" placeholder="Search current resource">
              <select id="sort"></select>
              <button id="reloadList" class="secondary" type="button">Reload list</button>
              <button id="clearFilters" class="secondary" type="button">Clear filters</button>
              <button id="bulkDelete" class="danger" type="button">Bulk delete</button>
            </div>
          </div>
          <form id="filtersForm" class="filters"></form>
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
      bulkSelected: [],
      relationSearch: {},
      relationTimers: {}
    };

    const els = {
      token: document.getElementById('token'),
      clearToken: document.getElementById('clearToken'),
      status: document.getElementById('status'),
      resources: document.getElementById('resources'),
      resourceTitle: document.getElementById('resourceTitle'),
      resourcePath: document.getElementById('resourcePath'),
      actions: document.getElementById('actions'),
      createForm: document.getElementById('createForm'),
      updateForm: document.getElementById('updateForm'),
      filtersForm: document.getElementById('filtersForm'),
      sort: document.getElementById('sort'),
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

    function resetQueryState() {
      state.bulkSelected = [];
      state.relationSearch = {};
      els.search.value = '';
      els.sort.innerHTML = '';
      els.filtersForm.innerHTML = '';
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
        await Promise.all([renderCreateForm(), renderUpdateForm(), renderList()]);
        renderSelectedRecord();
        setStatus('Loaded resource ' + resource.name + '.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    }

    async function loadRelationOptions(field, search) {
      const suffix = search ? ('?search=' + encodeURIComponent(search)) : '';
      const options = await request(currentBasePath() + '/fields/' + field.name + '/options' + suffix);
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

    function scheduleRelationSearch(field, searchInput, select, selectedValue) {
      const key = field.name + ':' + select.name;
      clearTimeout(state.relationTimers[key]);
      state.relationTimers[key] = setTimeout(async () => {
        try {
          const items = await loadRelationOptions(field, searchInput.value.trim());
          populateRelationSelect(select, items, selectedValue);
          setStatus('Loaded ' + items.length + ' relation option(s) for ' + field.name + '.');
        } catch (error) {
          setStatus(String(error.message || error));
        }
      }, 250);
    }

    async function buildFieldControl(field, value) {
      if (field.relation) {
        const wrapper = document.createElement('div');
        wrapper.className = 'relation-control';
        const searchInput = document.createElement('input');
        searchInput.type = 'text';
        searchInput.placeholder = 'Search related ' + field.label;
        searchInput.value = state.relationSearch[field.name] || '';
        const select = document.createElement('select');
        select.name = field.name;
        const help = document.createElement('div');
        help.className = 'field-help';
        help.textContent = 'Search updates the dropdown via /fields/' + field.name + '/options.';
        wrapper.appendChild(searchInput);
        wrapper.appendChild(select);
        wrapper.appendChild(help);
        const items = await loadRelationOptions(field, searchInput.value.trim());
        populateRelationSelect(select, items, value);
        searchInput.addEventListener('input', () => {
          state.relationSearch[field.name] = searchInput.value;
          scheduleRelationSearch(field, searchInput, select, value);
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

    async function renderForm(target, fieldNames, mode, values = {}) {
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
        const control = await buildFieldControl(field, values[name]);
        wrapper.appendChild(control);
        target.appendChild(wrapper);
      }
      const submit = document.createElement('button');
      submit.type = 'submit';
      submit.textContent = mode === 'update' ? 'Update' : 'Create';
      target.appendChild(submit);
    }

    async function renderCreateForm() {
      await renderForm(els.createForm, state.meta?.create_fields || [], 'create');
    }

    async function renderUpdateForm() {
      if (!state.selected) {
        els.updateForm.innerHTML = '<p class="muted">Select a row to edit it.</p>';
        return;
      }
      await renderForm(els.updateForm, state.meta?.update_fields || [], 'update', state.selected.item || {});
    }

    function renderSelectedRecord() {
      els.deleteRecord.disabled = !state.selected || !hasAction('delete');
      if (!state.selected) {
        els.selectionHint.textContent = 'Select a row to inspect and edit it.';
        els.detail.textContent = 'No record selected.';
        return;
      }
      els.selectionHint.textContent = 'Editing record #' + recordPrimaryKey(state.selected.item);
      els.detail.textContent = JSON.stringify(state.selected.item, null, 2);
    }

    function syncBulkDeleteState() {
      els.bulkDelete.disabled = !state.bulkSelected.length || !hasAction('bulk_delete');
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
      (state.meta?.filter_fields || []).forEach((name) => {
        const field = fieldValue(name);
        if (!field) return;
        const value = String(field.value || '').trim();
        if (value !== '') {
          params.set(name, value);
        }
      });
      const query = params.toString();
      return query ? ('?' + query) : '';
    }

    async function renderList() {
      if (!state.current) return;
      const data = await request(currentBasePath() + buildListQuery());
      const fields = state.meta?.list_fields || [];
      const rows = data.items || [];
      state.records = rows;
      state.bulkSelected = state.bulkSelected.filter((id) => rows.some((row) => String(recordPrimaryKey(row)) === id));
      syncBulkDeleteState();
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
      selectAll.checked = rows.length > 0 && rows.every((row) => state.bulkSelected.includes(String(recordPrimaryKey(row))));
      selectAll.onchange = () => {
        if (selectAll.checked) {
          state.bulkSelected = rows.map((row) => String(recordPrimaryKey(row)));
        } else {
          state.bulkSelected = [];
        }
        syncBulkDeleteState();
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
        const id = String(recordPrimaryKey(row));
        const checkCell = document.createElement('td');
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.checked = state.bulkSelected.includes(id);
        checkbox.onchange = () => {
          state.bulkSelected = checkbox.checked
            ? Array.from(new Set(state.bulkSelected.concat(id)))
            : state.bulkSelected.filter((value) => value !== id);
          syncBulkDeleteState();
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
          td.textContent = String(row[field] ?? '');
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

    async function reloadListWithStatus(message) {
      await renderList();
      syncBulkDeleteState();
      if (message) setStatus(message);
    }

    els.token.addEventListener('input', persistToken);
    els.clearToken.onclick = () => {
      els.token.value = '';
      persistToken();
      setStatus('Cleared saved token.');
    };
    els.loadResources.onclick = loadResources;
    els.reloadList.onclick = () => state.current && reloadListWithStatus('Reloaded list.').catch((error) => setStatus(String(error.message || error)));
    els.clearFilters.onclick = () => {
      if (!state.current) return;
      els.search.value = '';
      els.sort.value = '';
      Array.from(els.filtersForm.elements).forEach((element) => {
        if ('value' in element) element.value = '';
      });
      reloadListWithStatus('Cleared filters.').catch((error) => setStatus(String(error.message || error)));
    };
    els.filtersForm.onsubmit = (event) => {
      event.preventDefault();
      els.reloadList.click();
    };
    els.search.onkeydown = (event) => {
      if (event.key === 'Enter') {
        event.preventDefault();
        els.reloadList.click();
      }
    };
    els.sort.onchange = () => state.current && els.reloadList.click();
    els.filtersForm.onchange = () => state.current && els.reloadList.click();
    els.createForm.onsubmit = async (event) => {
      event.preventDefault();
      if (!state.current) return;
      try {
        await request(currentBasePath(), {
          method: 'POST',
          body: JSON.stringify(formPayload(els.createForm))
        });
        await renderCreateForm();
        await reloadListWithStatus('Created a new ' + state.current.name + ' record.');
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
        await selectRecord({ id });
        setStatus('Updated record #' + id + '.');
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
        state.bulkSelected = state.bulkSelected.filter((value) => value !== String(id));
        renderSelectedRecord();
        await renderUpdateForm();
        await reloadListWithStatus('Deleted record #' + id + '.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };
    els.bulkDelete.onclick = async () => {
      if (!state.current || !state.bulkSelected.length) return;
      try {
        const ids = state.bulkSelected
          .map((id) => state.records.find((row) => String(recordPrimaryKey(row)) === id))
          .filter(Boolean)
          .map((row) => recordPrimaryKey(row));
        const result = await request(currentBasePath() + '/bulk-delete', {
          method: 'POST',
          body: JSON.stringify({ ids: ids })
        });
        if (state.selected && ids.includes(recordPrimaryKey(state.selected.item))) {
          state.selected = null;
          renderSelectedRecord();
          await renderUpdateForm();
        }
        state.bulkSelected = [];
        await reloadListWithStatus('Bulk deleted ' + String(result.deleted || 0) + ' record(s).');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };

    restoreToken();
    syncBulkDeleteState();
  </script>
</body>
</html>`
