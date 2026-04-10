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
    label { display: grid; gap: 6px; font-size: 14px; }
    input, select, textarea, button { font: inherit; padding: 10px 12px; border-radius: 8px; border: 1px solid #d1d5db; }
    textarea { min-height: 96px; }
    button { cursor: pointer; background: #111827; color: #fff; }
    button.secondary { background: #fff; color: #111827; }
    button.danger { background: #b91c1c; }
    ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
    li button { width: 100%; text-align: left; }
    table { width: 100%; border-collapse: collapse; }
    th, td { border-bottom: 1px solid #e5e7eb; padding: 8px; text-align: left; font-size: 14px; vertical-align: top; }
    pre { margin: 0; white-space: pre-wrap; word-break: break-word; background: #111827; color: #f9fafb; padding: 12px; border-radius: 8px; }
    .muted { color: #6b7280; font-size: 14px; }
    .toolbar { display:flex; gap:12px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    .row-actions { display:flex; gap:8px; }
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
        <input id="token" placeholder="Paste a Bearer token from /api/v1/auth/login">
      </label>
      <div>
        <button id="loadResources">Load resources</button>
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
              <button id="reloadList" class="secondary" type="button">Reload list</button>
            </div>
          </div>
          <div id="list"></div>
        </section>
      </section>
    </section>
  </main>
  <script>
    const apiBase = '/api/v1/admin';
    const state = { current: null, meta: null, resources: [], records: [], selected: null };

    const els = {
      token: document.getElementById('token'),
      status: document.getElementById('status'),
      resources: document.getElementById('resources'),
      resourceTitle: document.getElementById('resourceTitle'),
      resourcePath: document.getElementById('resourcePath'),
      actions: document.getElementById('actions'),
      createForm: document.getElementById('createForm'),
      updateForm: document.getElementById('updateForm'),
      list: document.getElementById('list'),
      detail: document.getElementById('detail'),
      selectionHint: document.getElementById('selectionHint'),
      loadResources: document.getElementById('loadResources'),
      reloadList: document.getElementById('reloadList'),
      deleteRecord: document.getElementById('deleteRecord'),
      search: document.getElementById('search')
    };

    function setStatus(value) {
      els.status.textContent = value;
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
      try {
        state.meta = await request(currentBasePath() + '/meta');
        els.resourceTitle.textContent = state.meta.label;
        els.resourcePath.textContent = currentBasePath();
        els.actions.textContent = 'Actions: ' + (state.meta.actions || []).join(', ');
        els.search.value = '';
        await Promise.all([renderCreateForm(), renderUpdateForm(), renderList()]);
        renderSelectedRecord();
        setStatus('Loaded resource ' + resource.name + '.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    }

    async function loadRelationOptions(field) {
      const options = await request(currentBasePath() + '/fields/' + field.name + '/options');
      return options.items || [];
    }

    async function buildInput(field, value) {
      if (field.relation) {
        const select = document.createElement('select');
        select.dataset.component = 'select';
        const options = await loadRelationOptions(field);
        options.forEach((item) => {
          const option = document.createElement('option');
          option.value = String(item.value);
          option.textContent = item.label;
          if (String(value ?? '') === String(item.value)) {
            option.selected = true;
          }
          select.appendChild(option);
        });
        return select;
      }
      if (field.component === 'checkbox') {
        const input = document.createElement('input');
        input.type = 'checkbox';
        input.checked = Boolean(value);
        return input;
      }
      if (field.component === 'array' || field.component === 'text') {
        const input = document.createElement(field.component === 'array' ? 'textarea' : 'input');
        if (field.component === 'array') {
          input.value = Array.isArray(value) ? JSON.stringify(value, null, 2) : (value ? JSON.stringify(value, null, 2) : '');
        } else {
          input.value = value == null ? '' : String(value);
        }
        return input;
      }
      const input = document.createElement('input');
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
        const input = await buildInput(field, values[name]);
        input.name = field.name;
        wrapper.appendChild(input);
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

    function formPayload(form) {
      const payload = {};
      const data = new FormData(form);
      for (const [key, value] of data.entries()) {
        const field = fieldMeta(key);
        if (!field) continue;
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
      form.querySelectorAll('input[type=checkbox]').forEach((checkbox) => {
        payload[checkbox.name] = checkbox.checked;
      });
      return payload;
    }

    async function renderList() {
      if (!state.current) return;
      const query = els.search.value.trim() ? ('?search=' + encodeURIComponent(els.search.value.trim())) : '';
      const data = await request(currentBasePath() + query);
      const fields = state.meta?.list_fields || [];
      const rows = data.items || [];
      state.records = rows;
      if (!fields.length) {
        els.list.innerHTML = '<p class="muted">No list fields available.</p>';
        return;
      }
      const table = document.createElement('table');
      const thead = document.createElement('thead');
      thead.innerHTML = '<tr><th>Actions</th>' + fields.map((field) => '<th>' + field + '</th>').join('') + '</tr>';
      table.appendChild(thead);
      const tbody = document.createElement('tbody');
      rows.forEach((row) => {
        const tr = document.createElement('tr');
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

    els.loadResources.onclick = loadResources;
    els.reloadList.onclick = () => state.current && renderList().then(() => setStatus('Reloaded list.')).catch((error) => setStatus(String(error.message || error)));
    els.search.onkeydown = (event) => {
      if (event.key === 'Enter') {
        event.preventDefault();
        els.reloadList.click();
      }
    };
    els.createForm.onsubmit = async (event) => {
      event.preventDefault();
      if (!state.current) return;
      try {
        await request(currentBasePath(), {
          method: 'POST',
          body: JSON.stringify(formPayload(els.createForm))
        });
        await renderList();
        els.createForm.reset();
        setStatus('Created a new ' + state.current.name + ' record.');
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
        renderSelectedRecord();
        await renderUpdateForm();
        await renderList();
        setStatus('Deleted record #' + id + '.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };
  </script>
</body>
</html>`
