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
    header, main { max-width: 1120px; margin: 0 auto; padding: 16px; }
    .panel { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 16px; margin-bottom: 16px; }
    .grid { display: grid; gap: 16px; grid-template-columns: 280px 1fr; }
    .stack { display: grid; gap: 12px; }
    label { display: grid; gap: 6px; font-size: 14px; }
    input, select, textarea, button { font: inherit; padding: 10px 12px; border-radius: 8px; border: 1px solid #d1d5db; }
    button { cursor: pointer; background: #111827; color: #fff; }
    button.secondary { background: #fff; color: #111827; }
    ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 8px; }
    li button { width: 100%; text-align: left; }
    table { width: 100%; border-collapse: collapse; }
    th, td { border-bottom: 1px solid #e5e7eb; padding: 8px; text-align: left; font-size: 14px; }
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
        <section class="panel">
          <h3>Create record</h3>
          <form id="createForm" class="stack"></form>
        </section>
        <section class="panel">
          <div style="display:flex;justify-content:space-between;gap:12px;align-items:center;">
            <h3 style="margin:0;">List records</h3>
            <button id="reloadList" class="secondary" type="button">Reload list</button>
          </div>
          <div id="list"></div>
        </section>
      </section>
    </section>
  </main>
  <script>
    const apiBase = '/api/v1/admin';
    const state = { current: null, meta: null, resources: [] };

    const els = {
      token: document.getElementById('token'),
      status: document.getElementById('status'),
      resources: document.getElementById('resources'),
      resourceTitle: document.getElementById('resourceTitle'),
      resourcePath: document.getElementById('resourcePath'),
      actions: document.getElementById('actions'),
      createForm: document.getElementById('createForm'),
      list: document.getElementById('list'),
      loadResources: document.getElementById('loadResources'),
      reloadList: document.getElementById('reloadList')
    };

    function setStatus(value) {
      els.status.textContent = value;
    }

    async function request(path, options = {}) {
      const headers = new Headers(options.headers || {});
      headers.set('Content-Type', 'application/json');
      const token = els.token.value.trim();
      if (token) headers.set('Authorization', 'Bearer ' + token);
      const response = await fetch(path, { ...options, headers });
      const text = await response.text();
      let data = null;
      try { data = text ? JSON.parse(text) : null; } catch (_) { data = text; }
      if (!response.ok) {
        throw new Error(typeof data === 'string' ? data : JSON.stringify(data, null, 2));
      }
      return data;
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

    async function selectResource(resource) {
      state.current = resource;
      try {
        state.meta = await request(apiBase + '/resources' + resource.path + '/meta');
        els.resourceTitle.textContent = state.meta.label;
        els.resourcePath.textContent = apiBase + '/resources' + resource.path;
        els.actions.textContent = 'Actions: ' + (state.meta.actions || []).join(', ');
        await Promise.all([renderCreateForm(), renderList()]);
        setStatus('Loaded resource ' + resource.name + '.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    }

    function fieldMeta(name) {
      return (state.meta.fields || []).find((field) => field.name === name);
    }

    async function renderCreateForm() {
      els.createForm.innerHTML = '';
      if (!state.meta || !(state.meta.create_fields || []).length) {
        els.createForm.innerHTML = '<p class="muted">Create is not available for this resource.</p>';
        return;
      }
      for (const name of state.meta.create_fields) {
        const field = fieldMeta(name);
        if (!field) continue;
        const wrapper = document.createElement('label');
        wrapper.textContent = field.label;
        const input = await buildInput(field);
        input.name = field.name;
        wrapper.appendChild(input);
        els.createForm.appendChild(wrapper);
      }
      const submit = document.createElement('button');
      submit.type = 'submit';
      submit.textContent = 'Create';
      els.createForm.appendChild(submit);
    }

    async function buildInput(field) {
      if (field.relation) {
        const select = document.createElement('select');
        const options = await request(apiBase + '/resources' + state.current.path + '/fields/' + field.name + '/options');
        (options.items || []).forEach((item) => {
          const option = document.createElement('option');
          option.value = String(item.value);
          option.textContent = item.label;
          select.appendChild(option);
        });
        return select;
      }
      if (field.component === 'checkbox') {
        const input = document.createElement('input');
        input.type = 'checkbox';
        input.value = 'true';
        return input;
      }
      if (field.component === 'array') {
        const input = document.createElement('textarea');
        input.placeholder = 'JSON array';
        return input;
      }
      const input = document.createElement(field.component === 'textarea' ? 'textarea' : 'input');
      if (input.tagName === 'INPUT') {
        input.type = ({ email: 'email', password: 'password', number: 'number', datetime: 'datetime-local' }[field.component]) || 'text';
      }
      return input;
    }

    function formPayload() {
      const payload = {};
      new FormData(els.createForm).forEach((value, key) => {
        const field = fieldMeta(key);
        if (!field) return;
        if (field.component === 'checkbox') {
          payload[key] = true;
          return;
        }
        if (field.component === 'number') {
          payload[key] = value === '' ? null : Number(value);
          return;
        }
        if (field.component === 'array') {
          payload[key] = value ? JSON.parse(value) : [];
          return;
        }
        payload[key] = value;
      });
      els.createForm.querySelectorAll('input[type=checkbox]').forEach((checkbox) => {
        payload[checkbox.name] = checkbox.checked;
      });
      return payload;
    }

    async function renderList() {
      if (!state.current) return;
      const data = await request(apiBase + '/resources' + state.current.path);
      const fields = state.meta.list_fields || [];
      const rows = data.items || [];
      if (!fields.length) {
        els.list.innerHTML = '<p class="muted">No list fields available.</p>';
        return;
      }
      const table = document.createElement('table');
      const thead = document.createElement('thead');
      thead.innerHTML = '<tr>' + fields.map((field) => '<th>' + field + '</th>').join('') + '</tr>';
      table.appendChild(thead);
      const tbody = document.createElement('tbody');
      rows.forEach((row) => {
        const tr = document.createElement('tr');
        tr.innerHTML = fields.map((field) => '<td>' + String(row[field] ?? '') + '</td>').join('');
        tbody.appendChild(tr);
      });
      table.appendChild(tbody);
      els.list.innerHTML = '';
      els.list.appendChild(table);
    }

    els.loadResources.onclick = loadResources;
    els.reloadList.onclick = () => state.current && renderList().catch((error) => setStatus(String(error.message || error)));
    els.createForm.onsubmit = async (event) => {
      event.preventDefault();
      if (!state.current) return;
      try {
        await request(apiBase + '/resources' + state.current.path, {
          method: 'POST',
          body: JSON.stringify(formPayload())
        });
        await renderList();
        els.createForm.reset();
        setStatus('Created a new ' + state.current.name + ' record.');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    };
  </script>
</body>
</html>`
