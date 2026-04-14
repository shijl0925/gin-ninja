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
    :root {
      color-scheme: light;
      --admin-body-bg: #f4f6f9;
      --admin-surface: #ffffff;
      --admin-sidebar: #1f2d3d;
      --admin-sidebar-alt: #243447;
      --admin-sidebar-text: #c2c7d0;
      --admin-sidebar-active: #007bff;
      --admin-topbar: #ffffff;
      --admin-border: #dee2e6;
      --admin-text: #212529;
      --admin-muted: #6c757d;
      --admin-primary: #007bff;
      --admin-primary-dark: #0056b3;
      --admin-success: #00a65a;
      --admin-danger: #dd4b39;
      --admin-warning: #f39c12;
      --admin-shadow: 0 1px 3px rgba(0, 0, 0, 0.14), 0 1px 2px rgba(0, 0, 0, 0.2);
      --admin-radius: 0.5rem;
      --admin-topbar-min-height: 64px;
      --admin-topbar-height: calc(var(--admin-topbar-min-height) + 24px);
      --admin-content-gap: 18px;
      --admin-sidebar-width: 280px;
    }
    [data-theme="dark"] {
      color-scheme: dark;
      --admin-body-bg: #0f1117;
      --admin-surface: #1a1d27;
      --admin-sidebar: #111827;
      --admin-sidebar-alt: #1c2535;
      --admin-sidebar-text: #9ca3af;
      --admin-sidebar-active: #007bff;
      --admin-topbar: #1a1d27;
      --admin-border: #2d3242;
      --admin-text: #e2e8f0;
      --admin-muted: #9ca3af;
      --admin-primary: #66b0ff;
      --admin-primary-dark: #007bff;
      --admin-success: #22c55e;
      --admin-danger: #ef4444;
      --admin-warning: #f59e0b;
      --admin-shadow: 0 1px 4px rgba(0, 0, 0, 0.5), 0 1px 3px rgba(0, 0, 0, 0.4);
    }
    [data-theme="dark"] body { background: var(--admin-body-bg); color: var(--admin-text); }
    [data-theme="dark"] .topbar,
    [data-theme="dark"] .sidebar-shell,
    [data-theme="dark"] .panel,
    [data-theme="dark"] .modal-box { background: var(--admin-surface); border-color: var(--admin-border); }
    [data-theme="dark"] .topbar { background: var(--admin-topbar); border-bottom-color: var(--admin-border); }
    [data-theme="dark"] .topbar-user-menu,
    [data-theme="dark"] .topbar-search-expand input,
    [data-theme="dark"] .action-menu-list { background: var(--admin-surface); border-color: var(--admin-border); color: var(--admin-text); }
    [data-theme="dark"] .topbar-link,
    [data-theme="dark"] .topbar-user-btn,
    [data-theme="dark"] .topbar-user-menu-item { color: var(--admin-text); }
    [data-theme="dark"] .topbar-link:hover,
    [data-theme="dark"] .topbar-user-menu-item:hover { background: var(--admin-border); }
    [data-theme="dark"] .topbar-toggle { color: var(--admin-muted); }
    [data-theme="dark"] .topbar-toggle:hover,
    [data-theme="dark"] .topbar-action:hover { background: var(--admin-border); color: var(--admin-text); }
    [data-theme="dark"] .topbar-user-btn:hover { background: var(--admin-border); color: var(--admin-text); }
    [data-theme="dark"] .topbar-user-avatar { background: #2d3242; color: var(--admin-text); }
    [data-theme="dark"] input,
    [data-theme="dark"] select,
    [data-theme="dark"] textarea { background: #22253a; color: var(--admin-text); border-color: var(--admin-border); }
    [data-theme="dark"] input::placeholder,
    [data-theme="dark"] textarea::placeholder { color: var(--admin-muted); }
    [data-theme="dark"] table { color: var(--admin-text); }
    [data-theme="dark"] thead tr { background: #22253a; }
    [data-theme="dark"] th { background: #22253a; color: var(--admin-muted); }
    [data-theme="dark"] th.sortable-th:hover { background: #2d3150; color: #fff; }
    [data-theme="dark"] th.sortable-th.sort-asc,
    [data-theme="dark"] th.sortable-th.sort-desc { background: #1e2d4a; color: #93b4ff; }
    [data-theme="dark"] tbody tr:hover { background: #22253a; }
    [data-theme="dark"] tbody tr.row-selected { background: #1a2e3d; }
    [data-theme="dark"] .dashboard-tile { background: var(--admin-surface); border-color: var(--admin-border); color: var(--admin-text); }
    [data-theme="dark"] .dashboard-tile:hover { border-top-color: var(--dashboard-tile-accent); }
    [data-theme="dark"] .toast { background: var(--admin-surface); border-color: var(--admin-border); color: var(--admin-text); }
    [data-theme="dark"] .login-shell,
    [data-theme="dark"] .login-card,
    [data-theme="dark"] .session-panel { background: var(--admin-surface); border-color: var(--admin-border); color: var(--admin-text); }
    [data-theme="dark"] .nav-link { color: var(--admin-sidebar-text); }
    [data-theme="dark"] .nav-link:hover,
    [data-theme="dark"] .nav-link.active { background: var(--admin-sidebar-alt); color: #fff; }
    [data-theme="dark"] hr,
    [data-theme="dark"] .action-menu-divider { border-color: var(--admin-border); }
    [data-theme="dark"] .muted { color: var(--admin-muted); }
    [data-theme="dark"] th, [data-theme="dark"] td { border-bottom-color: var(--admin-border); }
    [data-theme="dark"] .table-shell { background: var(--admin-surface); border-color: var(--admin-border); box-shadow: none; }
    [data-theme="dark"] .empty-state { background: var(--admin-surface); border-color: var(--admin-border); }
    [data-theme="dark"] .detail-card { background: var(--admin-surface); border-color: var(--admin-border); box-shadow: none; }
    [data-theme="dark"] .detail-row { border-bottom-color: var(--admin-border); }
    [data-theme="dark"] .bulk-edit-field { background: var(--admin-surface); border-color: var(--admin-border); }
    [data-theme="dark"] .relation-preview li { background: var(--admin-surface); border-color: var(--admin-border); color: var(--admin-text); }
    [data-theme="dark"] .relation-preview mark { background: #4a4200; }
    [data-theme="dark"] .inline-field, [data-theme="dark"] .form-field { color: var(--admin-text); }
    [data-theme="dark"] label { color: var(--admin-text); }
    [data-theme="dark"] .modal-dialog { background: var(--admin-surface); border-color: var(--admin-border); }
    [data-theme="dark"] .action-menu-trigger,
    [data-theme="dark"] .action-btn-view,
    [data-theme="dark"] button.secondary { background: var(--admin-surface); color: var(--admin-text); border-color: var(--admin-border); }
    [data-theme="dark"] .action-menu-trigger:hover,
    [data-theme="dark"] .action-btn-view:hover,
    [data-theme="dark"] button.secondary:hover { background: #2a2d42; border-color: #4e5275; }
    [data-theme="dark"] .action-menu-item:hover { background: #2a2d42; }
    [data-theme="dark"] .action-menu-item.danger:hover { background: #2d0f0f; }
    [data-theme="dark"] .badge { background: #1a2a3e; color: #93b4ff; }
    [data-theme="dark"] .eyebrow.subtle { background: #22253a; color: var(--admin-muted); }
    [data-theme="dark"] .login-credentials { background: #22253a; border-color: var(--admin-border); }
    [data-theme="dark"] .login-credentials code { background: #1a1d2e; border-color: var(--admin-border); color: var(--admin-text); }
    [data-theme="dark"] body.standalone-admin-page .topbar,
    [data-theme="dark"] body.legacy-prototype-page .topbar { background: var(--admin-topbar); }
    [data-theme="dark"] button:hover:not(:disabled) { filter: brightness(1.15); }
    [data-theme="dark"] .search-result-item mark { background: #4a4200; }
    [hidden] { display:none !important; }
    * { box-sizing: border-box; }
    body {
      font-family: Inter, system-ui, "Segoe UI", sans-serif;
      margin: 0;
      min-height: 100vh;
      background: var(--admin-body-bg);
      color: var(--admin-text);
    }
    a { color: inherit; }
    .topbar {
      position: sticky;
      top: 0;
      z-index: 30;
      display:flex;
      align-items:center;
      justify-content:flex-start;
      gap:16px;
      min-height:var(--admin-topbar-min-height);
      padding:0 16px;
      background: var(--admin-topbar);
      border-bottom:1px solid var(--admin-border);
      box-shadow:0 1px 3px rgba(0,0,0,.1);
    }
    .topbar-left, .topbar-brand, .topbar-copy, .topbar-meta, .sidebar-heading { display:grid; gap:6px; }
    .topbar-left {
      display:flex;
      align-items:center;
      gap:4px;
      min-width:0;
      flex:0 1 auto;
    }
    .topbar-brand {
      grid-template-columns:auto 1fr;
      align-items:center;
      gap:14px;
      min-width:0;
    }
    .topbar .topbar-toggle {
      display:inline-flex;
      align-items:center;
      justify-content:center;
      width:42px;
      flex:0 0 42px;
      min-width:42px;
      min-height:42px;
      padding:0;
      border:none;
      background:transparent;
      color:#6c757d;
      box-shadow:none;
      font-size:20px;
      line-height:1;
      border-radius:0.35rem;
    }
    .topbar-nav {
      display:flex;
      align-items:stretch;
      gap:0;
      min-width:0;
      margin:0;
      flex-direction:row;
    }
    .topbar-link {
      display:inline-flex;
      align-items:center;
      gap:8px;
      min-height:var(--admin-topbar-min-height);
      padding:0 14px;
      border-radius:0;
      color:#495057;
      text-decoration:none;
      font-size:15px;
      font-weight:400;
    }
    .topbar-link:hover { background:#f4f6f9; color:#212529; }
    .brand-mark {
      width:38px;
      height:38px;
      border-radius:999px;
      display:grid;
      place-items:center;
      background:#f4f6f9;
      color:#495057;
      border:1px solid rgba(0,0,0,0.12);
      font-weight:800;
      letter-spacing:0.08em;
      box-shadow: 0 2px 6px rgba(0,0,0,0.14);
    }
    .topbar-copy h1, .sidebar-heading h2, .section-title { margin:0; }
    .topbar-copy p, .sidebar-heading p, .section-copy, .login-marketing p, .login-lead p { margin:0; }
    .topbar-meta {
      display:flex;
      align-items:center;
      justify-content:flex-end;
      gap:14px;
      min-width:0;
      margin-left:auto;
      flex:0 0 auto;
    }
    .topbar-actions {
      display:flex;
      align-items:center;
      gap:0;
      border-left:1px solid var(--admin-border);
    }
    .topbar-action {
      position:relative;
      display:inline-flex;
      align-items:center;
      justify-content:center;
      width:48px;
      min-height:var(--admin-topbar-min-height);
      padding:0;
      border:none;
      border-left:1px solid var(--admin-border);
      border-radius:0;
      background:transparent;
      color:#6c757d;
      box-shadow:none;
      font-size:18px;
      line-height:1;
    }
    .topbar-action:hover { background:#f4f6f9; color:#212529; }
    .topbar-user-dropdown { position:relative; display:inline-flex; align-items:center; }
    .topbar-user-btn {
      display:inline-flex;
      align-items:center;
      gap:8px;
      width:auto;
      min-height:var(--admin-topbar-min-height);
      padding:0 14px 0 12px;
      background:transparent;
      border:none;
      border-left:1px solid var(--admin-border);
      box-shadow:none;
      color:#495057;
      font-size:14px;
      font-weight:500;
      cursor:pointer;
      white-space:nowrap;
      border-radius:0;
    }
    .topbar-user-btn:hover { background:#f4f6f9; color:#212529; }
    .topbar-user-avatar {
      display:inline-grid;
      place-items:center;
      width:32px;
      height:32px;
      border-radius:999px;
      background:#d2d6de;
      color:#495057;
      font-size:13px;
      font-weight:700;
      flex-shrink:0;
    }
    .topbar-user-name { max-width:120px; overflow:hidden; text-overflow:ellipsis; }
    .topbar-caret { flex-shrink:0; color:#6c757d; }
    .topbar-user-menu {
      position:absolute;
      top:calc(100% + 4px);
      right:0;
      min-width:160px;
      background:#fff;
      border:1px solid var(--admin-border);
      border-radius:0.35rem;
      box-shadow:0 4px 16px rgba(0,0,0,0.12);
      list-style:none;
      margin:0;
      padding:4px 0;
      z-index:200;
    }
    .topbar-user-menu-item {
      display:block;
      width:100%;
      text-align:left;
      padding:9px 16px;
      background:none;
      border:none;
      box-shadow:none;
      color:var(--admin-text);
      font-size:14px;
      cursor:pointer;
      border-radius:0;
    }
    .topbar-user-menu-item:hover { background:#f4f6f9; }
    .topbar-search-wrap { position:relative; display:inline-flex; align-items:center; }
    .topbar-search-expand {
      display:none;
      position:absolute;
      top:calc(100% + 4px);
      right:0;
      width:320px;
      z-index:200;
    }
    .topbar-search-expand.open { display:block; }
    .topbar-search-expand input {
      width:100%;
      border-radius:0.35rem 0.35rem 0 0;
      border:1px solid var(--admin-border);
      padding:8px 14px;
      font-size:14px;
      box-shadow:0 2px 8px rgba(0,0,0,0.10);
    }
    .topbar-search-results {
      display:none;
      background:var(--admin-surface);
      border:1px solid var(--admin-border);
      border-top:none;
      border-radius:0 0 0.35rem 0.35rem;
      box-shadow:0 4px 12px rgba(0,0,0,0.12);
      max-height:320px;
      overflow-y:auto;
    }
    .topbar-search-results.has-results { display:block; }
    .search-results-group-label {
      padding:6px 12px 3px;
      font-size:11px;
      font-weight:700;
      letter-spacing:0.05em;
      text-transform:uppercase;
      color:var(--admin-muted);
      background:var(--admin-body-bg);
      border-top:1px solid var(--admin-border);
    }
    .search-results-group-label:first-child { border-top:none; }
    .search-result-item {
      display:flex;
      align-items:center;
      gap:8px;
      width:100%;
      padding:7px 12px;
      border:none;
      background:none;
      cursor:pointer;
      text-align:left;
      font-size:13px;
      color:var(--admin-text);
      transition:background 0.1s;
    }
    .search-result-item:hover,
    .search-result-item:focus { background:var(--admin-body-bg); outline:none; }
    .search-result-item mark { background:#fff9c4; color:inherit; border-radius:2px; padding:0 1px; }
    .search-result-summary { flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
    .search-result-id { font-size:11px; color:var(--admin-muted); flex-shrink:0; }
    .search-results-empty {
      padding:14px 12px;
      font-size:13px;
      color:var(--admin-muted);
      text-align:center;
    }
    .toast-container {
      position:fixed;
      top:calc(var(--admin-topbar-min-height) + 12px);
      right:18px;
      z-index:500;
      display:grid;
      gap:8px;
      pointer-events:none;
    }
    .toast {
      display:flex;
      align-items:flex-start;
      gap:12px;
      min-width:280px;
      max-width:420px;
      padding:12px 14px;
      background:#fff;
      border:1px solid var(--admin-border);
      border-left:4px solid var(--admin-border);
      border-radius:0.45rem;
      box-shadow:0 4px 18px rgba(0,0,0,.13);
      font-size:14px;
      pointer-events:all;
      animation:toast-in 200ms ease;
    }
    .toast[data-tone="success"] { border-left-color:var(--admin-success); }
    .toast[data-tone="danger"] { border-left-color:var(--admin-danger); }
    .toast[data-tone="info"] { border-left-color:var(--admin-primary); }
    .toast-message { flex:1 1 auto; min-width:0; line-height:1.45; }
    .toast-close {
      flex-shrink:0;
      background:none;
      border:none;
      box-shadow:none;
      cursor:pointer;
      color:#6c757d;
      padding:0;
      font-size:18px;
      line-height:1;
      width:20px;
      height:20px;
      display:grid;
      place-items:center;
    }
    .toast-close:hover { color:#212529; }
    @keyframes toast-in {
      from { opacity:0; transform:translateY(-8px); }
      to   { opacity:1; transform:translateY(0); }
    }
    .app-main {
      display:grid;
      gap:var(--admin-content-gap);
      padding:18px 24px 24px;
      align-items:start;
    }
    .panel {
      min-width:0;
      background:var(--admin-surface);
      border:1px solid var(--admin-border);
      border-top:3px solid #d2d6de;
      border-radius: var(--admin-radius);
      padding:18px;
      box-shadow: var(--admin-shadow);
    }
    .stack { display:grid; gap:16px; }
    .toolbar { display:flex; gap:12px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    .toolbar > *, .row-actions > *, .workspace-meta > *, .table-toolbar > *, .pagination-bar > * { min-width:0; }
    .row-actions { display:flex; gap:10px; align-items:center; flex-wrap:wrap; }
    .eyebrow {
      display:inline-flex;
      width:max-content;
      align-items:center;
      gap:6px;
      border-radius:999px;
      padding:6px 10px;
      background:rgba(0, 123, 255, 0.12);
      color:var(--admin-primary-dark);
      font-size:11px;
      font-weight:700;
      letter-spacing:0.08em;
      text-transform:uppercase;
    }
    .eyebrow.subtle { background:#e9ecef; color:#495057; }
    .badge {
      display:inline-flex;
      align-items:center;
      gap:6px;
      font-size:12px;
      font-weight:700;
      background:#eaf3f8;
      color:var(--admin-primary-dark);
      border-radius:999px;
      padding:6px 11px;
    }
    .visually-hidden { position:absolute !important; width:1px; height:1px; padding:0; margin:-1px; overflow:hidden; clip:rect(0, 0, 0, 0); white-space:nowrap; border:0; }
    .login-shell { display:grid; gap:20px; }
    .session-panel { position:relative; overflow:hidden; }
    .login-marketing, .login-lead, .login-credentials { display:none; }
    .login-marketing {
      align-content:start;
      background:linear-gradient(160deg, var(--admin-sidebar) 0%, var(--admin-sidebar-alt) 100%);
      color:#f8fafc;
      border:1px solid rgba(255,255,255,0.06);
      border-top:3px solid var(--admin-primary);
    }
    .login-marketing h2, .login-lead h2 { margin:0; line-height:1.15; }
    .login-feature-list { display:grid; gap:12px; }
    .login-feature {
      display:grid;
      gap:4px;
      padding:16px 18px;
      background:rgba(255,255,255,0.06);
      border:1px solid rgba(255,255,255,0.08);
      border-radius:14px;
    }
    .login-feature strong { font-size:15px; }
    .login-credentials {
      gap:8px;
      padding:12px 14px;
      border-radius:14px;
      background:#f8fbfd;
      border:1px solid #d8e5ec;
    }
    .login-credentials code {
      font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
      font-size:13px;
      background:#fff;
      border:1px solid #d8e5ec;
      border-radius:8px;
      padding:2px 6px;
    }
    .admin-shell {
      display:grid;
      gap:20px;
      grid-template-columns:var(--admin-sidebar-width) minmax(0, 1fr);
      align-items:start;
    }
    .sidebar-shell {
      position:sticky;
      top:calc(var(--admin-topbar-height) + var(--admin-content-gap));
      display:flex;
      flex-direction:column;
      min-height:calc(100vh - var(--admin-topbar-height) - (var(--admin-content-gap) * 2));
      background:#343a40;
      color:var(--admin-sidebar-text);
      border:none;
      border-radius:0.5rem;
      box-shadow:0 14px 30px rgba(17, 24, 39, 0.16);
      padding:0;
      overflow:hidden;
    }
    .sidebar-brand {
      display:grid;
      grid-template-columns:auto 1fr;
      align-items:center;
      gap:14px;
      padding:20px 18px;
      background:transparent;
      border-bottom:1px solid rgba(255,255,255,.1);
    }
    .sidebar-brand-mark {
      width:48px;
      height:48px;
      display:grid;
      place-items:center;
      border-radius:50%;
      background:#f4f6f9;
      color:#343a40;
      font-size:23px;
      font-weight:800;
      box-shadow:0 4px 14px rgba(0,0,0,0.28);
      flex-shrink:0;
    }
    .sidebar-brand-copy { display:grid; gap:2px; min-width:0; }
    .sidebar-brand-copy strong {
      color:#f8f9fa;
      font-size:1.05rem;
      font-weight:300;
      letter-spacing:0.01em;
      white-space:nowrap;
    }
    .sidebar-brand-copy span {
      color:rgba(255,255,255,.5);
      font-size:12px;
      letter-spacing:0.02em;
    }
    .sidebar-user-panel {
      display:grid;
      grid-template-columns:auto 1fr;
      align-items:center;
      gap:14px;
      padding:20px 18px;
      margin:0;
      border-bottom:1px solid rgba(255,255,255,.1);
    }
    .sidebar-user-avatar {
      width:46px;
      height:46px;
      display:grid;
      place-items:center;
      border-radius:50%;
      background:linear-gradient(135deg, #cfd4da 0%, #f8f9fa 100%);
      color:#495057;
      font-size:14px;
      font-weight:700;
      box-shadow:0 4px 10px rgba(0,0,0,0.24);
      flex-shrink:0;
    }
    .sidebar-user-copy { display:grid; gap:2px; min-width:0; }
    .sidebar-user-copy strong {
      color:#f8f9fa;
      font-size:16px;
      font-weight:400;
      white-space:nowrap;
      overflow:hidden;
      text-overflow:ellipsis;
    }
    .sidebar-user-status {
      display:flex;
      align-items:center;
      gap:5px;
      font-size:11px;
      color:#adb5bd;
    }
    .sidebar-user-online {
      display:inline-block;
      width:8px;
      height:8px;
      border-radius:50%;
      background:var(--admin-success);
      flex-shrink:0;
    }
    .sidebar-search {
      position:relative;
      padding:18px;
      border-bottom:1px solid rgba(255,255,255,0.08);
    }
    .sidebar-search input {
      width:100%;
      height:44px;
      padding-right:44px;
      border-color:#56606a;
      background:#495057;
      color:#f8f9fa;
    }
    .sidebar-search input::placeholder { color:#adb5bd; }
    .sidebar-search button {
      position:absolute;
      top:50%;
      right:24px;
      transform:translateY(-50%);
      width:36px;
      height:36px;
      padding:0;
      background:#495057;
      border:1px solid #56606a;
      color:#f8f9fa;
      box-shadow:none;
    }
    .sidebar-search-empty {
      padding:10px 14px;
      color:#adb5bd;
      font-size:12px;
      line-height:1.4;
    }
    .resource-strip-header, .resource-strip-copy { display:grid; gap:6px; }
    .resource-strip-header { display:none; }
    .resource-strip-copy strong, .resource-strip-copy p { margin:0; }
    .sidebar-heading h2 { color:#fff; font-size:1.15rem; }
    .sidebar-heading p, .sidebar-heading .eyebrow { color:var(--admin-sidebar-text); }
    .sidebar-heading .eyebrow.subtle { background:rgba(255,255,255,0.08); }
    .sidebar-nav-shell {
      display:grid;
      gap:10px;
      flex:1 1 auto;
      align-content:start;
      padding:14px 14px 18px;
      overflow:auto;
    }
    .sidebar-section-label {
      padding:8px 8px 2px;
      font-size:12px;
      line-height:1.2;
      font-weight:700;
      letter-spacing:0.08em;
      text-transform:uppercase;
      color:rgba(255,255,255,.58);
    }
    .sidebar-treeview { display:grid; gap:6px; }
    .sidebar-treeview-toggle {
      display:flex;
      align-items:center;
      justify-content:flex-start;
      gap:12px;
      width:100%;
      padding:11px 14px;
      background:transparent;
      border:none;
      border-radius:0.35rem;
      box-shadow:none;
      color:#c2c7d0;
      font-size:15px;
      font-weight:400;
      line-height:1.3;
      transition:background 120ms ease, color 120ms ease;
    }
    .sidebar-treeview-toggle-copy {
      display:flex;
      align-items:center;
      gap:12px;
      min-width:0;
      flex:1 1 auto;
    }
    .sidebar-treeview-toggle-icon,
    .sidebar-treeview-caret {
      flex-shrink:0;
      width:16px;
      height:16px;
      display:grid;
      place-items:center;
      color:#6c757d;
      transition:transform 120ms ease, color 120ms ease;
    }
    .sidebar-treeview-toggle-icon svg {
      width:15px;
      height:15px;
      stroke:currentColor;
      stroke-width:1.75;
      fill:none;
      stroke-linecap:round;
      stroke-linejoin:round;
    }
    .sidebar-treeview-caret svg {
      width:13px;
      height:13px;
      stroke:currentColor;
      stroke-width:2;
      fill:none;
      stroke-linecap:round;
      stroke-linejoin:round;
    }
    .sidebar-treeview-toggle-text {
      min-width:0;
      overflow:hidden;
      text-overflow:ellipsis;
      white-space:nowrap;
    }
    .sidebar-treeview-toggle:hover,
    .sidebar-treeview.open .sidebar-treeview-toggle {
      background:rgba(255,255,255,0.08);
      color:#fff;
    }
    .sidebar-treeview-toggle-badge {
      min-width:22px;
      height:22px;
      padding:0 7px;
      border-radius:0.45rem;
      display:grid;
      place-items:center;
      font-size:12px;
      font-weight:700;
      color:#fff;
      background:#17a2b8;
      box-shadow:inset 0 1px 0 rgba(255,255,255,.15);
    }
    .sidebar-treeview-toggle:hover .sidebar-treeview-toggle-icon,
    .sidebar-treeview-toggle:hover .sidebar-treeview-caret,
    .sidebar-treeview.open .sidebar-treeview-toggle-icon,
    .sidebar-treeview.open .sidebar-treeview-caret { color:#c2c7d0; }
    .sidebar-treeview.open .sidebar-treeview-caret {
      transform:rotate(-90deg);
      color:#fff;
    }
    .nav-list { list-style:none; margin:0; padding:0; display:grid; gap:4px; }
    .nav-list li { margin:0; min-width:0; }
    .sidebar-treeview-menu {
      display:grid;
      gap:2px;
      padding:2px 0 0;
      margin:0;
    }
    .sidebar-treeview:not(.open) .sidebar-treeview-menu { display:none; }
    .nav-link {
      position:relative;
      width:100%;
      text-align:left;
      background:transparent;
      color:var(--admin-sidebar-text);
      border:none;
      border-radius:0.5rem;
      padding:12px 14px;
      display:flex;
      align-items:center;
      gap:12px;
      font-size:14px;
      font-weight:400;
      line-height:1.25;
      box-shadow:none;
      transition:background 120ms ease, color 120ms ease;
    }
    .nav-link:hover {
      background:rgba(255,255,255,0.08);
      color:#fff;
    }
    .nav-link.active {
      background:#007bff;
      color:#fff;
      box-shadow:0 10px 18px rgba(0, 123, 255, 0.3);
    }
    .nav-link-icon {
      width:18px;
      height:18px;
      flex-shrink:0;
      display:grid;
      place-items:center;
      transition:transform 120ms ease, color 120ms ease;
    }
    .nav-link-icon svg {
      width:100%;
      height:100%;
      stroke:currentColor;
      stroke-width:1.8;
      fill:none;
      stroke-linecap:round;
      stroke-linejoin:round;
    }
    .nav-link:hover .nav-link-icon { color:#fff; }
    .nav-link.active .nav-link-icon {
      color:#fff;
      transform:none;
    }
    .nav-link-label {
      flex:1 1 auto;
      min-width:0;
      overflow:hidden;
      text-overflow:ellipsis;
      white-space:nowrap;
      font-size:14px;
    }
    .nav-link-label mark {
      background:rgba(255,255,255,0.18);
      color:#fff;
      border-radius:4px;
      padding:0 2px;
    }
    .workspace { min-width:0; }
    .workspace-header {
      display:grid;
      gap:0;
      padding:0;
      border-top-color:var(--admin-primary);
      overflow:hidden;
    }
    .workspace-header-main {
      display:flex;
      gap:18px;
      align-items:flex-start;
      justify-content:space-between;
      flex-wrap:wrap;
      padding:18px 20px 14px;
      background:linear-gradient(180deg, rgba(248,249,250,0.95) 0%, rgba(255,255,255,0.98) 100%);
      border-bottom:1px solid rgba(0,0,0,0.06);
    }
    .workspace-header-copy { display:grid; gap:4px; flex:1 1 320px; min-width:0; }
    .workspace-header-copy h2,
    .workspace-header-copy p { margin:0; }
    .workspace-header-copy h2 { font-size:clamp(1.35rem, 1.8vw, 1.65rem); line-height:1.15; }
    .workspace-header-kicker { margin-bottom:2px; }
    .workspace-path {
      display:inline-flex;
      width:max-content;
      max-width:100%;
      align-items:center;
      padding:0;
      font-size:12px;
      line-height:1.35;
      color:var(--admin-muted);
    }
    .content-header-breadcrumb {
      display:flex;
      align-items:center;
      gap:8px;
      flex-wrap:wrap;
      list-style:none;
      margin:0;
      padding:0;
      color:var(--admin-muted);
      font-size:12px;
      text-transform:uppercase;
      letter-spacing:0.06em;
    }
    .content-header-breadcrumb li {
      display:inline-flex;
      align-items:center;
      gap:8px;
    }
    .content-header-breadcrumb li + li::before {
      content:'/';
      color:#adb5bd;
      margin-right:8px;
    }
    .workspace-actions {
      display:flex;
      gap:10px;
      flex-wrap:wrap;
      align-items:center;
      justify-content:space-between;
      padding:14px 20px 18px;
      background:var(--admin-surface);
    }
    .workspace-actions-copy {
      display:grid;
      gap:2px;
      min-width:0;
    }
    .workspace-actions-copy strong {
      font-size:13px;
      color:#495057;
    }
    .workspace-actions-copy span {
      font-size:12px;
      color:var(--admin-muted);
    }
    .content-grid { display:grid; gap:16px; grid-template-columns:minmax(0, 1fr); align-items:start; }
    .section-shell { display:grid; gap:14px; }
    .section-heading { display:grid; gap:6px; }
    .section-card-header,
    .section-card-footer {
      display:flex;
      align-items:flex-start;
      justify-content:space-between;
      gap:14px;
      flex-wrap:wrap;
      padding:16px 18px;
      background:linear-gradient(180deg, rgba(248,249,250,0.95) 0%, rgba(255,255,255,1) 100%);
    }
    .section-card-header {
      border-bottom:1px solid rgba(0,0,0,0.06);
      border-top-left-radius:calc(var(--admin-radius) - 1px);
      border-top-right-radius:calc(var(--admin-radius) - 1px);
    }
    .section-card-body {
      display:grid;
      gap:14px;
      padding:18px;
      background:var(--admin-surface);
    }
    .section-card-footer {
      align-items:center;
      border-top:1px solid rgba(0,0,0,0.06);
      background:#fbfcfd;
    }
    .section-card-tools {
      display:flex;
      align-items:center;
      gap:10px;
      flex-wrap:wrap;
      margin-left:auto;
    }
    .section-card-tools .eyebrow {
      background:rgba(0,123,255,0.08);
    }
    .two-col { display:grid; gap:20px; grid-template-columns:repeat(auto-fit, minmax(240px, 1fr)); }
    .filters { display:grid; gap:12px; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); }
    .inline-field, .form-field { display:grid; gap:8px; font-size:14px; font-weight:600; color:#495057; }
    .field-help, .muted { font-size:13px; color:var(--admin-muted); }
    .relation-control { display:grid; gap:10px; }
    .multi-relation-dropdown { position:relative; }
    .multi-relation-dropdown summary {
      list-style:none;
      cursor:pointer;
      border:1px solid var(--admin-border);
      border-radius:12px;
      background:#fff;
      padding:12px 14px;
      font-size:14px;
      color:var(--admin-text);
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:12px;
    }
    .multi-relation-dropdown summary::-webkit-details-marker { display:none; }
    .multi-relation-dropdown summary::after { content:'▾'; font-size:12px; color:var(--admin-muted); }
    .multi-relation-dropdown[open] summary::after { content:'▴'; }
    .multi-relation-menu {
      position:absolute;
      top:calc(100% + 8px);
      left:0;
      right:0;
      z-index:5;
      display:grid;
      gap:6px;
      max-height:240px;
      overflow:auto;
      border:1px solid var(--admin-border);
      border-radius:12px;
      background:#fff;
      padding:10px;
      box-shadow:0 18px 40px rgba(15, 23, 42, 0.12);
    }
    .multi-relation-option {
      display:flex;
      align-items:center;
      gap:10px;
      padding:8px 10px;
      border-radius:10px;
      cursor:pointer;
      font-size:14px;
      color:var(--admin-text);
    }
    .multi-relation-option:hover { background:#f8fafc; }
    .multi-relation-option input { margin:0; }
    .multi-relation-empty {
      padding:8px 10px;
      border-radius:10px;
      font-size:13px;
      color:var(--admin-muted);
      background:#f8fafc;
    }
    .relation-preview { display:grid; gap:6px; margin:0; padding:0; list-style:none; }
    .relation-preview li { font-size:12px; color:#334155; background:#fff; border:1px solid var(--admin-border); border-radius:8px; padding:8px 10px; }
    .relation-preview mark { background:#fcf8e3; padding:0; }
    .detail-layout { display:grid; gap:16px; grid-template-columns:minmax(0, 1fr); align-items:start; }
    .content-grid > *, .content-grid form, .detail-layout > *, .detail-layout form, .bulk-edit-field { min-width:0; }
    .detail-card { border:1px solid var(--admin-border); border-radius:var(--admin-radius); padding:18px; background:var(--admin-surface); box-shadow: inset 0 1px 0 rgba(255,255,255,0.4); }
    .detail-grid { display:grid; gap:10px; }
    .detail-row { display:grid; grid-template-columns: 160px 1fr; gap:12px; border-bottom:1px solid #edf1f4; padding-bottom:10px; }
    .detail-row:last-child { border-bottom:none; padding-bottom:0; }
    .detail-label { font-size:12px; font-weight:700; color:var(--admin-muted); text-transform:uppercase; letter-spacing:0.06em; }
    .detail-value { font-size:14px; word-break:break-word; color:var(--admin-text); }
    .bulk-edit-fields { display:grid; gap:12px; }
    .bulk-edit-field { border:1px solid var(--admin-border); border-radius:var(--admin-radius); padding:14px; background:#fff; }
    .table-toolbar, .pagination-bar { display:flex; gap:12px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    .table-toolbar .row-actions { flex:1 1 480px; }
    .table-toolbar input, .table-toolbar select { flex:1 1 180px; min-width:0; }
    .pagination-info { font-size:14px; color:var(--admin-muted); }
    .table-shell { overflow:auto; border:1px solid var(--admin-border); border-radius:var(--admin-radius); background:var(--admin-surface); box-shadow: inset 0 1px 0 rgba(255,255,255,0.65); }
    .empty-state { border:1px dashed #c7d0d9; border-radius:var(--admin-radius); padding:28px 20px; background:#fff; color:var(--admin-muted); text-align:center; }
    .workspace-actions button { padding-inline:14px; }
    .modal-overlay { position:fixed; inset:0; background:rgba(17, 24, 39, 0.48); display:grid; place-items:center; padding:24px; z-index:50; }
    .modal-dialog {
      width:min(720px, 100%);
      max-height:min(85vh, 920px);
      overflow:auto;
      border-radius:0.3rem;
      border:1px solid var(--admin-border);
      border-top:3px solid var(--admin-primary);
      background:#fff;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.18);
    }
    .modal-dialog.large { width:min(860px, 100%); }
    .modal-header { display:flex; gap:16px; align-items:flex-start; justify-content:space-between; flex-wrap:wrap; padding:24px 24px 0; }
    .modal-body { padding:0 24px 24px; }
    .modal-close { min-width:44px; min-height:44px; padding:0 14px; }
    body.modal-open { overflow:hidden; }
    label { display:grid; gap:8px; font-size:14px; font-weight:600; color:#495057; }
    input, select, textarea, button {
      font: inherit;
      padding: 10px 12px;
      border-radius: 0.25rem;
      border: 1px solid #ced4da;
      background:#fff;
      color:var(--admin-text);
      transition:border-color 120ms ease, box-shadow 120ms ease, background 120ms ease;
    }
    input:focus, select:focus, textarea:focus {
      outline:none;
      border-color:#80bdff;
      box-shadow:0 0 0 0.2rem rgba(0, 123, 255, 0.2);
    }
    textarea { min-height: 112px; }
    button {
      cursor:pointer;
      background:var(--admin-primary);
      color:#fff;
      border-color:var(--admin-primary-dark);
      font-weight:400;
    }
    button.secondary { background:#fff; color:var(--admin-text); border-color:#ced4da; }
    button.secondary:hover { background:#e9ecef; border-color:#adb5bd; }
    button.danger { background:var(--admin-danger); border-color:#c0392b; }
    button.danger:hover { background:#c0392b; border-color:#a93226; }
    button:hover:not(:disabled) { filter:brightness(0.9); }
    button:disabled, input:disabled, select:disabled, textarea:disabled { opacity:0.6; cursor:not-allowed; }
    table { width:100%; border-collapse:collapse; min-width:720px; }
    th, td { border-bottom:1px solid #dee2e6; padding:0.75rem; text-align:left; font-size:14px; vertical-align:top; }
    th { background:#f8f9fa; color:#495057; font-size:12px; font-weight:700; text-transform:uppercase; letter-spacing:0.06em; border-bottom-width:2px; }
    th.sortable-th { cursor:pointer; user-select:none; white-space:nowrap; }
    th.sortable-th:hover { background:#e9ecef; color:#212529; }
    th.sortable-th.sort-asc, th.sortable-th.sort-desc { color:var(--admin-primary); background:#eef2ff; }
    .sort-icon { display:inline-block; margin-left:4px; font-style:normal; opacity:0.45; font-size:10px; vertical-align:middle; }
    th.sortable-th.sort-asc .sort-icon,
    th.sortable-th.sort-desc .sort-icon { opacity:1; }
    tbody tr:hover { background:rgba(0,0,0,.04); }
    tbody tr.row-selected { background:#eaf3f8; }
    .action-cell { display:flex; gap:6px; align-items:center; white-space:nowrap; }
    .action-menu { position:relative; display:inline-block; }
    .action-menu-trigger,
    .action-btn-view { background:#fff; color:var(--admin-text); border:1px solid #ced4da; padding:6px 10px; font-size:13px; font-weight:600; border-radius:0.25rem; cursor:pointer; line-height:1; }
    .action-menu-trigger:hover { background:#f8f9fa; border-color:#adb5bd; }
    .action-menu-list { display:none; position:absolute; right:0; top:calc(100% + 4px); min-width:130px; background:#fff; border:1px solid var(--admin-border); border-radius:0.25rem; box-shadow:0 8px 24px rgba(15,23,42,0.12); z-index:100; overflow:hidden; }
    .action-menu-list.open { display:block; }
    .action-menu-item { display:block; width:100%; text-align:left; background:none; color:var(--admin-text); border:none; border-radius:0; padding:10px 14px; font-size:14px; font-weight:500; cursor:pointer; transition:background 80ms; }
    .action-menu-item:hover { background:#f1f3f5; }
    .action-menu-item:disabled { opacity:0.45; cursor:not-allowed; }
    .action-menu-divider { border:none; border-top:1px solid var(--admin-border); margin:4px 0; }
    .action-menu-item.danger { background:transparent; color:var(--admin-danger); border-color:transparent; }
    .action-menu-item.danger:hover { background:#fdf1ef; }
    .action-btn-view:hover { background:#f8f9fa; border-color:#adb5bd; }
    pre { margin:0; white-space:pre-wrap; word-break:break-word; background:#1f2d3d; color:#e9ecef; padding:14px; border-radius:0.65rem; }
    @keyframes spin { to { transform:rotate(360deg); } }
    .list-loading {
      display:none;
      align-items:center;
      justify-content:center;
      gap:12px;
      padding:36px 20px;
      color:var(--admin-muted);
      font-size:14px;
    }
    .list-loading.active { display:flex; }
    .list-spinner {
      width:22px;
      height:22px;
      border:3px solid #dee2e6;
      border-top-color:var(--admin-primary);
      border-radius:50%;
      animation:spin 0.65s linear infinite;
      flex-shrink:0;
    }
    .confirm-dialog { width:min(460px, 100%); }
    .confirm-actions { display:flex; gap:10px; justify-content:flex-end; flex-wrap:wrap; }
    .dashboard-tiles {
      display:grid;
      gap:18px;
      grid-template-columns:repeat(auto-fill, minmax(240px, 1fr));
      align-items:stretch;
    }
    .dashboard-tile {
      --dashboard-tile-accent: var(--admin-primary);
      display:grid;
      gap:14px;
      padding:18px 18px 16px;
      border:1px solid var(--admin-border);
      border-top:3px solid var(--dashboard-tile-accent);
      border-radius:calc(var(--admin-radius) + 2px);
      background:#fff;
      box-shadow:var(--admin-shadow);
      cursor:pointer;
      text-align:left;
      color:var(--admin-text);
      transition:transform 140ms ease, border-top-color 140ms ease, box-shadow 140ms ease;
    }
    .dashboard-tile:hover,
    .dashboard-tile:focus-visible {
      transform:translateY(-2px);
      border-top-color:var(--dashboard-tile-accent);
      box-shadow:0 14px 28px rgba(31,45,61,0.14);
    }
    .dashboard-tile:focus-visible {
      outline:2px solid color-mix(in srgb, var(--dashboard-tile-accent) 55%, white);
      outline-offset:2px;
    }
    .dashboard-tile-header,
    .dashboard-tile-footer {
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:12px;
    }
    .dashboard-tile-badge {
      display:inline-flex;
      align-items:center;
      padding:5px 10px;
      border-radius:999px;
      background:color-mix(in srgb, var(--dashboard-tile-accent) 12%, white);
      color:var(--dashboard-tile-accent);
      font-size:11px;
      font-weight:700;
      letter-spacing:0.08em;
      text-transform:uppercase;
    }
    .dashboard-tile-icon-badge {
      width:42px;
      height:42px;
      display:grid;
      place-items:center;
      border-radius:14px;
      background:color-mix(in srgb, var(--dashboard-tile-accent) 14%, white);
      color:var(--dashboard-tile-accent);
      box-shadow:inset 0 1px 0 rgba(255,255,255,0.7);
      flex-shrink:0;
    }
    .dashboard-tile-icon-badge svg {
      width:22px;
      height:22px;
      stroke:currentColor;
      stroke-width:1.8;
      fill:none;
      stroke-linecap:round;
      stroke-linejoin:round;
    }
    .dashboard-tile-main {
      display:grid;
      gap:8px;
      min-height:132px;
      align-content:start;
    }
    .dashboard-tile-count-row {
      display:flex;
      align-items:flex-end;
      gap:8px;
    }
    .dashboard-tile-count {
      font-size:2.4rem;
      font-weight:800;
      color:var(--dashboard-tile-accent);
      line-height:0.95;
      letter-spacing:-0.03em;
    }
    .dashboard-tile-count-label {
      font-size:12px;
      font-weight:700;
      color:var(--admin-muted);
      text-transform:uppercase;
      letter-spacing:0.08em;
      transform:translateY(-2px);
    }
    .dashboard-tile-label { font-size:1.2rem; font-weight:700; line-height:1.15; }
    .dashboard-tile-hint {
      font-size:12px;
      color:var(--admin-muted);
      text-transform:uppercase;
      letter-spacing:0.08em;
    }
    .dashboard-tile-description {
      margin:0;
      font-size:13px;
      line-height:1.5;
      color:var(--admin-muted);
    }
    .dashboard-tile-meta {
      display:flex;
      align-items:center;
      gap:8px;
      font-size:12px;
      color:var(--admin-muted);
    }
    .dashboard-tile-meta-dot {
      width:8px;
      height:8px;
      border-radius:50%;
      background:var(--dashboard-tile-accent);
      box-shadow:0 0 0 4px color-mix(in srgb, var(--dashboard-tile-accent) 15%, transparent);
      flex-shrink:0;
    }
    .dashboard-tile-action {
      font-size:13px;
      font-weight:700;
      color:var(--dashboard-tile-accent);
    }
    .dashboard-tile-arrow {
      color:var(--dashboard-tile-accent);
      font-size:18px;
      line-height:1;
      transition:transform 140ms ease;
    }
    .dashboard-tile:hover .dashboard-tile-arrow,
    .dashboard-tile:focus-visible .dashboard-tile-arrow { transform:translateX(2px); }
    .sidebar-footer {
      padding:14px 18px 18px;
      border-top:1px solid rgba(255,255,255,.1);
      display:flex;
      align-items:center;
      justify-content:space-between;
      gap:8px;
      flex-shrink:0;
    }
    .sidebar-footer-text {
      font-size:12px;
      color:rgba(255,255,255,.48);
      white-space:nowrap;
    }
    .sidebar-footer-link {
      display:inline-flex;
      align-items:center;
      gap:5px;
      font-size:13px;
      color:rgba(255,255,255,.68);
      background:transparent;
      border:1px solid transparent;
      padding:6px 10px;
      cursor:pointer;
      border-radius:0.35rem;
    }
    .sidebar-footer-link:hover { filter:none; color:#fff; background:rgba(255,255,255,.08); }
    .topbar-action-badge {
      position:absolute;
      top:4px;
      right:2px;
      min-width:18px;
      height:18px;
      padding:0 5px;
      border-radius:999px;
      display:grid;
      place-items:center;
      font-size:10px;
      font-weight:700;
      color:#fff;
      background:#dc3545;
    }
    .topbar-action-badge.warning { background:#ffc107; color:#212529; }
    .main-header.topbar { margin-left:0; width:100%; }
    .topbar .topbar-link,
    .topbar .topbar-action,
    .topbar .topbar-user-btn { color:inherit; }
    .topbar .topbar-link:hover,
    .topbar .topbar-action:hover,
    .topbar .topbar-user-btn:hover { color:var(--admin-primary); }
    .topbar-search-wrap.navbar-search-block { position:relative; display:flex; align-items:center; margin-bottom:0; }
    .sidebar-brand { height:auto; color:inherit; border-bottom:1px solid rgba(255,255,255,.1); }
    .sidebar-user-panel { margin:0; padding:14px 16px 14px 20px; border-bottom:1px solid rgba(255,255,255,.1); }
    .sidebar-search.input-group { margin-bottom:0; }
    .sidebar-search.input-group .form-control,
    .sidebar-search.input-group .btn { min-height:38px; }
    .sidebar-search.input-group .btn { border-color:rgba(255,255,255,.1); }
    .workspace.content-wrapper { margin-left:0; min-height:auto; background:transparent; padding:0; }
    .workspace.content-wrapper > * { width:100%; }
    .session-panel.login-box { width:100%; max-width:none; }
    .panel.card,
    .detail-card.card { margin-bottom:0; }
    .dashboard-tile.small-box {
      position:relative;
      min-height:238px;
      margin-bottom:0;
      background:
        radial-gradient(circle at top right, color-mix(in srgb, var(--dashboard-tile-accent) 14%, transparent) 0, transparent 34%),
        linear-gradient(180deg, #ffffff 0%, #fbfcfe 100%);
      box-shadow:0 10px 20px rgba(31,45,61,0.08);
    }
    .table-shell.table-responsive { border-radius:var(--admin-radius); }
    .section-shell .table-shell {
      border-radius:0.35rem;
      box-shadow:none;
      border-color:#d8dee4;
    }
    .section-shell .table-shell table { min-width:760px; }
    .section-shell .pagination-bar { margin-top:2px; }
    .btn-sidebar { background:rgba(255,255,255,.08); color:var(--admin-sidebar-text); }
    [data-theme="dark"] .main-header,
    [data-theme="dark"] .content-wrapper,
    [data-theme="dark"] .card,
    [data-theme="dark"] .brand-link { background:var(--admin-surface); color:var(--admin-text); }
    [data-theme="dark"] .workspace-header-main,
    [data-theme="dark"] .section-card-header,
    [data-theme="dark"] .section-card-footer {
      background:#1f2430;
      border-color:var(--admin-border);
      box-shadow:none;
    }
    [data-theme="dark"] .workspace-actions { background:var(--admin-surface); }
    [data-theme="dark"] .workspace-actions-copy strong { color:var(--admin-text); }
    [data-theme="dark"] .content-header-breadcrumb { color:var(--admin-muted); }
    [data-theme="dark"] .section-shell .table-shell { border-color:var(--admin-border); }
    [data-theme="dark"] .dashboard-tile.small-box {
      background:
        radial-gradient(circle at top right, color-mix(in srgb, var(--dashboard-tile-accent) 16%, transparent) 0, transparent 34%),
        linear-gradient(180deg, rgba(26,29,39,0.98) 0%, rgba(26,29,39,0.92) 100%);
      box-shadow:none;
    }
    [data-theme="dark"] .dashboard-tile-badge,
    [data-theme="dark"] .dashboard-tile-icon-badge { background:color-mix(in srgb, var(--dashboard-tile-accent) 20%, #1b2333); }
    [data-theme="dark"] .dashboard-tile-description,
    [data-theme="dark"] .dashboard-tile-count-label,
    [data-theme="dark"] .dashboard-tile-meta { color:var(--admin-muted); }
    [data-theme="dark"] .btn-sidebar { background:#22253a; color:var(--admin-muted); border-color:var(--admin-border); }
    body.sidebar-collapsed .sidebar-shell { display:none !important; }
    @media (min-width: 1121px) {
      body.standalone-admin-page.sidebar-collapsed .topbar,
      body.legacy-prototype-page.sidebar-collapsed .topbar {
        width:100%;
        margin-left:0;
      }
      body.standalone-admin-page.sidebar-collapsed .app-main,
      body.legacy-prototype-page.sidebar-collapsed .app-main {
        margin-left:0;
      }
    }
    body.standalone-login-page {
      background:
        radial-gradient(circle at top left, rgba(0, 123, 255, 0.12), transparent 36%),
        linear-gradient(180deg, #eef2f6 0%, #f4f6f9 48%, #eef1f4 100%);
    }
    body.standalone-login-page .topbar,
    body.standalone-login-page .app-main { max-width:1200px; margin:0 auto; width:100%; }
    body.standalone-login-page .topbar { padding-top:24px; border-bottom:none; box-shadow:none; background:transparent; position:static; }
    body.standalone-login-page .topbar-nav,
    body.standalone-login-page .topbar-actions,
    body.standalone-login-page .topbar-toggle { display:none; }
    body.standalone-login-page .login-shell { gap:24px; grid-template-columns: minmax(0, 1.15fr) minmax(360px, 420px); align-items:stretch; }
    body.standalone-login-page .login-marketing,
    body.standalone-login-page .login-lead,
    body.standalone-login-page .login-credentials { display:grid; }
    body.standalone-login-page .session-panel { margin:0; padding:28px; border-top-color:var(--admin-primary); }
    body.standalone-login-page #loginForm { grid-template-columns:1fr; gap:14px; }
    body.standalone-login-page input { min-height:46px; }
    body.standalone-login-page button { min-height:46px; }
    body.standalone-login-page #loginButton { width:100%; }
    body.standalone-admin-page .topbar,
    body.legacy-prototype-page .topbar {
      background:#fff;
    }
    @media (min-width: 1121px) {
      body.standalone-admin-page .topbar,
      body.legacy-prototype-page .topbar {
        width:calc(100% - var(--admin-sidebar-width));
        margin-left:var(--admin-sidebar-width);
      }
      body.standalone-admin-page .app-main,
      body.legacy-prototype-page .app-main {
        margin-left:var(--admin-sidebar-width);
      }
      body.standalone-admin-page .admin-shell,
      body.legacy-prototype-page .admin-shell {
        grid-template-columns:minmax(0, 1fr);
      }
      body.standalone-admin-page .sidebar-shell,
      body.legacy-prototype-page .sidebar-shell {
        position:fixed;
        top:0;
        left:0;
        bottom:0;
        width:var(--admin-sidebar-width);
        min-height:100vh;
        border-radius:0;
        z-index:35;
      }
    }
    body.standalone-admin-page .topbar-brand,
    body.legacy-prototype-page .topbar-brand {
      display:none;
    }
    @media (min-width: 1180px) {
      .detail-layout { grid-template-columns: minmax(0, 1.1fr) minmax(300px, 0.9fr); }
    }
    @media (max-width: 1120px) {
      .admin-shell { grid-template-columns:1fr; }
      .sidebar-shell {
        position:static;
        min-height:0;
      }
    }
    @media (max-width: 960px) {
      body.standalone-login-page .login-shell { grid-template-columns:1fr; }
      .topbar, .app-main, body.standalone-login-page .topbar, body.standalone-login-page .app-main { padding-left:16px; padding-right:16px; }
      .topbar { flex-direction:column; align-items:flex-start; }
      .topbar-left, .topbar-meta { width:100%; }
      .topbar-nav { flex-wrap:wrap; }
      .topbar-meta { justify-content:flex-start; margin-left:0; }
      .workspace-header-main,
      .workspace-actions,
      .section-card-header,
      .section-card-footer { padding-left:16px; padding-right:16px; }
      .table-toolbar .row-actions { flex-basis:100%; }
    }
  </style>
</head>
<body class="sidebar-mini layout-fixed">
  <div class="wrapper">
  <div id="toastContainer" class="toast-container" aria-live="polite" aria-atomic="false"></div>
  <header class="topbar main-header navbar navbar-expand navbar-white navbar-light elevation-1">
    <div class="topbar-left">
      <button class="topbar-toggle nav-link" type="button" aria-label="Toggle navigation" data-widget="pushmenu" role="button"><span aria-hidden="true">☰</span></button>
      <nav class="topbar-nav navbar-nav" aria-label="Admin navigation shortcuts">
        <a class="topbar-link nav-link" href="/admin"><span>Home</span></a>
      </nav>
      <div class="topbar-brand brand-link">
        <span class="brand-mark">A</span>
        <div class="topbar-copy">
          <span id="shellEyebrow" class="eyebrow">Admin Console</span>
          <h1 id="pageTitle">Gin Ninja Admin</h1>
          <p id="pageIntro" class="muted">A metadata-driven admin UI for the example admin APIs.</p>
        </div>
      </div>
    </div>
    <div class="topbar-meta">
      <div class="topbar-actions" aria-label="Admin quick actions">
        <div class="topbar-search-wrap navbar-search-block">
          <button class="topbar-action topbar-search-toggle nav-link" type="button" aria-label="Toggle search" id="topbarSearchToggle">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="6.5" cy="6.5" r="4.5" stroke="currentColor" stroke-width="1.5"></circle><path d="M10 10l4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"></path></svg>
          </button>
          <div id="topbarSearchExpand" class="topbar-search-expand" role="search">
            <input type="search" id="topbarSearchInput" class="form-control form-control-navbar" placeholder="Search all resources…" aria-label="Site-wide search" aria-autocomplete="list" aria-controls="topbarSearchResults" autocomplete="off">
            <div id="topbarSearchResults" class="topbar-search-results" role="listbox" aria-label="Search results"></div>
          </div>
        </div>
        <button class="topbar-action nav-link" type="button" aria-label="Toggle dark mode" id="darkModeToggle">
          <svg id="darkModeIconMoon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16" aria-hidden="true"><path d="M6 .278a.77.77 0 0 1 .08.858 7.2 7.2 0 0 0-.878 3.46c0 4.021 3.278 7.277 7.318 7.277q.792-.001 1.533-.16a.79.79 0 0 1 .81.316.73.73 0 0 1-.031.893A8.35 8.35 0 0 1 8.344 16C3.734 16 0 12.286 0 7.71 0 4.266 2.114 1.312 5.124.06A.75.75 0 0 1 6 .278"/></svg>
          <svg id="darkModeIconSun" xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16" aria-hidden="true" hidden><path d="M8 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8M8 0a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 0m0 13a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 13m8-5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2a.5.5 0 0 1 .5.5M3 8a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2A.5.5 0 0 1 3 8m10.657-5.657a.5.5 0 0 1 0 .707l-1.414 1.415a.5.5 0 1 1-.707-.708l1.414-1.414a.5.5 0 0 1 .707 0m-9.193 9.193a.5.5 0 0 1 0 .707L3.05 13.657a.5.5 0 0 1-.707-.707l1.414-1.414a.5.5 0 0 1 .707 0m9.193 2.121a.5.5 0 0 1-.707 0l-1.414-1.414a.5.5 0 0 1 .707-.707l1.414 1.414a.5.5 0 0 1 0 .707M4.464 4.465a.5.5 0 0 1-.707 0L2.343 3.05a.5.5 0 1 1 .707-.707l1.414 1.414a.5.5 0 0 1 0 .708"/></svg>
        </button>
        <div class="topbar-user-dropdown nav-item dropdown" id="topbarUserDropdown" hidden>
          <button class="topbar-user-btn nav-link" type="button" aria-label="User menu" aria-haspopup="true" aria-expanded="false" id="topbarUserBtn">
            <span class="topbar-user-avatar" id="topbarUserAvatar">?</span>
            <span class="topbar-user-name" id="topbarUserName">Guest</span>
            <svg class="topbar-caret" xmlns="http://www.w3.org/2000/svg" width="10" height="10" fill="currentColor" viewBox="0 0 16 16" aria-hidden="true"><path d="M7.247 11.14 2.451 5.658C1.885 5.013 2.345 4 3.204 4h9.592a1 1 0 0 1 .753 1.659l-4.796 5.48a1 1 0 0 1-1.506 0z"/></svg>
          </button>
          <ul class="topbar-user-menu dropdown-menu dropdown-menu-right" id="topbarUserMenu" hidden role="menu">
            <li role="none"><button id="clearToken" class="topbar-user-menu-item dropdown-item" type="button" role="menuitem">Sign out</button></li>
          </ul>
        </div>
      </div>
      <div id="sessionActions" hidden></div>
    </div>
  </header>
  <main class="app-main">
    <div id="status" class="visually-hidden" aria-live="polite" aria-atomic="true">Ready.</div>
    <section id="sessionShell" class="login-shell">
      <section class="panel login-marketing card card-outline card-secondary">
        <span class="eyebrow">Admin Console</span>
        <div class="stack">
          <h2>An AdminLTE-inspired sign-in for the standalone admin console.</h2>
          <p>Use a dedicated entrypoint to authenticate, then jump straight into the example back-office experience.</p>
        </div>
        <div class="login-feature-list">
          <div class="login-feature">
            <strong>Dashboard-style entrypoint</strong>
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
      <section class="panel stack session-panel login-box card card-outline card-primary">
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
            <input id="loginEmail" class="form-control" type="email" placeholder="alice@example.com" autocomplete="username email">
          </label>
          <label>Password
            <input id="loginPassword" class="form-control" type="password" placeholder="password123" autocomplete="current-password">
          </label>
          <div class="row-actions">
            <button id="loginButton" class="btn btn-primary" type="submit">Sign in</button>
          </div>
        </form>
        <div id="manualTokenTools" class="stack">
          <label>JWT token
            <input id="token" class="form-control" placeholder="Paste a token from /api/v1/auth/login" autocomplete="off">
          </label>
          <p class="muted">Successful sign-in stores the JWT in localStorage and attaches it to every admin request automatically.</p>
        </div>
      </section>
    </section>
    <section id="adminShell" class="admin-shell" hidden>
      <aside class="panel resource-strip stack sidebar-shell">
        <div class="sidebar-brand brand-link">
          <span class="sidebar-brand-mark">G</span>
          <div class="sidebar-brand-copy">
            <strong>Gin Ninja</strong>
            <span>Admin console</span>
          </div>
        </div>
        <div class="sidebar-user-panel user-panel">
          <span class="sidebar-user-avatar">AP</span>
          <div class="sidebar-user-copy">
            <strong>Alexander Pierce</strong>
            <span class="sidebar-user-status">
              <span class="sidebar-user-online" aria-hidden="true"></span>
              Online
            </span>
          </div>
        </div>
        <div class="sidebar-search input-group input-group-sm sidebar-search-form">
          <input id="sidebarResourceSearch" class="form-control form-control-sidebar" type="search" placeholder="Search" aria-label="Search sidebar navigation">
          <div class="input-group-append">
            <button id="sidebarResourceSearchButton" class="btn btn-sidebar" type="button" aria-label="Clear sidebar search">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="6.5" cy="6.5" r="4.5" stroke="currentColor" stroke-width="1.5"></circle><path d="M10 10l4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"></path></svg>
            </button>
          </div>
        </div>
        <div class="resource-strip-header">
          <div class="resource-strip-copy sidebar-heading">
            <span class="eyebrow subtle">Resource navigation</span>
            <h2>Switch workspaces</h2>
            <p class="muted">Move between admin resources from a left-hand menu while keeping the workspace focused.</p>
          </div>
        </div>
        <div class="sidebar-nav-shell">
          <div class="sidebar-section-label">Overview</div>
          <ul class="nav-list nav nav-pills nav-sidebar flex-column" role="menu">
            <li class="nav-item">
              <button id="sidebarDashboardLink" class="nav-link" type="button" aria-label="Open admin dashboard">
                <span class="nav-link-icon" aria-hidden="true">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M2.5 12.5h11"/><path d="M4.25 11.75V8.5"/><path d="M8 11.75V4.25"/><path d="M11.75 11.75V6.5"/><path d="M8 2.25a.75.75 0 1 1 0 1.5"/></svg>
                </span>
                <span class="nav-link-label">Dashboard</span>
              </button>
            </li>
          </ul>
          <div class="sidebar-section-label">Resources</div>
          <div class="sidebar-treeview open nav nav-pills nav-sidebar flex-column" id="resourceTreeview" data-widget="treeview" role="menu">
            <button class="sidebar-treeview-toggle" id="resourceTreeviewToggle" type="button" aria-expanded="true" aria-controls="resources">
              <span class="sidebar-treeview-toggle-copy">
                <span class="sidebar-treeview-toggle-icon" aria-hidden="true">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M2.5 4.5h4l1.4 1.5H13a1 1 0 0 1 1 1v4.5a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V5.5a1 1 0 0 1 .5-1z"/></svg>
                </span>
                <span class="sidebar-treeview-toggle-text">Resources</span>
              </span>
              <span class="sidebar-treeview-toggle-badge" id="resourceTreeviewBadge">0</span>
              <span class="sidebar-treeview-caret" aria-hidden="true">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M6 3.5 10.5 8 6 12.5"/></svg>
              </span>
            </button>
            <ul id="resources" class="nav-list nav nav-treeview sidebar-treeview-menu"></ul>
          </div>
        </div>
        <div class="sidebar-footer">
          <span class="sidebar-footer-text">v0.1 · Gin Ninja</span>
          <button id="sidebarSignOut" class="sidebar-footer-link btn btn-link" type="button" aria-label="Sign out"><span aria-hidden="true">⏻</span> Sign out</button>
        </div>
      </aside>
      <section class="workspace stack content-wrapper">
        <section id="workspaceHeader" class="panel workspace-header card card-outline card-primary">
          <div class="workspace-header-main">
            <div class="workspace-header-copy">
              <span class="workspace-header-kicker eyebrow">Workspace</span>
              <h2 id="resourceTitle">Select a resource</h2>
              <p id="resourcePath" class="workspace-path muted">Sign in to open a resource workspace.</p>
            </div>
            <ol class="content-header-breadcrumb" aria-label="Workspace breadcrumb">
              <li>Home</li>
              <li>Admin</li>
              <li>Workspace</li>
            </ol>
          </div>
          <div class="workspace-actions">
            <div class="workspace-actions-copy">
              <strong>AdminLTE workspace chrome</strong>
              <span>Operate resources with dashboard, filters, and table controls in one place.</span>
            </div>
            <button id="openCreateModal" class="btn btn-primary" type="button">Create record</button>
          </div>
        </section>
        <section id="dashboardShell" class="panel stack card card-outline card-info" hidden>
          <div class="card-header section-card-header">
            <div class="section-heading">
              <h3 class="section-title">Resources</h3>
              <p class="section-copy muted">Select a resource below to open its workspace.</p>
            </div>
            <div class="section-card-tools">
              <span class="eyebrow subtle">Dashboard cards</span>
            </div>
          </div>
          <div class="card-body section-card-body">
            <div id="dashboardTiles" class="dashboard-tiles"></div>
          </div>
        </section>
        <section class="content-grid">
          <section class="stack">
            <section id="recordsShell" class="panel section-shell card card-outline card-primary">
              <div class="card-header section-card-header">
                <div class="section-heading">
                  <h3 class="section-title">Records</h3>
                  <p class="section-copy muted">Search, filter, sort, and bulk manage the current resource.</p>
                </div>
                <div class="row-actions">
                  <span id="selectedCountBadge" class="badge">0 selected</span>
                </div>
              </div>
              <div class="card-body section-card-body">
                <div class="toolbar">
                  <div class="section-heading">
                    <span class="eyebrow subtle">Table tools</span>
                    <p class="section-copy muted">Apply fast filters, reload data, and run bulk actions from the record list.</p>
                  </div>
                  <div class="row-actions">
                    <button id="reloadList" class="secondary btn btn-default" type="button">Refresh list</button>
                    <button id="clearFilters" class="secondary btn btn-default" type="button">Clear filters</button>
                    <button id="bulkDelete" class="danger btn btn-danger" type="button">Bulk delete</button>
                  </div>
                </div>
                <div class="table-toolbar">
                  <div class="row-actions">
                    <input id="search" class="form-control" placeholder="Search current resource">
                    <select id="sort" class="custom-select"></select>
                    <select id="pageSize" class="custom-select">
                      <option value="5">5 / page</option>
                      <option value="10" selected>10 / page</option>
                      <option value="20">20 / page</option>
                      <option value="50">50 / page</option>
                    </select>
                  </div>
                  <div class="pagination-info" id="paginationInfo">Page 1 of 1</div>
                </div>
                <form id="filtersForm" class="filters"></form>
                <div id="list"></div>
                <div id="listLoading" class="list-loading" aria-live="polite" aria-label="Loading records">
                  <span class="list-spinner" aria-hidden="true"></span>
                  <span>Loading records…</span>
                </div>
              </div>
              <div class="card-footer section-card-footer">
                <div class="muted">Use filters to refine the current workspace.</div>
                <div class="row-actions">
                  <button id="prevPage" class="secondary btn btn-default" type="button">Previous</button>
                  <button id="nextPage" class="secondary btn btn-default" type="button">Next</button>
                </div>
              </div>
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
              <button id="closeCreateModal" type="button" class="secondary modal-close btn btn-default" aria-label="Close create record dialog">Close</button>
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
              <button id="closeRecordModal" type="button" class="secondary modal-close btn btn-default" aria-label="Close record dialog">Close</button>
            </div>
            <div class="modal-body">
              <div class="detail-layout">
                <section class="stack">
                  <div class="detail-card stack card card-outline card-primary">
                    <div class="toolbar">
                      <strong id="detailTitle">No record selected</strong>
                    </div>
                    <div id="detailFields" class="detail-grid">
                      <p class="muted">No record selected.</p>
                    </div>
                  </div>
                  <div class="detail-card stack card card-outline card-secondary">
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
              <button id="closeEditModal" type="button" class="secondary modal-close btn btn-default" aria-label="Close edit record dialog">Close</button>
            </div>
            <div class="modal-body">
              <form id="updateForm" class="stack"></form>
            </div>
          </div>
        </section>
        <section id="confirmModal" class="modal-overlay" hidden>
          <div class="modal-dialog confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="confirmModalTitle">
            <div class="modal-header">
              <div class="section-heading">
                <h3 id="confirmModalTitle" class="section-title">Confirm action</h3>
              </div>
              <button id="closeConfirmModal" type="button" class="secondary modal-close btn btn-default" aria-label="Close confirm dialog">Close</button>
            </div>
            <div class="modal-body">
              <p id="confirmModalMessage" class="muted"></p>
              <div class="confirm-actions">
                <button id="confirmModalCancel" type="button" class="secondary btn btn-default">Cancel</button>
                <button id="confirmModalConfirm" type="button" class="danger btn btn-danger">Delete</button>
              </div>
            </div>
          </div>
        </section>
      </section>
    </section>
  </main>
  </div>
  <script>
    const apiBase = '/api/v1/admin';
    const tokenStorageKey = 'gin-ninja-admin-token';
    const flashStorageKey = 'gin-ninja-admin-flash';
    const themeStorageKey = 'gin-ninja-admin-theme';
    const toastDefaultDurationMs = 4000;
    const globalSearchDebounceMs = 350;
    const stringLikeComponents = new Set(['text', 'textarea', 'email']);
    const adminPagePath = '/admin';
    const adminLoginPath = '/admin/login';
    const prototypePagePath = '/admin-prototype';
    const numericFieldPattern = /^-?\d+(?:\.\d+)?$/;
    const dashboardCountPlaceholder = '—';
    let pendingConfirmCallback = null;
    const state = {
      auth: { name: '', userID: null },
      current: null,
      meta: null,
      resources: [],
      resourceSearch: '',
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
      sidebarDashboardLink: document.getElementById('sidebarDashboardLink'),
      resourceTreeviewBadge: document.getElementById('resourceTreeviewBadge'),
      resources: document.getElementById('resources'),
      sidebarResourceSearch: document.getElementById('sidebarResourceSearch'),
      sidebarResourceSearchButton: document.getElementById('sidebarResourceSearchButton'),
      toastContainer: document.getElementById('toastContainer'),
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
      reloadList: document.getElementById('reloadList'),
      clearFilters: document.getElementById('clearFilters'),
      bulkDelete: document.getElementById('bulkDelete'),
      search: document.getElementById('search'),
      listLoading: document.getElementById('listLoading'),
      workspaceHeader: document.getElementById('workspaceHeader'),
      recordsShell: document.getElementById('recordsShell'),
      dashboardShell: document.getElementById('dashboardShell'),
      dashboardTiles: document.getElementById('dashboardTiles'),
      confirmModal: document.getElementById('confirmModal'),
      closeConfirmModal: document.getElementById('closeConfirmModal'),
      confirmModalCancel: document.getElementById('confirmModalCancel'),
      confirmModalConfirm: document.getElementById('confirmModalConfirm'),
      confirmModalTitle: document.getElementById('confirmModalTitle'),
      confirmModalMessage: document.getElementById('confirmModalMessage'),
      darkModeToggle: document.getElementById('darkModeToggle'),
      darkModeIconMoon: document.getElementById('darkModeIconMoon'),
      darkModeIconSun: document.getElementById('darkModeIconSun'),
      topbarSearchInput: document.getElementById('topbarSearchInput'),
      topbarSearchResults: document.getElementById('topbarSearchResults')
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

    function showToast(message, tone, durationMs) {
      if (!els.toastContainer) return;
      const toast = document.createElement('div');
      toast.className = 'toast';
      toast.setAttribute('role', 'status');
      toast.dataset.tone = tone || inferStatusTone(message);
      const msg = document.createElement('span');
      msg.className = 'toast-message';
      msg.textContent = message;
      const closeBtn = document.createElement('button');
      closeBtn.type = 'button';
      closeBtn.className = 'toast-close';
      closeBtn.textContent = '×';
      closeBtn.setAttribute('aria-label', 'Dismiss notification');
      closeBtn.onclick = () => toast.remove();
      toast.appendChild(msg);
      toast.appendChild(closeBtn);
      els.toastContainer.appendChild(toast);
      const timeout = durationMs !== undefined ? durationMs : toastDefaultDurationMs;
      if (timeout > 0) {
        setTimeout(() => { if (toast.parentNode) toast.remove(); }, timeout);
      }
    }

    function currentPagePath() {
      return window.location.pathname || '';
    }

    function buildNavigationState(view, resourceName) {
      return {
        pagePath: currentPagePath(),
        view: view || 'dashboard',
        resourceName: resourceName || ''
      };
    }

    function sameNavigationState(a, b) {
      return !!a && !!b &&
        (a.pagePath || '') === (b.pagePath || '') &&
        (a.view || '') === (b.view || '') &&
        (a.resourceName || '') === (b.resourceName || '');
    }

    function updateNavigationState(mode, view, resourceName) {
      if (!window.history) return;
      const nextState = buildNavigationState(view, resourceName);
      const currentState = window.history.state;
      if (sameNavigationState(currentState, nextState)) return;
      if (mode === 'push' && typeof window.history.pushState === 'function') {
        window.history.pushState(nextState, '', nextState.pagePath);
        return;
      }
      if (typeof window.history.replaceState === 'function') {
        window.history.replaceState(nextState, '', nextState.pagePath);
      }
    }

    async function restoreNavigationState(navState) {
      if (!state.resources.length) return false;
      try {
        if (navState?.view === 'dashboard') {
          showDashboard({ history: 'none' });
          return true;
        }
        if (!navState?.resourceName) return false;
        const resource = state.resources.find((item) => item.name === navState.resourceName);
        if (!resource) return false;
        await selectResource(resource, { history: 'none' });
        return true;
      } catch (error) {
        console.error('navigation state restore failed:', error);
        return false;
      }
    }

    function applyTheme(dark) {
      if (dark) {
        document.documentElement.setAttribute('data-theme', 'dark');
      } else {
        document.documentElement.removeAttribute('data-theme');
      }
      document.body.classList.toggle('dark-mode', dark);
      if (els.darkModeIconMoon) els.darkModeIconMoon.hidden = dark;
      if (els.darkModeIconSun) els.darkModeIconSun.hidden = !dark;
      if (els.darkModeToggle) els.darkModeToggle.setAttribute('aria-pressed', String(dark));
    }

    function toggleDarkMode() {
      const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
      const next = !isDark;
      applyTheme(next);
      try {
        if (next) {
          localStorage.setItem(themeStorageKey, 'dark');
        } else {
          localStorage.removeItem(themeStorageKey);
        }
      } catch (_) {
        // localStorage may be unavailable in some contexts
      }
    }

    function restoreTheme() {
      try {
        const saved = localStorage.getItem(themeStorageKey);
        if (saved === 'dark') {
          applyTheme(true);
          return true;
        }
      } catch (_) {
        // ignore
      }
      applyTheme(false);
      return false;
    }

    let globalSearchTimer = null;

    function closeGlobalSearch() {
      if (els.topbarSearchResults) {
        els.topbarSearchResults.classList.remove('has-results');
        els.topbarSearchResults.innerHTML = '';
      }
    }

    function recordDisplayLabel(record, fields) {
      // Pick the first string-like field that isn't the primary key for a summary
      const strField = (fields || []).find((f) => f.name !== 'id' && (stringLikeComponents.has(f.component) || !f.component));
      if (strField) {
        const val = record[strField.name];
        if (val !== undefined && val !== null && val !== '') return String(val);
      }
      // Fallback: first non-id field
      const keys = Object.keys(record).filter((k) => k !== 'id');
      if (keys.length) return String(record[keys[0]]);
      return '';
    }

    async function globalSearch(query) {
      closeGlobalSearch();
      if (!els.topbarSearchResults) return;
      const q = query.trim();
      if (!q || q.length < 2) return;
      if (!state.resources.length) return;

      const results = await Promise.all(
        state.resources.map(async (resource) => {
          try {
            const basePath = apiBase + '/resources' + resource.path;
            const data = await request(basePath + '?page=1&size=5&search=' + encodeURIComponent(q));
            const items = data.items || data.results || data.data || [];
            return { resource, items };
          } catch (_) {
            return { resource, items: [] };
          }
        })
      );

      const noResults = results.every((r) => r.items.length === 0);
      if (noResults) {
        els.topbarSearchResults.innerHTML = '<div class="search-results-empty">No results for &ldquo;' + escapeHTML(q) + '&rdquo;</div>';
        els.topbarSearchResults.classList.add('has-results');
        return;
      }

      results.forEach(({ resource, items }) => {
        if (!items.length) return;
        const group = document.createElement('div');
        group.className = 'search-results-group';
        const label = document.createElement('div');
        label.className = 'search-results-group-label';
        label.textContent = resource.label || resource.name;
        group.appendChild(label);

        items.forEach((record) => {
          const btn = document.createElement('button');
          btn.type = 'button';
          btn.className = 'search-result-item';
          btn.setAttribute('role', 'option');
          const pk = recordPrimaryKey(record);
          const displayLabel = recordDisplayLabel(record, []);
          const summary = document.createElement('span');
          summary.className = 'search-result-summary';
          summary.innerHTML = highlightMatch(displayLabel || String(pk ?? ''), q);
          const idSpan = document.createElement('span');
          idSpan.className = 'search-result-id';
          idSpan.textContent = '#' + String(pk ?? '');
          btn.appendChild(summary);
          if (displayLabel) btn.appendChild(idSpan);
          btn.addEventListener('click', async () => {
            if (els.topbarSearchInput) els.topbarSearchInput.value = '';
            const expandEl = document.getElementById('topbarSearchExpand');
            if (expandEl) expandEl.classList.remove('open');
            closeGlobalSearch();
            await selectResource(resource);
            // Try to open the specific record
            const found = (state.records || []).find((r) => String(recordPrimaryKey(r)) === String(pk));
            if (found) {
              state.selected = { item: found };
              renderSelectedRecord();
              await renderUpdateForm();
              openModal(els.editModal);
            }
          });
          group.appendChild(btn);
        });
        els.topbarSearchResults.appendChild(group);
      });

      if (els.topbarSearchResults.children.length) {
        els.topbarSearchResults.classList.add('has-results');
      }
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
      document.body.classList.toggle('standalone-admin-page', isStandaloneAdminPage());
      document.body.classList.toggle('legacy-prototype-page', isLegacyPrototypePage());
      document.body.classList.toggle('login-page', isStandaloneLoginPage());
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
      const topbarUserDropdown = document.getElementById('topbarUserDropdown');
      if (topbarUserDropdown) topbarUserDropdown.hidden = true;
    }

    function renderSignedInState() {
      const standaloneLoginPage = isStandaloneLoginPage();
      els.loginForm.hidden = true;
      els.sessionActions.hidden = standaloneLoginPage;
      els.sessionShell.hidden = true;
      els.manualTokenTools.hidden = true;
      els.adminShell.hidden = standaloneLoginPage;
      // Update user info in sidebar and topbar
      const name = state.auth.name || 'Admin';
      const initials = name.split(/\s+/).map(w => w[0] || '').slice(0, 2).join('').toUpperCase() || '?';
      const sidebarAvatar = document.querySelector('.sidebar-user-avatar');
      const sidebarName = document.querySelector('.sidebar-user-copy strong');
      if (sidebarAvatar) sidebarAvatar.textContent = initials;
      if (sidebarName) sidebarName.textContent = name;
      const topbarAvatar = document.getElementById('topbarUserAvatar');
      const topbarName = document.getElementById('topbarUserName');
      if (topbarAvatar) topbarAvatar.textContent = initials;
      if (topbarName) topbarName.textContent = name;
      const topbarUserDropdown = document.getElementById('topbarUserDropdown');
      if (topbarUserDropdown) topbarUserDropdown.hidden = standaloneLoginPage;
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
      state.resourceSearch = '';
      state.records = [];
      state.selected = null;
      state.bulkSelected = {};
      state.relationSearch = {};
      state.relationTimers = {};
      state.pagination = { page: 1, size: Number(els.pageSize.value || 10), pages: 1, total: 0 };
       if (els.sidebarResourceSearch) {
         els.sidebarResourceSearch.value = '';
       }
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

    function isMultiRelationField(field) {
      return !!(field && field.relation && field.type === 'array');
    }

    function selectedRelationValues(select, field) {
      if (!select) return isMultiRelationField(field) ? [] : '';
      if (isMultiRelationField(field)) {
        return Array.from(select.selectedOptions || []).map((option) => option.value).filter((value) => value !== '');
      }
      return select.value;
    }

    function relationSummaryText(field, select) {
      const selected = Array.from(select.selectedOptions || []);
      if (!selected.length) return 'Choose ' + field.label;
      const selectedLabels = selected
        .map((option) => option.textContent || '')
        .filter((label) => label && !label.startsWith('Selected: '));
      if (selectedLabels.length > 0 && selectedLabels.length === selected.length && selectedLabels.length <= 2) {
        return selectedLabels.join(', ');
      }
      return selected.length + ' selected';
    }

    function syncMultiRelationDropdown(field, select, summary, menu, items) {
      if (!summary || !menu || !isMultiRelationField(field)) return;
      summary.textContent = relationSummaryText(field, select);
      menu.innerHTML = '';
      if (!items.length) {
        const empty = document.createElement('div');
        empty.className = 'multi-relation-empty';
        empty.textContent = 'No matching options.';
        menu.appendChild(empty);
        return;
      }
      const selected = new Set(selectedRelationValues(select, field).map((value) => String(value)));
      items.forEach((item) => {
        const label = document.createElement('label');
        label.className = 'multi-relation-option';
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.value = String(item.value);
        checkbox.checked = selected.has(String(item.value));
        const text = document.createElement('span');
        text.textContent = item.label;
        checkbox.addEventListener('change', () => {
          let option = Array.from(select.options).find((candidate) => candidate.value === String(item.value));
          if (!option) {
            option = document.createElement('option');
            option.value = String(item.value);
            option.textContent = item.label;
            select.appendChild(option);
          }
          option.selected = checkbox.checked;
          summary.textContent = relationSummaryText(field, select);
        });
        label.appendChild(checkbox);
        label.appendChild(text);
        menu.appendChild(label);
      });
    }

    function resetQueryState() {
      state.bulkSelected = {};
      state.relationSearch = {};
      state.resourceSearch = '';
      state.pagination = { page: 1, size: Number(els.pageSize.value || 10), pages: 1, total: 0 };
      if (els.sidebarResourceSearch) {
        els.sidebarResourceSearch.value = '';
      }
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

    function filteredResources() {
      const term = String(state.resourceSearch || '').trim().toLowerCase();
      if (!term) return state.resources.slice();
      return state.resources.filter((resource) => {
        const label = String(resource.label || '').toLowerCase();
        const name = String(resource.name || '').toLowerCase();
        return label.includes(term) || name.includes(term);
      });
    }

    function renderResources() {
      els.resources.innerHTML = '';
      const resourceTreeview = document.getElementById('resourceTreeview');
      if (resourceTreeview) resourceTreeview.classList.add('open');
      if (els.resourceTreeviewBadge) els.resourceTreeviewBadge.textContent = String(state.resources.length);
      if (els.sidebarDashboardLink) {
        els.sidebarDashboardLink.classList.toggle('active', !state.current);
        els.sidebarDashboardLink.setAttribute('aria-current', !state.current ? 'page' : 'false');
      }
      const matches = filteredResources();
      if (!matches.length) {
        const li = document.createElement('li');
        li.className = 'sidebar-search-empty';
        li.textContent = state.resourceSearch
          ? ('No resources matched "' + String(state.resourceSearch || '').trim() + '".')
          : 'No resources available.';
        els.resources.appendChild(li);
      }
      matches.forEach((resource, index) => {
        const li = document.createElement('li');
        const button = document.createElement('button');
        const icon = document.createElement('span');
        const label = document.createElement('span');
        li.className = 'nav-item';
        button.type = 'button';
        button.className = 'nav-link' + (state.current?.name === resource.name ? ' active' : '');
        button.classList.add('d-flex', 'align-items-center');
        icon.className = 'nav-link-icon';
        icon.setAttribute('aria-hidden', 'true');
        icon.innerHTML = dashboardTileMeta(resource, index).icon;
        label.className = 'nav-link-label';
        label.innerHTML = highlightMatch(resource.label, state.resourceSearch);
        button.setAttribute('aria-current', state.current?.name === resource.name ? 'page' : 'false');
        button.appendChild(icon);
        button.appendChild(label);
        button.onclick = () => selectResource(resource);
        li.appendChild(button);
        els.resources.appendChild(li);
      });
      if (els.sidebarResourceSearchButton) {
        setSidebarSearchButtonContent(Boolean(state.resourceSearch));
        els.sidebarResourceSearchButton.setAttribute('aria-label', state.resourceSearch ? 'Clear sidebar search' : 'Focus sidebar search');
      }
    }

    function setSidebarSearchButtonContent(activeClear) {
      if (!els.sidebarResourceSearchButton) return;
      els.sidebarResourceSearchButton.replaceChildren();
      if (activeClear) {
        const span = document.createElement('span');
        span.setAttribute('aria-hidden', 'true');
        span.textContent = '×';
        els.sidebarResourceSearchButton.appendChild(span);
        return;
      }
      const svgNS = 'http://www.w3.org/2000/svg';
      const svg = document.createElementNS(svgNS, 'svg');
      svg.setAttribute('width', '14');
      svg.setAttribute('height', '14');
      svg.setAttribute('viewBox', '0 0 16 16');
      svg.setAttribute('fill', 'none');
      svg.setAttribute('aria-hidden', 'true');
      const circle = document.createElementNS(svgNS, 'circle');
      circle.setAttribute('cx', '6.5');
      circle.setAttribute('cy', '6.5');
      circle.setAttribute('r', '4.5');
      circle.setAttribute('stroke', 'currentColor');
      circle.setAttribute('stroke-width', '1.5');
      const path = document.createElementNS(svgNS, 'path');
      path.setAttribute('d', 'M10 10l4 4');
      path.setAttribute('stroke', 'currentColor');
      path.setAttribute('stroke-width', '1.5');
      path.setAttribute('stroke-linecap', 'round');
      svg.appendChild(circle);
      svg.appendChild(path);
      els.sidebarResourceSearchButton.appendChild(svg);
    }

     function openModal(modal) {
        if (!modal || modal.hidden) {
          if (modal) {
            modal.hidden = false;
            modal.classList.add('show');
         }
       }
        document.body.classList.add('modal-open');
      }

     function anyModalOpen() {
       return [els.createModal, els.recordModal, els.editModal, els.confirmModal].some((modal) => modal && !modal.hidden);
     }

     function closeModal(modal) {
        if (modal) {
          modal.classList.remove('show');
          modal.hidden = true;
        }
       if (!anyModalOpen()) {
         document.body.classList.remove('modal-open');
       }
     }

     function closeAllModals() {
       [els.createModal, els.recordModal, els.editModal, els.confirmModal].forEach((modal) => closeModal(modal));
     }

    let _actionMenuPortal = null;
    function getActionMenuPortal() {
      if (!_actionMenuPortal) {
        _actionMenuPortal = document.createElement('div');
        _actionMenuPortal.id = 'action-menu-portal';
        document.body.appendChild(_actionMenuPortal);
      }
      return _actionMenuPortal;
    }
    function closeActionMenuPortal() {
      const portal = getActionMenuPortal();
      portal.innerHTML = '';
      delete portal.dataset.forRow;
    }
    function openActionMenuAt(triggerEl, rowId, items) {
      const portal = getActionMenuPortal();
      const isOpen = portal.dataset.forRow === String(rowId) && portal.firstChild;
      closeActionMenuPortal();
      if (isOpen) return;
      const rect = triggerEl.getBoundingClientRect();
      const menu = document.createElement('div');
      menu.className = 'action-menu-list open';
      menu.style.cssText = 'position:fixed;z-index:1500;top:' + (rect.bottom + 4) + 'px;right:' + (window.innerWidth - rect.right) + 'px;left:auto;';
      items.forEach((item) => {
        if (item.divider) {
          const hr = document.createElement('hr');
          hr.className = 'action-menu-divider';
          menu.appendChild(hr);
        } else {
          const btn = document.createElement('button');
          btn.type = 'button';
          btn.className = 'action-menu-item' + (item.className ? ' ' + item.className : '');
          btn.textContent = item.label;
          btn.disabled = !!item.disabled;
          btn.onclick = (e) => { e.stopPropagation(); closeActionMenuPortal(); item.onClick(); };
          menu.appendChild(btn);
        }
      });
      portal.dataset.forRow = String(rowId);
      portal.appendChild(menu);
    }

    function openConfirmDialog(title, message, onConfirm, confirmLabel) {
      pendingConfirmCallback = onConfirm;
      els.confirmModalTitle.textContent = title;
      els.confirmModalMessage.textContent = message;
      els.confirmModalConfirm.textContent = confirmLabel || 'Confirm';
      openModal(els.confirmModal);
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

    function applySortFromHeader(field) {
      const current = els.sort.value;
      let next;
      if (current === field) {
        next = '-' + field;
      } else if (current === '-' + field) {
        next = '';
      } else {
        next = field;
      }
      els.sort.value = next;
      cancelScheduledSearchReload();
      resetToFirstPage();
      renderList().catch((error) => setStatus(String(error.message || error)));
    }

    function activeSortField() {
      const v = els.sort.value;
      if (!v) return { field: '', dir: '' };
      if (v.startsWith('-')) return { field: v.slice(1), dir: 'desc' };
      return { field: v, dir: 'asc' };
    }

    function setListLoading(active) {
      if (els.listLoading) els.listLoading.classList.toggle('active', active);
      if (active) els.list.innerHTML = '';
    }

    function resourceIcon(name) {
      const byName = {
        users: {
          icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M8 8a2.75 2.75 0 1 0 0-5.5A2.75 2.75 0 0 0 8 8Z"/><path d="M3.5 13.25a4.5 4.5 0 0 1 9 0"/></svg>'
        },
        roles: {
          icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M8 2.5 3.5 4.75v3c0 2.9 1.85 5.5 4.5 6.25 2.65-.75 4.5-3.35 4.5-6.25v-3z"/><path d="m6.5 8 1 1 2-2.25"/></svg>'
        },
        projects: {
          icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M3 4.5h10"/><path d="M5 2.75v3.5"/><path d="M11 2.75v3.5"/><rect x="3" y="4.5" width="10" height="8.5" rx="1.25"/><path d="M6 8h4"/><path d="M6 10.5h2.5"/></svg>'
        }
      };
      const normalizedName = String(name || '').toLowerCase();
      return (byName[normalizedName] || {
        icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect x="3" y="3.5" width="10" height="9" rx="1.25"/><path d="M6 6.5h4"/><path d="M6 9h4"/></svg>'
      }).icon;
    }

    function dashboardTileMeta(resource, index) {
      const palette = ['#007bff', '#17a2b8', '#6f42c1', '#fd7e14', '#20c997', '#e83e8c'];
      const byName = {
        users: {
          badge: 'Core access',
          description: 'Open the user workspace to review profiles, roles, and sign-in ready records.'
        },
        roles: {
          badge: 'Permissions',
          description: 'Inspect role definitions, capability groupings, and policy-oriented admin access.'
        },
        projects: {
          badge: 'Delivery',
          description: 'Jump into project workspaces with ownership context, progress tracking, and team-ready records.'
        }
      };
      const normalizedName = String(resource?.name || '').toLowerCase();
      const meta = byName[normalizedName] || {
        badge: 'Workspace',
        description: 'Open this admin resource to review records, filters, and available actions.'
      };
      return {
        accent: palette[index % palette.length],
        badge: meta.badge,
        description: meta.description,
        icon: resourceIcon(resource?.name)
      };
    }

    function renderDashboard() {
      if (!els.dashboardShell || !els.dashboardTiles) return;
      if (state.current || !state.resources.length) {
        els.dashboardShell.hidden = true;
        return;
      }
      els.dashboardShell.hidden = false;
      els.dashboardTiles.innerHTML = '';
      state.resources.forEach((resource, index) => {
        const meta = dashboardTileMeta(resource, index);
        const tile = document.createElement('button');
        tile.type = 'button';
        tile.className = 'dashboard-tile small-box bg-info';
        tile.style.setProperty('--dashboard-tile-accent', meta.accent);
        tile.setAttribute('aria-label', 'Open ' + resource.label + ' workspace');
        tile.innerHTML =
          '<div class="dashboard-tile-header">' +
            '<span class="dashboard-tile-badge">' + escapeHTML(meta.badge) + '</span>' +
            '<span class="dashboard-tile-icon-badge" aria-hidden="true">' + meta.icon + '</span>' +
          '</div>' +
          '<div class="dashboard-tile-main">' +
            '<div class="dashboard-tile-count-row">' +
              '<span class="dashboard-tile-count">' + dashboardCountPlaceholder + '</span>' +
              '<span class="dashboard-tile-count-label">records</span>' +
            '</div>' +
            '<span class="dashboard-tile-label">' + escapeHTML(resource.label) + '</span>' +
            '<span class="dashboard-tile-hint">' + escapeHTML(resource.name) + '</span>' +
            '<p class="dashboard-tile-description">' + escapeHTML(meta.description) + '</p>' +
            '<div class="dashboard-tile-meta"><span class="dashboard-tile-meta-dot" aria-hidden="true"></span><span>Connected admin workspace</span></div>' +
          '</div>' +
          '<div class="dashboard-tile-footer">' +
            '<span class="dashboard-tile-action">Open workspace</span>' +
            '<span class="dashboard-tile-arrow" aria-hidden="true">→</span>' +
          '</div>';
        tile.onclick = () => selectResource(resource);
        els.dashboardTiles.appendChild(tile);
        // Load record count in background for each tile
        const basePath = apiBase + '/resources' + resource.path;
        request(basePath + '?page=1&size=1')
          .then((data) => {
            const countEl = tile.querySelector('.dashboard-tile-count');
            if (countEl) countEl.textContent = String(data.total ?? dashboardCountPlaceholder);
          })
          .catch((err) => {
            // Count is decorative; log but don't surface to user
            console.error('dashboard tile count load failed for ' + resource.name + ':', err);
          });
      });
    }

    function buildFilterControl(field) {
      const wrapper = document.createElement('label');
      wrapper.className = 'inline-field form-group';
      wrapper.textContent = field.label;
      let input;
      if (field.component === 'checkbox') {
        input = document.createElement('select');
        input.className = 'custom-select';
        [['', 'Any'], ['true', 'True'], ['false', 'False']].forEach((pair) => {
          const option = document.createElement('option');
          option.value = pair[0];
          option.textContent = pair[1];
          input.appendChild(option);
        });
      } else if (field.component === 'number') {
        input = document.createElement('input');
        input.type = 'number';
        input.className = 'form-control';
      } else if (field.component === 'datetime') {
        input = document.createElement('input');
        input.type = 'datetime-local';
        input.className = 'form-control';
      } else {
        input = document.createElement('input');
        input.type = 'text';
        input.className = 'form-control';
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
      if (!preview) {
        return;
      }
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

    function resolveRelationSelection(field, items, selectedValue, term) {
      if (isMultiRelationField(field)) {
        const selected = Array.isArray(selectedValue) ? selectedValue.map((value) => String(value)) : [];
        return selected;
      }
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

    function populateRelationSelect(field, select, items, selectedValue, placeholderLabel) {
      const multiple = isMultiRelationField(field);
      select.multiple = multiple;
      select.removeAttribute('size');
      select.innerHTML = '';
      if (multiple) {
        const selectedSet = new Set((Array.isArray(selectedValue) ? selectedValue : []).map((value) => String(value)));
        items.forEach((item) => {
          const option = document.createElement('option');
          option.value = String(item.value);
          option.textContent = item.label;
          option.selected = selectedSet.has(String(item.value));
          select.appendChild(option);
        });
        selectedSet.forEach((value) => {
          if (Array.from(select.options).some((option) => option.value === value)) return;
          const option = document.createElement('option');
          option.value = value;
          option.textContent = 'Selected: ' + value;
          option.selected = true;
          select.appendChild(option);
        });
        return;
      }
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

    function scheduleRelationSearch(field, scopeKey, searchInput, select, preview, summary, menu) {
      const key = relationStateKey(scopeKey, field);
      clearTimeout(state.relationTimers[key]);
      state.relationTimers[key] = setTimeout(async () => {
        try {
          const term = searchInput.value.trim();
          const items = await loadRelationOptions(field, term, 1, 8);
          const nextValue = resolveRelationSelection(field, items, selectedRelationValues(select, field), term);
          populateRelationSelect(field, select, items, nextValue, field.label);
          syncMultiRelationDropdown(field, select, summary, menu, items);
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
        wrapper.className = 'relation-control form-group';
        const searchKey = relationStateKey(scopeKey, field);
        const searchInput = document.createElement('input');
        searchInput.type = 'text';
        searchInput.placeholder = 'Search related ' + field.label;
        searchInput.value = state.relationSearch[searchKey] || '';
        searchInput.className = 'form-control';
        const select = document.createElement('select');
        select.name = field.name;
        select.className = 'custom-select';
        const preview = isMultiRelationField(field) ? null : document.createElement('ul');
        if (preview) preview.className = 'relation-preview';
        const dropdown = isMultiRelationField(field) ? document.createElement('details') : null;
        const dropdownSummary = dropdown ? document.createElement('summary') : null;
        const dropdownMenu = dropdown ? document.createElement('div') : null;
        if (dropdown) {
          dropdown.className = 'multi-relation-dropdown';
          dropdownSummary.className = 'multi-relation-summary';
          dropdownMenu.className = 'multi-relation-menu';
          dropdown.appendChild(dropdownSummary);
          dropdown.appendChild(dropdownMenu);
          select.hidden = true;
        }
        const help = document.createElement('div');
        help.className = 'field-help';
        help.textContent = isMultiRelationField(field)
          ? 'Search related records and choose one or more matching options for this field.'
          : 'Search related records and choose the best matching option for this field.';
        wrapper.appendChild(searchInput);
        if (dropdown) {
          wrapper.appendChild(dropdown);
        } else {
          wrapper.appendChild(select);
        }
        if (preview) wrapper.appendChild(preview);
        if (dropdown) wrapper.appendChild(select);
        wrapper.appendChild(help);
        const items = await loadRelationOptions(field, searchInput.value.trim(), 1, 8);
        const nextValue = resolveRelationSelection(field, items, value, searchInput.value);
        populateRelationSelect(field, select, items, nextValue, field.label);
        syncMultiRelationDropdown(field, select, dropdownSummary, dropdownMenu, items);
        updateRelationPreview(preview, items, searchInput.value.trim());
        searchInput.addEventListener('input', () => {
          state.relationSearch[searchKey] = searchInput.value;
          scheduleRelationSearch(field, scopeKey, searchInput, select, preview, dropdownSummary, dropdownMenu);
        });
        select.addEventListener('change', () => {
          syncMultiRelationDropdown(field, select, dropdownSummary, dropdownMenu, items);
        });
        return wrapper;
      }
      if (field.component === 'checkbox') {
        const input = document.createElement('input');
        input.type = 'checkbox';
        input.name = field.name;
        input.checked = Boolean(value);
        input.className = 'form-check-input';
        return input;
      }
      if (field.component === 'array' || field.component === 'text' || field.component === 'textarea') {
        const input = document.createElement('textarea');
        input.name = field.name;
        input.className = 'form-control';
        input.value = field.component === 'array'
          ? (Array.isArray(value) ? JSON.stringify(value, null, 2) : (value ? JSON.stringify(value, null, 2) : ''))
          : (value == null ? '' : String(value));
        return input;
      }
      const input = document.createElement('input');
      input.name = field.name;
      input.type = ({ email: 'email', password: 'password', number: 'number', datetime: 'datetime-local' }[field.component]) || 'text';
      input.value = value == null ? '' : String(value);
      input.className = 'form-control';
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
        wrapper.className = 'form-field form-group';
        wrapper.textContent = field.label;
        const control = await buildFieldControl(field, values[name], scopeKey);
        wrapper.appendChild(control);
        target.appendChild(wrapper);
      }
      const submit = document.createElement('button');
      submit.type = 'submit';
      submit.textContent = mode === 'update' ? 'Update' : 'Create';
      submit.className = 'btn btn-primary';
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
          if (isMultiRelationField(field)) {
            if (!Array.isArray(payload[key])) payload[key] = [];
            if (value !== '') {
              payload[key].push(numericFieldPattern.test(value) ? Number(value) : value);
            }
            continue;
          }
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
      form.querySelectorAll('select[multiple][name]').forEach((select) => {
        const field = fieldMeta(select.name);
        if (!field || !isMultiRelationField(field) || select.disabled) return;
        payload[select.name] = Array.from(select.selectedOptions).map((option) => {
          const value = option.value;
          return numericFieldPattern.test(value) ? Number(value) : value;
        });
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
      setListLoading(true);
      let data;
      try {
        data = await request(currentBasePath() + buildListQuery());
      } finally {
        setListLoading(false);
      }
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
      tableShell.className = 'table-shell table-responsive p-0';
      const table = document.createElement('table');
      table.className = 'table table-bordered table-striped table-hover';
      const thead = document.createElement('thead');
      thead.className = 'thead-light';
      const headRow = document.createElement('tr');
      const bulkCell = document.createElement('th');
      const selectAll = document.createElement('input');
      selectAll.type = 'checkbox';
      selectAll.className = 'form-check-input position-static';
      selectAll.checked = rows.length > 0 && rows.every((row) => isSelectedForBulk(recordPrimaryKey(row)));
      selectAll.onchange = () => {
        rows.forEach((row) => setSelectedForBulk(recordPrimaryKey(row), selectAll.checked));
        syncBulkActionState();
        renderList().catch((error) => setStatus(String(error.message || error)));
      };
      bulkCell.appendChild(selectAll);
      headRow.appendChild(bulkCell);
      const sortable = new Set(state.meta?.sort_fields || []);
      const { field: sortField, dir: sortDir } = activeSortField();
      fields.forEach((field) => {
        const th = document.createElement('th');
        if (sortable.has(field)) {
          th.className = 'sortable-th' + (sortField === field ? ' sort-' + sortDir : '');
          th.setAttribute('aria-sort', sortField === field ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none');
          th.setAttribute('title', 'Click to sort by ' + field);
          const labelSpan = document.createElement('span');
          labelSpan.textContent = field;
          const iconSpan = document.createElement('span');
          iconSpan.className = 'sort-icon';
          iconSpan.setAttribute('aria-hidden', 'true');
          iconSpan.textContent = sortField === field ? (sortDir === 'asc' ? '▲' : '▼') : '⇅';
          th.appendChild(labelSpan);
          th.appendChild(iconSpan);
          th.onclick = () => applySortFromHeader(field);
        } else {
          th.textContent = field;
        }
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
        checkbox.className = 'form-check-input position-static';
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
        openButton.className = 'action-btn-view btn btn-default btn-sm';
        openButton.textContent = 'View';
        openButton.onclick = () => selectRecord(row, { openModal: 'record' });
        actionWrap.appendChild(openButton);
        // More (···) dropdown menu — uses portal to escape overflow:auto clipping
        const trigger = document.createElement('button');
        trigger.type = 'button';
        trigger.className = 'action-menu-trigger btn btn-default btn-sm dropdown-toggle';
        trigger.setAttribute('aria-label', 'More actions');
        trigger.textContent = '···';
        trigger.onclick = (e) => {
          e.stopPropagation();
          openActionMenuAt(trigger, id, [
            { label: 'Edit', disabled: !hasAction('update'), onClick: () => selectRecord(row, { openModal: 'edit' }) },
            { divider: true },
            { label: 'Delete', className: 'danger', disabled: !hasAction('delete'), onClick: () => deleteRecordByID(id) },
          ]);
        };
        actionWrap.appendChild(trigger);
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
          showToast(String(error.message || error), 'danger');
          setStatus(String(error.message || error));
        }
      }

      async function deleteRecordByID(id) {
        if (!state.current || id == null) return;
        openConfirmDialog(
          'Delete record',
          'Are you sure you want to permanently delete record #' + id + '? This action cannot be undone.',
          async () => {
            closeModal(els.confirmModal);
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
              showToast('Deleted record #' + id + '.', 'success');
              await reloadListWithStatus('Deleted record #' + id + '.', false);
            } catch (error) {
              showToast(String(error.message || error), 'danger');
              setStatus(String(error.message || error));
            }
          },
          'Delete'
        );
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
        renderDashboard();
        setStatus('Loaded ' + state.resources.length + ' resources.');
        if (await restoreNavigationState(window.history.state)) {
          return;
        }
        if (state.resources.length) {
          await selectResource(state.resources[0], { history: 'replace' });
          return;
        }
        updateNavigationState('replace', 'dashboard');
      } catch (error) {
        setStatus(String(error.message || error));
      }
    }

    async function selectResource(resource, options) {
      const navigationMode = options?.history || 'push';
      state.current = resource;
      state.selected = null;
      resetQueryState();
      renderDashboard();
      if (els.workspaceHeader) els.workspaceHeader.hidden = false;
      if (els.recordsShell) els.recordsShell.hidden = false;
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
        if (navigationMode !== 'none') {
          updateNavigationState(navigationMode, 'resource', resource.name);
        }
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

    // User dropdown toggle
    const topbarUserBtn = document.getElementById('topbarUserBtn');
    const topbarUserMenu = document.getElementById('topbarUserMenu');
    if (topbarUserBtn && topbarUserMenu) {
      topbarUserBtn.addEventListener('click', (event) => {
        event.stopPropagation();
        const open = topbarUserMenu.hidden === false;
        topbarUserMenu.hidden = open;
        topbarUserBtn.setAttribute('aria-expanded', String(!open));
      });
      document.addEventListener('click', () => {
        if (topbarUserMenu) topbarUserMenu.hidden = true;
        if (topbarUserBtn) topbarUserBtn.setAttribute('aria-expanded', 'false');
      });
    }

    // Topbar search toggle
    const topbarSearchToggle = document.getElementById('topbarSearchToggle');
    const topbarSearchExpand = document.getElementById('topbarSearchExpand');
    if (topbarSearchToggle && topbarSearchExpand) {
      topbarSearchToggle.addEventListener('click', (event) => {
        event.stopPropagation();
        topbarSearchExpand.classList.toggle('open');
        if (!topbarSearchExpand.classList.contains('open')) {
          closeGlobalSearch();
          return;
        }
        if (els.topbarSearchInput) els.topbarSearchInput.focus();
      });
      document.addEventListener('click', (event) => {
        if (topbarSearchExpand && !topbarSearchExpand.contains(event.target) && event.target !== topbarSearchToggle) {
          topbarSearchExpand.classList.remove('open');
          closeGlobalSearch();
        }
      });
    }

    function showDashboard(options) {
      const navigationMode = options?.history || 'push';
      state.current = null;
      state.meta = null;
      state.selected = null;
      resetQueryState();
      renderResources();
      renderDashboard();
      if (els.workspaceHeader) els.workspaceHeader.hidden = true;
      if (els.recordsShell) els.recordsShell.hidden = true;
      els.resourceTitle.textContent = 'Admin dashboard';
      els.resourcePath.textContent = 'Choose a resource from the sidebar to load its workspace.';
      els.detailTitle.textContent = 'No record selected';
      els.detailObjectBadge.textContent = 'Dashboard';
      els.detail.textContent = 'Choose a resource from the sidebar.';
      els.detailFields.innerHTML = '<p class="muted">Select a resource to inspect records, filters, and actions.</p>';
      els.createForm.innerHTML = '<p class="muted">Select a resource to create records.</p>';
      els.updateForm.innerHTML = '<p class="muted">Select a resource to edit records.</p>';
      els.editHint.textContent = 'Select a resource to open the change form.';
      els.list.innerHTML = '<div class="empty-state">Select a resource from the sidebar to load records.</div>';
      renderPagination();
      syncBulkActionState();
      syncWorkspaceActionState();
      if (navigationMode !== 'none') {
        updateNavigationState(navigationMode, 'dashboard');
      }
      setStatus('Showing Dashboard.');
    }
    window.addEventListener('popstate', (event) => {
      restoreNavigationState(event.state).catch((error) => {
        console.error('navigation state restore failed:', error);
        setStatus(String(error.message || error));
      });
    });
    if (els.topbarSearchInput) {
      els.topbarSearchInput.addEventListener('input', (event) => {
        clearTimeout(globalSearchTimer);
        const q = event.target.value;
        if (!q.trim() || q.trim().length < 2) {
          closeGlobalSearch();
          return;
        }
        globalSearchTimer = setTimeout(() => {
          globalSearch(q).catch((err) => console.error('global search error:', err));
        }, globalSearchDebounceMs);
      });
      els.topbarSearchInput.addEventListener('keydown', (event) => {
        if (event.key === 'Escape') {
          event.stopPropagation();
          els.topbarSearchInput.value = '';
          closeGlobalSearch();
          if (topbarSearchExpand) topbarSearchExpand.classList.remove('open');
        }
      });
    }
    // Topbar ☰ toggle: collapse / expand sidebar
    const topbarToggle = document.querySelector('.topbar-toggle');
    if (topbarToggle) {
      topbarToggle.addEventListener('click', (event) => {
        event.stopPropagation();
        document.body.classList.toggle('sidebar-collapsed');
      });
    }
    const sidebarSignOut = document.getElementById('sidebarSignOut');
    if (sidebarSignOut) {
      sidebarSignOut.addEventListener('click', () => {
        els.clearToken.click();
      });
    }
    const resourceTreeview = document.getElementById('resourceTreeview');
    const resourceTreeviewToggle = document.getElementById('resourceTreeviewToggle');
    if (resourceTreeview && resourceTreeviewToggle) {
      resourceTreeviewToggle.addEventListener('click', () => {
        const open = resourceTreeview.classList.toggle('open');
        resourceTreeviewToggle.setAttribute('aria-expanded', String(open));
      });
    }
    if (els.sidebarResourceSearch) {
      els.sidebarResourceSearch.addEventListener('input', () => {
        state.resourceSearch = els.sidebarResourceSearch.value.trim();
        renderResources();
      });
      els.sidebarResourceSearch.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && els.sidebarResourceSearch.value) {
          event.preventDefault();
          state.resourceSearch = '';
          els.sidebarResourceSearch.value = '';
          renderResources();
        }
      });
    }
    if (els.sidebarResourceSearchButton) {
      els.sidebarResourceSearchButton.addEventListener('click', () => {
        if (!els.sidebarResourceSearch) return;
        if (!els.sidebarResourceSearch.value) {
          els.sidebarResourceSearch.focus();
          return;
        }
        state.resourceSearch = '';
        els.sidebarResourceSearch.value = '';
        renderResources();
        els.sidebarResourceSearch.focus();
      });
    }
    if (els.sidebarDashboardLink) {
      els.sidebarDashboardLink.addEventListener('click', () => {
        showDashboard();
      });
    }
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
       if (event.key === 'Escape') {
         closeActionMenuPortal();
         closeAllModals();
         return;
       }
       // Ignore shortcuts when focus is in a text control or a modal is open
       const tag = document.activeElement ? document.activeElement.tagName : '';
       if (['INPUT', 'TEXTAREA', 'SELECT'].includes(tag)) return;
       if (anyModalOpen()) return;
       if (event.ctrlKey || event.metaKey || event.altKey) return;
       if (event.key === '/' && state.current) {
         event.preventDefault();
         if (els.search) els.search.focus();
       } else if (event.key === 'n' && !event.shiftKey) {
         event.preventDefault();
         if (!els.openCreateModal.disabled) els.openCreateModal.click();
       }
     });
     document.addEventListener('click', () => {
       closeActionMenuPortal();
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
        showToast('Created a new ' + state.current.name + ' record.', 'success');
        await reloadListWithStatus('Created a new ' + state.current.name + ' record.', true);
      } catch (error) {
        showToast(String(error.message || error), 'danger');
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
        showToast('Updated record #' + id + '.', 'success');
        setStatus('Updated record #' + id + '.');
      } catch (error) {
        showToast(String(error.message || error), 'danger');
        setStatus(String(error.message || error));
      }
    };
    els.bulkDelete.onclick = () => {
      if (!state.current || !selectedIDs().length) return;
      const count = selectedIDs().length;
      openConfirmDialog(
        'Bulk delete',
        'Are you sure you want to permanently delete ' + count + ' selected record(s)? This action cannot be undone.',
        async () => {
          closeModal(els.confirmModal);
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
            const deleted = String(result.deleted || 0);
            showToast('Bulk deleted ' + deleted + ' record(s).', 'success');
            await reloadListWithStatus('Bulk deleted ' + deleted + ' record(s).', false);
          } catch (error) {
            showToast(String(error.message || error), 'danger');
            setStatus(String(error.message || error));
          }
        },
        'Delete ' + count
      );
    };
    els.closeConfirmModal.onclick = () => { pendingConfirmCallback = null; closeModal(els.confirmModal); };
    els.confirmModalCancel.onclick = () => { pendingConfirmCallback = null; closeModal(els.confirmModal); };
    els.confirmModalConfirm.onclick = () => { const cb = pendingConfirmCallback; pendingConfirmCallback = null; if (cb) cb(); };
    els.confirmModal.addEventListener('click', (event) => {
      if (event.target === els.confirmModal) {
        pendingConfirmCallback = null;
        closeModal(els.confirmModal);
      }
    });
    if (els.darkModeToggle) {
      els.darkModeToggle.addEventListener('click', toggleDarkMode);
    }

    resetAdminState();
    updatePageChrome();
    restoreTheme();
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
