const adminTitle = __GIN_NINJA_ADMIN_TITLE_JS__;
const adminBrandName = __GIN_NINJA_ADMIN_BRAND_NAME_JS__;
const adminLogoText = __GIN_NINJA_ADMIN_LOGO_TEXT_JS__;
const adminLocale = __GIN_NINJA_ADMIN_LOCALE__;
const adminDefaultTheme = __GIN_NINJA_ADMIN_DEFAULT_THEME__;
const tokenStorageDriver = __GIN_NINJA_ADMIN_TOKEN_STORAGE__;
const apiBase = __GIN_NINJA_ADMIN_API_BASE__;
const tokenStorageKey = 'gin-ninja-admin-token';
const authIdentityStorageKey = 'gin-ninja-admin-auth-identity';
const flashStorageKey = 'gin-ninja-admin-flash';
const themeStorageKey = 'gin-ninja-admin-theme';
const adminThemeStorageKey = 'gin-ninja-admin-color-theme';
const columnVisibilityStoragePrefix = 'gin-ninja-admin-columns:';
const listStateStoragePrefix = 'gin-ninja-admin-list-state:';
const toastDefaultDurationMs = 4000;
const globalSearchDebounceMs = 350;
const adminThemes = [
  { value: 'claude', label: 'Claude' },
  { value: 'discord', label: 'Discord' },
  { value: 'supabase', label: 'Supabase' },
  { value: 'vercel', label: 'Vercel' },
  { value: 'mono', label: 'Mono' },
  { value: 'notebook', label: 'Notebook' },
  { value: 'light-green', label: 'Light Green' },
  { value: 'zen', label: 'Zen' },
  { value: 'astro-vista', label: 'Astro Vista' },
  { value: 'whatsapp', label: 'WhatsApp' }
];
const defaultAdminTheme = 'claude';
const stringLikeComponents = new Set(['text', 'textarea', 'email']);
const formControlTagNames = new Set(['INPUT', 'SELECT', 'TEXTAREA']);
const adminPagePath = __GIN_NINJA_ADMIN_PAGE_PATH__;
const adminLoginPath = __GIN_NINJA_ADMIN_LOGIN_PATH__;
const authMePath = __GIN_NINJA_ADMIN_AUTH_ME_PATH__;
const loginTokenExtractExpr = __GIN_NINJA_ADMIN_TOKEN_EXTRACT_EXPR__;
const loginNameExtractExpr = __GIN_NINJA_ADMIN_USER_NAME_EXTRACT_EXPR__;
const loginUserIDExtractExpr = __GIN_NINJA_ADMIN_USER_ID_EXTRACT_EXPR__;
function readPayloadPath(payload, expression) {
  const path = String(expression || '').trim();
  if (path === 'payload') return payload;
  if (!/^payload(?:\.[A-Za-z_$][\w$]*)+$/.test(path)) return undefined;
  return path.slice('payload.'.length).split('.').reduce((current, key) => {
    if (current == null || !Object.prototype.hasOwnProperty.call(Object(current), key)) return undefined;
    return current[key];
  }, payload);
}
function evaluatePayloadExtractExpr(payload, expression) {
  for (const fallback of String(expression || '').split('||')) {
    const terms = fallback.split('&&').map((term) => term.trim()).filter(Boolean);
    if (!terms.length) continue;
    let value;
    let matched = true;
    for (const term of terms) {
      value = readPayloadPath(payload, term);
      if (!value) {
        matched = false;
        break;
      }
    }
    if (matched) return value;
  }
  return undefined;
}
function extractLoginToken(payload) { return evaluatePayloadExtractExpr(payload, loginTokenExtractExpr); }
function extractLoginName(payload) { return evaluatePayloadExtractExpr(payload, loginNameExtractExpr); }
function extractLoginUserID(payload) { return evaluatePayloadExtractExpr(payload, loginUserIDExtractExpr); }
const numericFieldPattern = /^-?\d+(?:\.\d+)?$/;
const dashboardCountPlaceholder = '—';
let pendingConfirmCallback = null;
let modalScrollListenersActive = false;
let modalTouchStartY = 0;
const state = {
  auth: { name: '', userID: null, email: '', isAdmin: false, roles: [], expiresAt: '', issuedAt: '', issuer: '' },
  current: null,
  meta: null,
  resources: [],
  resourceSearch: '',
  records: [],
  selected: null,
  errorPage: null,
  bulkSelected: {},
  visibleColumns: {},
  relationSearch: {},
  relationTimers: {},
  listUpdatedAt: null,
  forms: {
    create: { dirty: false, initial: '', pending: false },
    update: { dirty: false, initial: '', pending: false }
  },
  searchTimer: null,
  pagination: { page: 1, size: 10, pages: 1, total: 0 }
};

 const els = {
  loginForm: document.getElementById('loginForm'),
  loginFeedback: document.getElementById('loginFeedback'),
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
  topbarBreadcrumbRoot: document.getElementById('topbarBreadcrumbRoot'),
  topbarBreadcrumbCurrent: document.getElementById('topbarBreadcrumbCurrent'),
  adminShell: document.getElementById('adminShell'),
  sidebarDashboardLink: document.getElementById('sidebarDashboardLink'),
  resources: document.getElementById('resources'),
  sidebarResourceSearch: document.getElementById('sidebarResourceSearch'),
  sidebarResourceSearchButton: document.getElementById('sidebarResourceSearchButton'),
  toastContainer: document.getElementById('toastContainer'),
  resourceTitle: document.getElementById('resourceTitle'),
  resourcePath: document.getElementById('resourcePath'),
  resourceActionSummary: document.getElementById('resourceActionSummary'),
  selectedCountBadge: document.getElementById('selectedCountBadge'),
  copySelectedIDs: document.getElementById('copySelectedIDs'),
  clearSelection: document.getElementById('clearSelection'),
  bulkActionBar: document.getElementById('bulkActionBar'),
  bulkActionCount: document.getElementById('bulkActionCount'),
  bulkActionLabel: document.getElementById('bulkActionLabel'),
  bulkActionClear: document.getElementById('bulkActionClear'),
  bulkActionExport: document.getElementById('bulkActionExport'),
  bulkActionDelete: document.getElementById('bulkActionDelete'),
  detailTitle: document.getElementById('detailTitle'),
  detailObjectBadge: document.getElementById('detailObjectBadge'),
  detailFields: document.getElementById('detailFields'),
  openSelectedEdit: document.getElementById('openSelectedEdit'),
  copyRecordJSON: document.getElementById('copyRecordJSON'),
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
  columnToggle: document.getElementById('columnToggle'),
  columnMenu: document.getElementById('columnMenu'),
  paginationInfo: document.getElementById('paginationInfo'),
  firstPage: document.getElementById('firstPage'),
  prevPage: document.getElementById('prevPage'),
  nextPage: document.getElementById('nextPage'),
  lastPage: document.getElementById('lastPage'),
  paginationPages: document.getElementById('paginationPages'),
  list: document.getElementById('list'),
  detail: document.getElementById('detail'),
  search: document.getElementById('search'),
  listLoading: document.getElementById('listLoading'),
  workspaceHeader: document.getElementById('workspaceHeader'),
  recordsShell: document.getElementById('recordsShell'),
  dashboardShell: document.getElementById('dashboardShell'),
  dashboardTiles: document.getElementById('dashboardTiles'),
  errorShell: document.getElementById('errorShell'),
  errorCode: document.getElementById('errorCode'),
  errorTitle: document.getElementById('errorTitle'),
  errorCopy: document.getElementById('errorCopy'),
  errorActions: document.getElementById('errorActions'),
  errorGoBack: document.getElementById('errorGoBack'),
  errorHome: document.getElementById('errorHome'),
  confirmModal: document.getElementById('confirmModal'),
  closeConfirmModal: document.getElementById('closeConfirmModal'),
  confirmModalCancel: document.getElementById('confirmModalCancel'),
  confirmModalConfirm: document.getElementById('confirmModalConfirm'),
  confirmModalTitle: document.getElementById('confirmModalTitle'),
  confirmModalMessage: document.getElementById('confirmModalMessage'),
  darkModeToggle: document.getElementById('darkModeToggle'),
  darkModeIconMoon: document.getElementById('darkModeIconMoon'),
  darkModeIconSun: document.getElementById('darkModeIconSun'),
  adminThemeToggle: document.getElementById('adminThemeToggle'),
  adminThemeLabel: document.getElementById('adminThemeLabel'),
  adminThemeMenu: document.getElementById('adminThemeMenu'),
  topbarSearchInput: document.getElementById('topbarSearchInput'),
  topbarSearchResults: document.getElementById('topbarSearchResults'),
  topbarProfileName: document.getElementById('topbarProfileName'),
  topbarProfileMeta: document.getElementById('topbarProfileMeta'),
  sidebarAuthStatus: document.getElementById('sidebarAuthStatus'),
  sidebarAuthMeta: document.getElementById('sidebarAuthMeta')
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

function extractErrorMessage(value) {
  if (!value) return '';
  if (typeof value === 'string') {
    try {
      return extractErrorMessage(JSON.parse(value));
    } catch (_) {
      return value;
    }
  }
  if (typeof value === 'object') {
    if (typeof value.message === 'string' && value.message.trim()) return value.message;
    if (typeof value.error === 'string' && value.error.trim()) return value.error;
    if (value.error) return extractErrorMessage(value.error);
  }
  return String(value);
}

function createHTTPError(status, message) {
  const error = new Error(message || ('Request failed with status ' + status + '.'));
  error.status = Number(status || 0);
  error.isHTTPError = true;
  return error;
}

function setLoginFeedback(message) {
  if (!els.loginFeedback) return;
  const value = extractErrorMessage(message);
  if (!value) {
    els.loginFeedback.hidden = true;
    els.loginFeedback.textContent = '';
    return;
  }
  els.loginFeedback.textContent = value;
  els.loginFeedback.hidden = false;
}

function updateModalLayoutMetrics() {
  const workspaceRect = document.querySelector('.app-main')?.getBoundingClientRect();
  const leftOffset = Math.max(0, Math.round(workspaceRect?.left || 0));
  document.documentElement.style.setProperty('--admin-modal-workspace-left', leftOffset + 'px');
}

function scrollableModalBodyFromEvent(event) {
  const target = event?.target;
  if (!target || typeof target.closest !== 'function') return null;
  return target.closest('.modal-overlay:not([hidden]) .modal-body');
}

function canScrollModalBody(element, deltaY) {
  if (!element) return false;
  if (deltaY < 0) return element.scrollTop > 0;
  if (deltaY > 0) return element.scrollTop + element.clientHeight < element.scrollHeight;
  return true;
}

function preventBackgroundWheel(event) {
  if (!document.body.classList.contains('modal-open')) return;
  if (canScrollModalBody(scrollableModalBodyFromEvent(event), event.deltaY)) return;
  event.preventDefault();
}

function preventBackgroundTouchStart(event) {
  if (!document.body.classList.contains('modal-open') || !event.touches?.length) return;
  modalTouchStartY = event.touches[0].clientY;
}

function preventBackgroundTouchMove(event) {
  if (!document.body.classList.contains('modal-open') || !event.touches?.length) return;
  const deltaY = modalTouchStartY - event.touches[0].clientY;
  modalTouchStartY = event.touches[0].clientY;
  if (canScrollModalBody(scrollableModalBodyFromEvent(event), deltaY)) return;
  event.preventDefault();
}

function enableModalScrollGuard() {
  if (modalScrollListenersActive) return;
  modalScrollListenersActive = true;
  document.addEventListener('wheel', preventBackgroundWheel, { passive: false });
  document.addEventListener('touchstart', preventBackgroundTouchStart, { passive: true });
  document.addEventListener('touchmove', preventBackgroundTouchMove, { passive: false });
}

function disableModalScrollGuard() {
  if (!modalScrollListenersActive) return;
  modalScrollListenersActive = false;
  document.removeEventListener('wheel', preventBackgroundWheel, { passive: false });
  document.removeEventListener('touchstart', preventBackgroundTouchStart, { passive: true });
  document.removeEventListener('touchmove', preventBackgroundTouchMove, { passive: false });
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

function normalizedPath(path) {
  let value = String(path || '/').trim();
  if (!value.startsWith('/')) value = '/' + value;
  value = value.replace(/\/+$/, '');
  return value || '/';
}

function adminRootPath() {
  return normalizedPath(adminPagePath);
}

function pathWithinAdmin(path) {
  const root = adminRootPath();
  const current = normalizedPath(path);
  return current === root || current.startsWith(root + '/');
}

function resourceNameFromAdminPath(path) {
  const root = adminRootPath();
  const current = normalizedPath(path);
  if (current === root || !current.startsWith(root + '/')) return '';
  const segment = current.slice(root.length + 1).split('/')[0] || '';
  if (!segment) return '';
  try {
    return decodeURIComponent(segment);
  } catch (_) {
    return segment;
  }
}

function buildNavigationState(view, resourceName) {
  return {
    pagePath: currentPagePath(),
    view: view || 'dashboard',
    resourceName: resourceName || ''
  };
}

function navigationStateFromURL() {
  if (isStandaloneLoginPage()) return null;
  const resourceName = resourceNameFromAdminPath(currentPagePath());
  if (resourceName) {
    return buildNavigationState('resource', resourceName);
  }
  if (pathWithinAdmin(currentPagePath())) {
    return buildNavigationState('dashboard');
  }
  return null;
}

function requestedResourceNameFromURL() {
  const navState = navigationStateFromURL();
  return navState?.view === 'resource' ? navState.resourceName : '';
}

function buildNavigationURL(view, resourceName, includeListState) {
  const params = new URLSearchParams();
  let path = adminRootPath();
  if (view === 'resource' && resourceName) {
    path += '/' + encodeURIComponent(resourceName);
    if (includeListState) {
      appendListQueryParams(params);
    }
  }
  const query = params.toString();
  return path + (query ? '?' + query : '');
}

function sameNavigationState(a, b) {
  return !!a && !!b &&
    (a.pagePath || '') === (b.pagePath || '') &&
    (a.view || '') === (b.view || '') &&
    (a.resourceName || '') === (b.resourceName || '');
}

function updateNavigationState(mode, view, resourceName, options) {
  if (!window.history) return;
  const nextState = buildNavigationState(view, resourceName);
  const currentState = window.history.state;
  const nextURL = buildNavigationURL(view, resourceName, Boolean(options?.includeListState));
  const currentURL = currentPagePath() + (window.location.search || '');
  if (sameNavigationState(currentState, nextState) && currentURL === nextURL) return;
  if (mode === 'push' && typeof window.history.pushState === 'function') {
    window.history.pushState(nextState, '', nextURL);
    return;
  }
  if (typeof window.history.replaceState === 'function') {
    window.history.replaceState(nextState, '', nextURL);
  }
}

async function restoreNavigationState(navState) {
  navState = navigationStateFromURL() || navState;
  if (!state.resources.length) {
    if (navState?.view === 'resource') {
      showNotFound({ history: 'none' });
      return true;
    }
    return false;
  }
  try {
    if (navState?.view === 'dashboard') {
      showDashboard({ history: 'none' });
      return true;
    }
    if (!navState?.resourceName) return false;
    const resource = state.resources.find((item) => item.name === navState.resourceName);
    if (!resource) {
      showNotFound({ history: 'none' });
      return true;
    }
    await selectResource(resource, { history: 'none', restoreQuery: true });
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
  if (els.darkModeIconMoon) els.darkModeIconMoon.removeAttribute('hidden');
  if (els.darkModeIconSun) els.darkModeIconSun.removeAttribute('hidden');
  if (els.darkModeToggle) {
    els.darkModeToggle.classList.toggle('is-dark', dark);
    els.darkModeToggle.setAttribute('aria-pressed', String(dark));
    els.darkModeToggle.setAttribute('aria-label', dark ? 'Switch to light mode' : 'Switch to dark mode');
  }
}

function persistThemePreference(dark) {
  try {
    if (dark) {
      localStorage.setItem(themeStorageKey, 'dark');
    } else {
      localStorage.removeItem(themeStorageKey);
    }
  } catch (_) {
    // localStorage may be unavailable in some contexts
  }
}

function animateThemeChangeFromControl(control, update) {
  const root = document.documentElement;
  const reducedMotion = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  if (!document.startViewTransition || reducedMotion || !control) {
    update();
    return;
  }
  const rect = control.getBoundingClientRect();
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;
  const endRadius = Math.hypot(
    Math.max(x, window.innerWidth - x),
    Math.max(y, window.innerHeight - y)
  );
  root.style.setProperty('--admin-theme-transition-x', x + 'px');
  root.style.setProperty('--admin-theme-transition-y', y + 'px');
  root.style.setProperty('--admin-theme-transition-radius', endRadius + 'px');
  root.classList.add('theme-view-transition');
  const transition = document.startViewTransition(update);
  transition.finished.finally(() => {
    root.classList.remove('theme-view-transition');
    root.style.removeProperty('--admin-theme-transition-x');
    root.style.removeProperty('--admin-theme-transition-y');
    root.style.removeProperty('--admin-theme-transition-radius');
  });
}

function toggleDarkMode(event) {
  const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
  const next = !isDark;
  animateThemeChangeFromControl(event?.currentTarget || els.darkModeToggle, () => {
    applyTheme(next);
    persistThemePreference(next);
  });
}

function themeByValue(value) {
  return adminThemes.find((theme) => theme.value === value) || adminThemes.find((theme) => theme.value === defaultAdminTheme) || adminThemes[0];
}

function applyAdminTheme(value) {
  const theme = themeByValue(value);
  if (!theme) return;
  document.documentElement.setAttribute('data-admin-theme', theme.value);
  if (els.adminThemeLabel) els.adminThemeLabel.textContent = theme.label;
  if (els.adminThemeToggle) {
    els.adminThemeToggle.setAttribute('aria-label', 'Select theme. Current theme: ' + theme.label);
  }
  if (els.adminThemeMenu) {
    Array.from(els.adminThemeMenu.querySelectorAll('[data-admin-theme-option]')).forEach((button) => {
      const selected = button.dataset.adminThemeOption === theme.value;
      button.classList.toggle('active', selected);
      button.setAttribute('aria-checked', String(selected));
    });
  }
}

function persistAdminTheme(value) {
  try {
    localStorage.setItem(adminThemeStorageKey, value);
  } catch (_) {
    // localStorage may be unavailable in some contexts
  }
}

function restoreAdminTheme() {
  let saved = '';
  try {
    saved = localStorage.getItem(adminThemeStorageKey) || '';
  } catch (_) {
    // ignore
  }
  applyAdminTheme(saved || defaultAdminTheme);
}

function closeAdminThemeMenu() {
  if (!els.adminThemeMenu || !els.adminThemeToggle) return;
  els.adminThemeMenu.hidden = true;
  els.adminThemeToggle.setAttribute('aria-expanded', 'false');
}

function renderAdminThemeMenu() {
  if (!els.adminThemeMenu) return;
  els.adminThemeMenu.innerHTML = '';
  const title = document.createElement('div');
  title.className = 'admin-theme-menu-title';
  title.textContent = 'Themes';
  els.adminThemeMenu.appendChild(title);
  adminThemes.forEach((theme) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'admin-theme-option';
    button.dataset.adminThemeOption = theme.value;
    button.setAttribute('role', 'menuitemradio');
    button.setAttribute('aria-checked', 'false');
    const swatch = document.createElement('span');
    swatch.className = 'admin-theme-swatch';
    swatch.setAttribute('aria-hidden', 'true');
    const label = document.createElement('span');
    label.className = 'admin-theme-option-label';
    label.textContent = theme.label;
    const check = document.createElement('span');
    check.className = 'admin-theme-option-check';
    check.setAttribute('aria-hidden', 'true');
    check.innerHTML = '<svg viewBox="0 0 16 16" focusable="false"><path d="M3.5 8.5 6.5 11.5 12.5 4.5"/></svg>';
    button.appendChild(swatch);
    button.appendChild(label);
    button.appendChild(check);
    button.addEventListener('click', (event) => {
      event.stopPropagation();
      applyAdminTheme(theme.value);
      persistAdminTheme(theme.value);
      closeAdminThemeMenu();
    });
    els.adminThemeMenu.appendChild(button);
  });
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
  if (adminDefaultTheme === 'dark') {
    applyTheme(true);
    return true;
  }
  if (adminDefaultTheme === 'system' && window.matchMedia) {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    applyTheme(prefersDark);
    return prefersDark;
  }
  applyTheme(false);
  return false;
}

let globalSearchTimer = null;
let globalSearchActiveIndex = -1;
const globalSearchDialog = document.getElementById('topbarSearchExpand');
const globalSearchDialogMount = globalSearchDialog ? globalSearchDialog.parentNode : null;
const globalSearchDialogNextSibling = globalSearchDialog ? globalSearchDialog.nextSibling : null;

function closeGlobalSearch() {
  globalSearchActiveIndex = -1;
  const expandEl = document.getElementById('topbarSearchExpand');
  if (expandEl) expandEl.classList.remove('has-results');
  if (els.topbarSearchResults) {
    els.topbarSearchResults.classList.remove('has-results');
    els.topbarSearchResults.innerHTML = '';
    els.topbarSearchResults.removeAttribute('aria-activedescendant');
  }
}

function openGlobalSearchDialog() {
  const expandEl = globalSearchDialog || document.getElementById('topbarSearchExpand');
  if (!expandEl) return;
  if (expandEl.parentNode !== document.body) {
    document.body.appendChild(expandEl);
  }
  expandEl.classList.add('open');
  expandEl.setAttribute('aria-modal', 'true');
  document.body.classList.add('search-dialog-open');
  window.requestAnimationFrame(() => {
    if (els.topbarSearchInput) els.topbarSearchInput.focus();
  });
}

function closeGlobalSearchDialog(options = {}) {
  clearTimeout(globalSearchTimer);
  globalSearchTimer = null;
  const expandEl = globalSearchDialog || document.getElementById('topbarSearchExpand');
  if (options.clearInput && els.topbarSearchInput) {
    els.topbarSearchInput.value = '';
  }
  if (expandEl) {
    expandEl.classList.remove('open');
    expandEl.setAttribute('aria-modal', 'false');
  }
  document.body.classList.remove('search-dialog-open');
  closeGlobalSearch();
  if (expandEl && globalSearchDialogMount && expandEl.parentNode !== globalSearchDialogMount) {
    globalSearchDialogMount.insertBefore(expandEl, globalSearchDialogNextSibling);
  }
}

function globalSearchResultItems() {
  return els.topbarSearchResults ? Array.from(els.topbarSearchResults.querySelectorAll('.search-result-item')) : [];
}

function setGlobalSearchActiveIndex(index) {
  const items = globalSearchResultItems();
  if (!items.length) {
    globalSearchActiveIndex = -1;
    if (els.topbarSearchResults) els.topbarSearchResults.removeAttribute('aria-activedescendant');
    return;
  }
  globalSearchActiveIndex = ((index % items.length) + items.length) % items.length;
  items.forEach((item, itemIndex) => {
    const active = itemIndex === globalSearchActiveIndex;
    item.classList.toggle('active', active);
    item.setAttribute('aria-selected', String(active));
    if (active) {
      if (!item.id) item.id = 'global-search-result-' + itemIndex;
      if (els.topbarSearchResults) els.topbarSearchResults.setAttribute('aria-activedescendant', item.id);
      item.scrollIntoView({ block: 'nearest' });
    }
  });
}

function moveGlobalSearchSelection(delta) {
  const items = globalSearchResultItems();
  if (!items.length) return false;
  setGlobalSearchActiveIndex(globalSearchActiveIndex + delta);
  return true;
}

function openActiveGlobalSearchResult() {
  const items = globalSearchResultItems();
  if (globalSearchActiveIndex < 0 || !items[globalSearchActiveIndex]) return false;
  items[globalSearchActiveIndex].click();
  return true;
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

async function globalSearchFallback(q) {
  return Promise.all(
    state.resources.map(async (resource) => {
      try {
        const basePath = apiBase + '/resources' + resource.path;
        const data = await request(basePath + '?page=1&size=5&search=' + encodeURIComponent(q));
        const items = data.items || data.results || data.data || [];
        return {
          resource,
          items: items.map((item) => ({
            id: recordPrimaryKey(item),
            label: recordDisplayLabel(item, []),
            item: item
          }))
        };
      } catch (_) {
        return { resource, items: [] };
      }
    })
  );
}

function normalizeGlobalSearchText(value) {
  return String(value || '')
    .toLowerCase()
    .replace(/[_/.-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function compactGlobalSearchText(value) {
  return normalizeGlobalSearchText(value).replace(/\s+/g, '');
}

function globalSearchNeedles(q) {
  const normalized = normalizeGlobalSearchText(q);
  const compact = compactGlobalSearchText(q);
  const needles = [normalized, compact].filter(Boolean);
  if (compact.endsWith('s') && compact.length > 3) {
    needles.push(compact.slice(0, -1));
  }
  if (compact.endsWith('ies') && compact.length > 4) {
    needles.push(compact.slice(0, -3) + 'y');
  }
  return Array.from(new Set(needles));
}

function scoreGlobalSearchField(value, needles, weight) {
  const normalized = normalizeGlobalSearchText(value);
  const compact = compactGlobalSearchText(value);
  if (!normalized && !compact) return 0;
  return needles.reduce((best, needle) => {
    if (normalized === needle || compact === needle) return Math.max(best, weight + 30);
    if (normalized.startsWith(needle) || compact.startsWith(needle)) return Math.max(best, weight + 18);
    if (normalized.includes(needle) || compact.includes(needle)) return Math.max(best, weight);
    return best;
  }, 0);
}

function globalResourceSearchScore(resource, needles) {
  return Math.max(
    scoreGlobalSearchField(resource?.label, needles, 90),
    scoreGlobalSearchField(resource?.name, needles, 85),
    scoreGlobalSearchField(resource?.path, needles, 72),
    scoreGlobalSearchField(resource?.group, needles, 42),
    scoreGlobalSearchField(resource?.description, needles, 16)
  );
}

function globalResourceSearch(q) {
  const needles = globalSearchNeedles(q);
  if (!needles.length) return [];
  return state.resources
    .map((resource, index) => ({ resource, index, score: globalResourceSearchScore(resource, needles) }))
    .filter((match) => match.score > 0)
    .sort((a, b) => (b.score - a.score) || (a.index - b.index))
    .map((match) => match.resource);
}

async function fetchGlobalSearchResults(q) {
  try {
    const payload = await request(apiBase + '/search?q=' + encodeURIComponent(q) + '&size=5');
    return (payload.results || []).map((group) => ({
      resource: state.resources.find((item) => item.name === group.resource?.name) || group.resource,
      items: group.items || []
    }));
  } catch (error) {
    console.error('global search aggregate failed:', error);
    return globalSearchFallback(q);
  }
}

function renderGlobalResourceResults(resources, q) {
  if (!resources.length || !els.topbarSearchResults) return;
  const group = document.createElement('div');
  group.className = 'search-results-group';
  const label = document.createElement('div');
  label.className = 'search-results-group-label';
  label.textContent = 'Resources';
  group.appendChild(label);

  resources.forEach((resource) => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'search-result-item search-result-resource';
    btn.setAttribute('role', 'option');
    btn.setAttribute('aria-selected', 'false');

    const icon = document.createElement('span');
    icon.className = 'search-result-icon';
    icon.setAttribute('aria-hidden', 'true');
    icon.innerHTML = resourceIcon(resource);

    const main = document.createElement('span');
    main.className = 'search-result-main';
    const summary = document.createElement('span');
    summary.className = 'search-result-summary';
    summary.innerHTML = highlightMatch(resource.label || resource.name, q);
    const meta = document.createElement('span');
    meta.className = 'search-result-resource-meta';
    const metaParts = [resource.group, resource.name || resource.path].filter(Boolean);
    meta.textContent = metaParts.join(' / ');

    main.appendChild(summary);
    if (meta.textContent) main.appendChild(meta);
    btn.appendChild(icon);
    btn.appendChild(main);
    btn.addEventListener('click', async () => {
      closeGlobalSearchDialog({ clearInput: true });
      await selectResource(resource, { restoreQuery: false });
    });
    group.appendChild(btn);
  });

  els.topbarSearchResults.appendChild(group);
}

async function globalSearch(query) {
  closeGlobalSearch();
  if (!els.topbarSearchResults) return;
  const q = query.trim();
  if (!q || q.length < 2) return;
  if (!state.resources.length) return;

  const resourceMatches = globalResourceSearch(q);
  const results = await fetchGlobalSearchResults(q);
  if (els.topbarSearchInput && els.topbarSearchInput.value.trim() !== q) return;

  const noResults = resourceMatches.length === 0 && results.every((r) => r.items.length === 0);
  if (noResults) {
    els.topbarSearchResults.innerHTML = '<div class="search-results-empty">No results for &ldquo;' + escapeHTML(q) + '&rdquo;</div>';
    els.topbarSearchResults.classList.add('has-results');
    const expandEl = document.getElementById('topbarSearchExpand');
    if (expandEl) expandEl.classList.add('has-results');
    return;
  }

  renderGlobalResourceResults(resourceMatches, q);

  results.forEach(({ resource, items }) => {
    if (!items.length) return;
    const group = document.createElement('div');
    group.className = 'search-results-group';
    const label = document.createElement('div');
    label.className = 'search-results-group-label';
    label.textContent = 'Records / ' + (resource.label || resource.name);
    group.appendChild(label);

    items.forEach((result) => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'search-result-item';
      btn.setAttribute('role', 'option');
      btn.setAttribute('aria-selected', 'false');
      const record = result.item || {};
      const pk = result.id ?? recordPrimaryKey(record);
      const displayLabel = result.label || recordDisplayLabel(record, []);
      const summary = document.createElement('span');
      summary.className = 'search-result-summary';
      summary.innerHTML = highlightMatch(displayLabel || String(pk ?? ''), q);
      const idSpan = document.createElement('span');
      idSpan.className = 'search-result-id';
      idSpan.textContent = '#' + String(pk ?? '');
      btn.appendChild(summary);
      if (displayLabel) btn.appendChild(idSpan);
      btn.addEventListener('click', async () => {
        closeGlobalSearchDialog({ clearInput: true });
        await selectResource(resource, { restoreQuery: false });
        if (pk != null && pk !== '') {
          state.selected = await request(currentBasePath() + '/' + encodeURIComponent(String(pk)));
          renderSelectedRecord();
          await renderUpdateForm();
          openModal(els.recordModal);
        }
      });
      group.appendChild(btn);
    });
    els.topbarSearchResults.appendChild(group);
  });

  if (els.topbarSearchResults.children.length) {
    els.topbarSearchResults.classList.add('has-results');
    const expandEl = document.getElementById('topbarSearchExpand');
    if (expandEl) expandEl.classList.add('has-results');
    setGlobalSearchActiveIndex(0);
  }
}

function isStandaloneLoginPage() {
  return currentPagePath() === adminLoginPath;
}

function isStandaloneAdminPage() {
  return !isStandaloneLoginPage() && pathWithinAdmin(currentPagePath());
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
  document.body.classList.toggle('standalone-admin-page', !isStandaloneLoginPage());
  document.body.classList.toggle('login-page', isStandaloneLoginPage());
  if (isStandaloneLoginPage()) {
    document.title = adminTitle + ' Login';
    els.shellEyebrow.textContent = 'Admin Login';
    els.pageTitle.textContent = adminTitle + ' Login';
    els.pageIntro.textContent = 'Sign in to enter ' + adminBrandName + ' admin.';
    return;
  }
  document.title = adminTitle;
  els.shellEyebrow.textContent = 'Admin Console';
  els.pageTitle.textContent = adminTitle;
  els.pageIntro.textContent = 'An operations workspace for ' + adminBrandName + '.';
}

function setTopbarBreadcrumb(root, current) {
  if (els.topbarBreadcrumbRoot) els.topbarBreadcrumbRoot.textContent = root || 'Dashboard';
  if (els.topbarBreadcrumbCurrent) els.topbarBreadcrumbCurrent.textContent = current || 'Overview';
}

function redirectToLogin(message) {
  rememberFlashMessage(message);
  if (!isStandaloneLoginPage()) {
    window.location.replace(adminLoginPath);
  }
}

function redirectToAdmin(message) {
  rememberFlashMessage(message);
  if (!isStandaloneAdminPage()) {
    window.location.replace(adminPagePath);
  }
}

function hasToken() {
  return !!els.token.value.trim();
}

function tokenStorage() {
  return tokenStorageDriver === 'session' ? sessionStorage : localStorage;
}

function clearPersistedAuthIdentity() {
  localStorage.removeItem(authIdentityStorageKey);
  sessionStorage.removeItem(authIdentityStorageKey);
}

function emptyAuthIdentity() {
  return { name: '', userID: null, email: '', isAdmin: false, roles: [], expiresAt: '', issuedAt: '', issuer: '' };
}

function normalizeAuthIdentity(input = {}) {
  const roleSource = Array.isArray(input.roles) ? input.roles : (Array.isArray(input.permissions) ? input.permissions : []);
  const roles = roleSource.map((role) => {
    if (typeof role === 'string') return { name: role, code: role };
    if (!role || typeof role !== 'object') return null;
    return {
      id: role.id ?? role.ID ?? null,
      name: String(role.name || role.Name || role.code || role.Code || '').trim(),
      code: String(role.code || role.Code || '').trim(),
      status: role.status ?? role.Status ?? null
    };
  }).filter(Boolean);
  return {
    name: String(input.name || input.username || input.user_name || '').trim(),
    userID: input.user_id ?? input.userID ?? input.id ?? null,
    email: String(input.email || '').trim(),
    isAdmin: Boolean(input.is_admin ?? input.isAdmin ?? input.admin ?? false),
    roles,
    expiresAt: String(input.expires_at || input.expiresAt || '').trim(),
    issuedAt: String(input.issued_at || input.issuedAt || '').trim(),
    issuer: String(input.issuer || '').trim()
  };
}

function persistAuthIdentity() {
  const auth = normalizeAuthIdentity(state.auth);
  if (!auth.name && (auth.userID == null || auth.userID === '') && !auth.email) {
    clearPersistedAuthIdentity();
    return;
  }
  tokenStorage().setItem(authIdentityStorageKey, JSON.stringify(auth));
}

function restoreAuthIdentity() {
  const saved = tokenStorage().getItem(authIdentityStorageKey);
  if (!saved) return false;
  try {
    const parsed = JSON.parse(saved);
    if (!parsed || typeof parsed !== 'object') return false;
    state.auth = normalizeAuthIdentity(parsed);
    return !!state.auth.name || state.auth.userID != null || !!state.auth.email;
  } catch (error) {
    clearPersistedAuthIdentity();
    return false;
  }
}

function persistToken() {
  const token = els.token.value.trim();
  const storage = tokenStorage();
  if (token) {
    storage.setItem(tokenStorageKey, token);
  } else {
    localStorage.removeItem(tokenStorageKey);
    sessionStorage.removeItem(tokenStorageKey);
    clearPersistedAuthIdentity();
  }
}

function restoreToken() {
  const saved = tokenStorage().getItem(tokenStorageKey);
  if (saved) {
    els.token.value = saved;
    restoreAuthIdentity();
    return true;
  }
  return false;
}

function formatDateTime(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
}

function tokenExpirySummary(value) {
  if (!value) return 'Token expiry unavailable';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Token expiry unavailable';
  const diff = date.getTime() - Date.now();
  if (diff <= 0) return 'Token expired';
  const minutes = Math.round(diff / 60000);
  if (minutes < 60) return 'Token expires in ' + minutes + ' min';
  const hours = Math.round(minutes / 60);
  if (hours < 48) return 'Token expires in ' + hours + ' hr';
  return 'Token expires ' + formatDateTime(value);
}

 function renderSignedOutState() {
   closeAllModals();
   setLoginFeedback('');
   els.loginForm.hidden = false;
   els.sessionShell.hidden = false;
  els.sessionActions.hidden = true;
  els.manualTokenTools.hidden = true;
  els.adminShell.hidden = true;
  if (els.sidebarAuthStatus) els.sidebarAuthStatus.textContent = 'Signed out';
  if (els.sidebarAuthMeta) els.sidebarAuthMeta.textContent = 'No active session';
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
  const name = state.auth.name || state.auth.email || 'Admin';
  const initials = name.split(/\s+/).map(w => w[0] || '').slice(0, 2).join('').toUpperCase() || '?';
  const sidebarAvatar = document.querySelector('.sidebar-user-avatar');
  const sidebarName = document.querySelector('.sidebar-user-copy strong');
  const sidebarMenuAvatar = document.getElementById('sidebarMenuAvatar');
  const sidebarMenuName = document.getElementById('sidebarMenuName');
  const sidebarMenuMeta = document.getElementById('sidebarMenuMeta');
  if (sidebarAvatar) sidebarAvatar.textContent = initials;
  if (sidebarName) sidebarName.textContent = name;
  if (sidebarMenuAvatar) sidebarMenuAvatar.textContent = initials;
  if (sidebarMenuName) sidebarMenuName.textContent = name;
  if (els.sidebarAuthStatus) els.sidebarAuthStatus.textContent = state.auth.isAdmin ? 'Admin session' : 'Authenticated';
  if (els.sidebarAuthMeta) {
    const meta = state.auth.email || (state.auth.userID != null ? ('User #' + state.auth.userID) : tokenExpirySummary(state.auth.expiresAt));
    els.sidebarAuthMeta.textContent = meta;
    if (sidebarMenuMeta) sidebarMenuMeta.textContent = meta;
  }
  const topbarAvatar = document.getElementById('topbarUserAvatar');
  const topbarName = document.getElementById('topbarUserName');
  if (topbarAvatar) topbarAvatar.textContent = initials;
  if (topbarName) topbarName.textContent = name;
  if (els.topbarProfileName) els.topbarProfileName.textContent = name;
  if (els.topbarProfileMeta) {
    els.topbarProfileMeta.textContent = state.auth.email || 'No email available';
  }
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
    if (isStandaloneAdminPage()) {
      redirectToLogin('Ready. Sign in to continue.');
      return;
    }
    renderSignedOutState();
  }
}

function applyAuthProfile(payload) {
  const source = payload && typeof payload === 'object' && payload.user && typeof payload.user === 'object' ? { ...payload.user, ...payload } : (payload || {});
  const profile = normalizeAuthIdentity(source);
  state.auth = {
    ...state.auth,
    ...profile,
    name: profile.name || state.auth.name,
    userID: profile.userID ?? state.auth.userID
  };
  persistAuthIdentity();
  renderSignedInState();
}

async function refreshAuthProfile(options = {}) {
  if (!hasToken() || !authMePath) return false;
  const response = await fetch(authMePath, { method: 'GET', headers: requestHeaders() });
  const text = await response.text();
  let payload = null;
  try { payload = text ? JSON.parse(text) : null; } catch (_) { payload = text; }
  if (response.status === 401) {
    logout('Your session has expired. Please sign in again.');
    throw new Error('Your session has expired. Please sign in again.');
  }
  if (response.status === 403) {
    const message = extractErrorMessage(payload) || 'Your account is authenticated but does not have permission to open this admin area.';
    if (!isStandaloneLoginPage()) {
      showErrorPage({ status: 403 });
    }
    setStatus(message, 'danger');
    return false;
  }
  if (!response.ok) {
    if (!isStandaloneLoginPage()) {
      showErrorPage({ status: response.status || 500 });
    }
    if (options.required) {
      throw createHTTPError(response.status || 500, extractErrorMessage(payload) || 'Unable to load the current user profile.');
    }
    setStatus(extractErrorMessage(payload) || response.statusText || 'Unable to load the current user profile.', 'danger');
    return false;
  }
  applyAuthProfile(payload);
  return true;
}

function resetAdminState() {
  state.auth = emptyAuthIdentity();
  state.current = null;
  state.meta = null;
  state.errorPage = null;
  setTopbarBreadcrumb('Dashboard', 'Overview');
  state.resources = [];
  state.resourceSearch = '';
  state.records = [];
  state.selected = null;
  state.bulkSelected = {};
  state.visibleColumns = {};
  state.listUpdatedAt = null;
  updateColumnToggleLabel();
  state.relationSearch = {};
  state.relationTimers = {};
  state.forms = {
    create: { dirty: false, initial: '', pending: false },
    update: { dirty: false, initial: '', pending: false }
  };
  state.pagination = { page: 1, size: Number(els.pageSize.value || 10), pages: 1, total: 0 };
   if (els.sidebarResourceSearch) {
     els.sidebarResourceSearch.value = '';
   }
    renderResources();
   els.resourceTitle.textContent = 'Select a resource';
   els.resourcePath.textContent = 'Sign in to open a resource workspace.';
   els.resourcePath.hidden = false;
   els.detailTitle.textContent = 'No record selected';
   els.detailObjectBadge.textContent = 'Draft view';
   els.detailFields.innerHTML = '<p class="muted">No record selected.</p>';
  els.detail.textContent = 'No record selected.';
  els.createForm.innerHTML = '<p class="muted">Sign in to create records.</p>';
  els.updateForm.innerHTML = '<p class="muted">Sign in to edit records.</p>';
  els.filtersForm.innerHTML = '';
  els.sort.innerHTML = '';
  els.search.placeholder = 'Search current resource';
  els.list.innerHTML = '<div class="empty-state">Sign in to browse records in the admin workspace.</div>';
  if (els.workspaceHeader) els.workspaceHeader.hidden = false;
  if (els.recordsShell) els.recordsShell.hidden = false;
  if (els.dashboardShell) els.dashboardShell.hidden = true;
  if (els.errorShell) els.errorShell.hidden = true;
  renderResourceActionSummary();
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
      const message = 'Your session has expired or the token is invalid. Please sign in again.';
      logout(message);
      throw createHTTPError(response.status, message);
    }
    if (response.status === 403) {
      throw createHTTPError(response.status, extractErrorMessage(data) || 'You are signed in, but your account does not have permission for this action.');
    }
    throw createHTTPError(response.status, extractErrorMessage(data) || response.statusText || ('Request failed with status ' + response.status + '.'));
  }
  return data;
}

function currentBasePath() {
  return apiBase + '/resources' + state.current.path;
}

function hasAction(action) {
  return (state.meta?.actions || []).includes(action);
}

function actionLabel(action) {
  return ({
    list: 'List',
    detail: 'Detail',
    create: 'Create',
    update: 'Update',
    delete: 'Delete',
    bulk_delete: 'Bulk delete'
  })[action] || String(action || '').replace(/_/g, ' ');
}

function actionTone(action) {
  if (action === 'delete' || action === 'bulk_delete') return 'danger';
  if (action === 'create' || action === 'update') return 'write';
  return '';
}

function renderResourceActionSummary() {
  if (!els.resourceActionSummary) return;
  const actions = Array.isArray(state.meta?.actions) ? state.meta.actions : [];
  els.resourceActionSummary.innerHTML = '';
  els.resourceActionSummary.hidden = !state.current || actions.length === 0;
  if (els.resourceActionSummary.hidden) return;
  const label = document.createElement('span');
  label.className = 'resource-action-label';
  label.textContent = 'Allowed';
  els.resourceActionSummary.appendChild(label);
  actions.forEach((action) => {
    const pill = document.createElement('span');
    const tone = actionTone(action);
    pill.className = 'resource-action-pill' + (tone ? ' ' + tone : '');
    pill.textContent = actionLabel(action);
    els.resourceActionSummary.appendChild(pill);
  });
  if (!actions.some((action) => action === 'create' || action === 'update' || action === 'delete' || action === 'bulk_delete')) {
    const pill = document.createElement('span');
    pill.className = 'resource-action-pill readonly';
    pill.textContent = 'Read only';
    els.resourceActionSummary.appendChild(pill);
  }
}

function recordPrimaryKey(record) {
  return record?.id;
}

function fieldMeta(name) {
  return (state.meta?.fields || []).find((field) => field.name === name);
}

function fieldLabel(name) {
  return fieldMeta(name)?.label || name;
}

function fieldValue(name) {
  return els.filtersForm.elements.namedItem(name);
}

function isDateTimeField(field) {
  if (!field) return false;
  const component = String(field.component || '').toLowerCase();
  const type = String(field.type || '').toLowerCase();
  const name = String(field.name || '').toLowerCase();
  return component === 'datetime' ||
    component === 'date' ||
    type === 'time' ||
    type === 'datetime' ||
    type === 'date' ||
    name === 'createdat' ||
    name === 'updatedat' ||
    name === 'created_at' ||
    name === 'updated_at';
}

function filterFieldInputs(name) {
  const field = fieldMeta(name);
  if (isDateTimeField(field)) {
    return {
      field,
      from: fieldValue(name + '__from'),
      to: fieldValue(name + '__to')
    };
  }
  return {
    field,
    value: fieldValue(name)
  };
}

function dateFilterBoundaryValue(value, edge) {
  const raw = String(value || '').trim();
  if (!raw) return '';
  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
    return raw + (edge === 'to' ? ' 23:59:59' : ' 00:00:00');
  }
  return raw;
}

function setDateInputValue(input, value) {
  if (!input) return;
  const raw = String(value || '').trim();
  input.value = raw.match(/^\d{4}-\d{2}-\d{2}/)?.[0] || '';
}

function getFilterStateValue(name) {
  const inputs = filterFieldInputs(name);
  if (isDateTimeField(inputs.field)) {
    const from = String(inputs.from?.value || '').trim();
    const to = String(inputs.to?.value || '').trim();
    if (!from && !to) return null;
    return { from, to };
  }
  const value = String(inputs.value?.value || '').trim();
  return value === '' ? null : value;
}

function setFilterStateValue(name, value) {
  const inputs = filterFieldInputs(name);
  if (isDateTimeField(inputs.field)) {
    if (value && typeof value === 'object') {
      setDateInputValue(inputs.from, value.from);
      setDateInputValue(inputs.to, value.to);
    } else {
      setDateInputValue(inputs.from, '');
      setDateInputValue(inputs.to, '');
    }
    syncFilterFacetState(inputs.from || inputs.to);
    return;
  }
  if (inputs.value) inputs.value.value = value == null ? '' : String(value);
  syncFilterFacetState(inputs.value);
}

function clearListFilters(options = {}) {
  if (options.includeSearch !== false) {
    els.search.value = '';
  }
  if (options.includeSort !== false) {
    els.sort.value = '';
  }
  (state.meta?.filter_fields || []).forEach((name) => {
    setFilterStateValue(name, '');
  });
}

function appendFilterParams(params, options = {}) {
  const prefix = options.prefix || '';
  const boundary = options.boundary || false;
  (state.meta?.filter_fields || []).forEach((name) => {
    const value = getFilterStateValue(name);
    if (value == null) return;
    const field = fieldMeta(name);
    if (isDateTimeField(field) && typeof value === 'object') {
      const from = dateFilterBoundaryValue(value.from, 'from');
      const to = dateFilterBoundaryValue(value.to, 'to');
      if (from) params.set(prefix + name + '__from', boundary ? from : value.from);
      if (to) params.set(prefix + name + '__to', boundary ? to : value.to);
      return;
    }
    params.set(prefix + name, value);
  });
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

function isDateLikeValue(field, value) {
  if (value == null || value === '') return false;
  if (field?.component === 'datetime') return true;
  if (typeof value !== 'string') return false;
  return /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(value);
}

function formatDateTimeParts(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return { primary: String(value), secondary: '' };
  }
  return {
    primary: new Intl.DateTimeFormat(adminLocale || undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(date),
    secondary: new Intl.DateTimeFormat(adminLocale || undefined, { hour: 'numeric', minute: '2-digit' }).format(date),
  };
}

function normalizedFieldFormat(field) {
  return String(field?.format || '').trim().toLowerCase();
}

function titleCaseValue(value) {
  return String(value).replace(/\w\S*/g, (word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase());
}

function numberValue(value) {
  const number = typeof value === 'number' ? value : Number(String(value).replace(/,/g, ''));
  return Number.isFinite(number) ? number : null;
}

function formatNumberValue(value, options) {
  const number = numberValue(value);
  if (number == null) return String(value);
  return new Intl.NumberFormat(adminLocale || undefined, options).format(number);
}

function formatRelativeTime(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  const seconds = Math.round((date.getTime() - Date.now()) / 1000);
  const units = [
    ['year', 60 * 60 * 24 * 365],
    ['month', 60 * 60 * 24 * 30],
    ['day', 60 * 60 * 24],
    ['hour', 60 * 60],
    ['minute', 60],
    ['second', 1]
  ];
  const [unit, size] = units.find(([, unitSeconds]) => Math.abs(seconds) >= unitSeconds) || units[units.length - 1];
  return new Intl.RelativeTimeFormat(adminLocale || undefined, { numeric: 'auto' }).format(Math.round(seconds / size), unit);
}

function currencyCodeFromFormat(format) {
  const match = format.match(/^currency(?::|-)?([a-z]{3})?$/i);
  return match && match[1] ? match[1].toUpperCase() : '';
}

function formatFieldDisplay(field, fieldName, value) {
  const format = normalizedFieldFormat(field);
  const display = {
    primary: formatValue(value),
    secondary: '',
    mono: fieldName === 'id' || fieldName.endsWith('_id') || fieldName.endsWith('Id'),
    numeric: field?.component === 'number' || field?.type === 'number' || field?.type === 'integer',
    applied: false
  };
  if (!format) return display;
  if (format === 'title') {
    display.primary = titleCaseValue(value);
    display.applied = true;
    return display;
  }
  if (format === 'uppercase') {
    display.primary = String(value).toUpperCase();
    display.applied = true;
    return display;
  }
  if (format === 'lowercase') {
    display.primary = String(value).toLowerCase();
    display.applied = true;
    return display;
  }
  if (format === 'mono' || format === 'code') {
    display.mono = true;
    display.applied = true;
    return display;
  }
  if (format === 'number' || format === 'decimal') {
    display.primary = formatNumberValue(value, { maximumFractionDigits: 2 });
    display.numeric = true;
    display.applied = true;
    return display;
  }
  if (format === 'integer') {
    display.primary = formatNumberValue(value, { maximumFractionDigits: 0 });
    display.numeric = true;
    display.applied = true;
    return display;
  }
  if (format === 'percent') {
    const number = numberValue(value);
    display.primary = number == null
      ? String(value)
      : (Math.abs(number) <= 1
        ? new Intl.NumberFormat(adminLocale || undefined, { style: 'percent', maximumFractionDigits: 2 }).format(number)
        : formatNumberValue(number, { maximumFractionDigits: 2 }) + '%');
    display.numeric = true;
    display.applied = true;
    return display;
  }
  if (format.startsWith('currency')) {
    const currency = currencyCodeFromFormat(format);
    display.primary = currency
      ? formatNumberValue(value, { style: 'currency', currency })
      : formatNumberValue(value, { maximumFractionDigits: 2 });
    display.numeric = true;
    display.applied = true;
    return display;
  }
  if (format === 'date') {
    const date = new Date(value);
    display.primary = Number.isNaN(date.getTime())
      ? String(value)
      : new Intl.DateTimeFormat(adminLocale || undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(date);
    display.applied = true;
    return display;
  }
  if (format === 'datetime' || format === 'time') {
    const formattedDate = formatDateTimeParts(value);
    display.primary = formattedDate.primary;
    display.secondary = formattedDate.secondary;
    display.applied = true;
    return display;
  }
  if (format === 'relative') {
    display.primary = formatRelativeTime(value);
    display.secondary = formatDateTimeParts(value).primary;
    display.applied = true;
    return display;
  }
  return display;
}

function fieldControlID(scopeKey, field) {
  return 'field-' + scopeKey + '-' + field.name;
}

function shouldSpanFullWidth(field) {
  if (!field) return false;
  return field.width === 'full' || field.component === 'textarea' || field.component === 'text' || field.component === 'array' || isMultiRelationField(field);
}

function isFieldRequiredForForm(field, scopeKey) {
  if (!field?.required) return false;
  // Update forms allow blank passwords so users only re-enter one when changing it.
  return !(scopeKey === 'update' && field.component === 'password');
}

function applyFieldControlState(control, field, scopeKey) {
  if (!control || !field) return;
  const controlID = fieldControlID(scopeKey, field);
  const directControl = formControlTagNames.has(control.tagName)
    ? control
    : control.querySelector('input[name="' + field.name + '"], select[name="' + field.name + '"], textarea[name="' + field.name + '"]');
  if (directControl) {
    directControl.id = controlID;
    if (isFieldRequiredForForm(field, scopeKey) && !field.read_only && directControl.type !== 'checkbox') {
      directControl.required = true;
    }
    if (field.read_only) {
      directControl.disabled = true;
    }
  }
}

function formState(scopeKey) {
  if (!state.forms[scopeKey]) {
    state.forms[scopeKey] = { dirty: false, initial: '', pending: false };
  }
  return state.forms[scopeKey];
}

function formSnapshot(form) {
  const values = [];
  Array.from(form.elements || []).forEach((element) => {
    if (!element.name || element.disabled || element.type === 'submit' || element.type === 'button') return;
    if (element.type === 'checkbox') {
      values.push([element.name, element.checked ? '1' : '0']);
      return;
    }
    if (element.tagName === 'SELECT' && element.multiple) {
      values.push([element.name, Array.from(element.selectedOptions).map((option) => option.value).sort().join('\u0000')]);
      return;
    }
    values.push([element.name, element.value]);
  });
  return JSON.stringify(values.sort((a, b) => a[0].localeCompare(b[0]) || String(a[1]).localeCompare(String(b[1]))));
}

function updateFormStatus(form, scopeKey) {
  if (!form) return;
  const info = formState(scopeKey);
  form.dataset.dirty = info.dirty ? 'true' : 'false';
  const status = form.querySelector('[data-form-status="' + scopeKey + '"]');
  if (status) {
    status.textContent = info.dirty
      ? 'Unsaved changes'
      : (scopeKey === 'update' ? 'Review the changes before saving.' : 'Only required fields need values to create the record.');
  }
  const submit = form.querySelector('button[type="submit"]');
  if (submit) {
    submit.disabled = info.pending;
    submit.textContent = info.pending ? (submit.dataset.pendingText || 'Saving...') : (submit.dataset.defaultText || submit.textContent);
  }
}

function markFormClean(form, scopeKey) {
  if (!form) return;
  const info = formState(scopeKey);
  info.initial = formSnapshot(form);
  info.dirty = false;
  updateFormStatus(form, scopeKey);
}

function syncFormDirty(form, scopeKey) {
  if (!form) return;
  const info = formState(scopeKey);
  info.dirty = formSnapshot(form) !== info.initial;
  updateFormStatus(form, scopeKey);
}

function setFormPending(form, scopeKey, pending) {
  const info = formState(scopeKey);
  info.pending = pending;
  updateFormStatus(form, scopeKey);
}

function formFieldWrapper(form, name) {
  return Array.from(form.querySelectorAll('.form-field-card')).find((wrapper) => wrapper.dataset.fieldName === name);
}

function clearFormErrors(form) {
  form.querySelectorAll('.form-field-card.has-error').forEach((wrapper) => wrapper.classList.remove('has-error'));
  form.querySelectorAll('.field-error').forEach((errorEl) => {
    errorEl.textContent = '';
    errorEl.hidden = true;
  });
  form.querySelectorAll('[aria-invalid="true"]').forEach((control) => {
    control.removeAttribute('aria-invalid');
    control.removeAttribute('aria-describedby');
  });
}

function setFormFieldError(form, name, message) {
  const wrapper = formFieldWrapper(form, name);
  if (!wrapper) return false;
  wrapper.classList.add('has-error');
  const errorEl = wrapper.querySelector('.field-error');
  if (errorEl) {
    errorEl.textContent = message;
    errorEl.hidden = false;
  }
  const control = Array.from(wrapper.querySelectorAll('[name]')).find((item) => item.name === name);
  if (control) {
    control.setAttribute('aria-invalid', 'true');
    if (errorEl?.id) control.setAttribute('aria-describedby', errorEl.id);
  }
  return true;
}

function focusFormFieldWrapper(wrapper) {
  if (!wrapper) return false;
  wrapper.scrollIntoView({ block: 'center', behavior: 'smooth' });
  window.requestAnimationFrame(() => {
    const target = wrapper.querySelector('.multi-relation-dropdown summary, input:not([disabled]):not([type="hidden"]), select:not([disabled]):not([hidden]), textarea:not([disabled]), button:not([disabled])');
    if (target && typeof target.focus === 'function') {
      target.focus();
    }
  });
  return true;
}

function focusFirstFormError(form) {
  return focusFormFieldWrapper(form.querySelector('.form-field-card.has-error'));
}

function formFieldValue(form, field) {
  const control = form.elements.namedItem(field.name);
  if (!control) return '';
  if (typeof RadioNodeList !== 'undefined' && control instanceof RadioNodeList) {
    return control.value;
  }
  if (control.tagName === 'SELECT' && control.multiple) {
    return Array.from(control.selectedOptions).map((option) => option.value);
  }
  if (control.type === 'checkbox') {
    return control.checked;
  }
  return control.value;
}

function applyServerErrorToForm(form, error) {
  const message = extractErrorMessage(error);
  const match = message.match(/^field "([^"]+)"(?:: | is )(.*)$/);
  if (match && setFormFieldError(form, match[1], match[2] ? 'Field ' + match[2] : message)) {
    focusFirstFormError(form);
    return true;
  }
  return false;
}

function createFieldTag(text, tone) {
  const badge = document.createElement('span');
  badge.className = 'field-tag' + (tone ? ' ' + tone : '');
  badge.textContent = text;
  return badge;
}

function buildTableCellContent(fieldName, value) {
  const field = fieldMeta(fieldName);
  const wrap = document.createElement('div');
  wrap.className = 'table-cell';
  if (value == null || value === '') {
    const badge = document.createElement('span');
    badge.className = 'table-badge neutral';
    badge.textContent = 'Empty';
    wrap.appendChild(badge);
    return wrap;
  }
  if (typeof value === 'boolean') {
    const badge = document.createElement('span');
    badge.className = 'table-badge boolean ' + (value ? 'boolean-true' : 'boolean-false');
    const dot = document.createElement('span');
    dot.className = 'table-dot';
    badge.appendChild(dot);
    badge.appendChild(document.createTextNode(value ? 'Yes' : 'No'));
    wrap.appendChild(badge);
    return wrap;
  }
  if (Array.isArray(value)) {
    const badge = document.createElement('span');
    badge.className = 'table-badge info';
    badge.textContent = value.length + ' item' + (value.length === 1 ? '' : 's');
    wrap.appendChild(badge);
    if (value.length) {
      const hint = document.createElement('span');
      hint.className = 'table-cell-hint';
      hint.textContent = value.map((item) => formatValue(item)).join(', ');
      wrap.appendChild(hint);
    }
    return wrap;
  }
  const primary = document.createElement('span');
  primary.className = 'table-cell-value';
  const formatted = formatFieldDisplay(field, fieldName, value);
  if (formatted.applied) {
    primary.textContent = formatted.primary;
    wrap.appendChild(primary);
    if (formatted.secondary) {
      const secondary = document.createElement('span');
      secondary.className = 'table-cell-hint';
      secondary.textContent = formatted.secondary;
      wrap.appendChild(secondary);
    }
  } else if (isDateLikeValue(field, value)) {
    const formattedDate = formatDateTimeParts(value);
    primary.textContent = formattedDate.primary;
    const secondary = document.createElement('span');
    secondary.className = 'table-cell-hint';
    secondary.textContent = formattedDate.secondary;
    wrap.appendChild(primary);
    if (formattedDate.secondary) {
      wrap.appendChild(secondary);
    }
  } else {
    primary.textContent = formatValue(value);
    wrap.appendChild(primary);
  }
  if (formatted.mono) {
    primary.classList.add('mono');
  }
  if (formatted.numeric) {
    primary.classList.add('numeric');
  }
  if (field?.unique) {
    const badge = document.createElement('span');
    badge.className = 'table-badge neutral';
    badge.textContent = 'Unique';
    wrap.appendChild(badge);
  }
  return wrap;
}

function buildDetailValueContent(fieldName, value) {
  const field = fieldMeta(fieldName);
  const fragment = document.createDocumentFragment();
  if (value == null || value === '') {
    const badge = document.createElement('span');
    badge.className = 'table-badge neutral';
    badge.textContent = 'Empty';
    fragment.appendChild(badge);
    return fragment;
  }
  if (typeof value === 'boolean') {
    const badge = document.createElement('span');
    badge.className = 'table-badge boolean ' + (value ? 'boolean-true' : 'boolean-false');
    const dot = document.createElement('span');
    dot.className = 'table-dot';
    badge.appendChild(dot);
    badge.appendChild(document.createTextNode(value ? 'Yes' : 'No'));
    fragment.appendChild(badge);
    return fragment;
  }
  if (Array.isArray(value)) {
    const badge = document.createElement('span');
    badge.className = 'table-badge info';
    badge.textContent = value.length + ' item' + (value.length === 1 ? '' : 's');
    fragment.appendChild(badge);
    if (value.length) {
      const hint = document.createElement('span');
      hint.className = 'table-cell-hint';
      hint.textContent = value.map((item) => formatValue(item)).join(', ');
      fragment.appendChild(hint);
    }
    return fragment;
  }
  const primary = document.createElement('span');
  primary.className = 'detail-value-text';
  const formatted = formatFieldDisplay(field, fieldName, value);
  if (formatted.mono) {
    primary.classList.add('mono');
  }
  if (formatted.numeric) {
    primary.classList.add('numeric');
  }
  if (formatted.applied) {
    primary.textContent = formatted.primary;
    fragment.appendChild(primary);
    if (formatted.secondary) {
      const secondary = document.createElement('span');
      secondary.className = 'table-cell-hint';
      secondary.textContent = formatted.secondary;
      fragment.appendChild(secondary);
    }
    return fragment;
  }
  if (isDateLikeValue(field, value)) {
    const formattedDate = formatDateTimeParts(value);
    primary.textContent = formattedDate.primary;
    fragment.appendChild(primary);
    if (formattedDate.secondary) {
      const secondary = document.createElement('span');
      secondary.className = 'table-cell-hint';
      secondary.textContent = formattedDate.secondary;
      fragment.appendChild(secondary);
    }
    return fragment;
  }
  primary.textContent = formatValue(value);
  fragment.appendChild(primary);
  return fragment;
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

function syncRelationDropdown(field, select, summary, optionsContainer, items, dropdown) {
  if (!summary || !optionsContainer || !field?.relation) return;
  summary.textContent = relationSummaryText(field, select);
  optionsContainer.innerHTML = '';
  if (!items.length) {
    const empty = document.createElement('div');
    empty.className = 'multi-relation-empty';
    empty.textContent = 'No matching options.';
    optionsContainer.appendChild(empty);
    return;
  }
  const multiRelation = isMultiRelationField(field);
  const options = document.createElement('div');
  options.className = 'multi-relation-options';
  const selected = multiRelation
    ? new Set(selectedRelationValues(select, field).map((value) => String(value)))
    : new Set(String(selectedRelationValues(select, field) || '') ? [String(selectedRelationValues(select, field))] : []);
  items.forEach((item) => {
    const optionValue = String(item.value);
    const optionElement = document.createElement(multiRelation ? 'label' : 'button');
    optionElement.className = 'multi-relation-option';
    if (!multiRelation) {
      optionElement.type = 'button';
      optionElement.setAttribute('aria-pressed', selected.has(optionValue) ? 'true' : 'false');
    }
    if (multiRelation) {
      const checkbox = document.createElement('input');
      checkbox.type = 'checkbox';
      checkbox.value = optionValue;
      checkbox.checked = selected.has(optionValue);
      checkbox.addEventListener('change', () => {
        let option = Array.from(select.options).find((candidate) => candidate.value === optionValue);
        if (!option) {
          option = document.createElement('option');
          option.value = optionValue;
          option.textContent = item.label;
          select.appendChild(option);
        }
        option.selected = checkbox.checked;
        summary.textContent = relationSummaryText(field, select);
      });
      optionElement.appendChild(checkbox);
    } else {
      if (selected.has(optionValue)) {
        optionElement.classList.add('selected');
      }
      optionElement.addEventListener('click', () => {
        let option = Array.from(select.options).find((candidate) => candidate.value === optionValue);
        if (!option) {
          option = document.createElement('option');
          option.value = optionValue;
          option.textContent = item.label;
          select.appendChild(option);
        }
        select.value = optionValue;
        select.dispatchEvent(new Event('change', { bubbles: true }));
        if (dropdown) dropdown.open = false;
      });
    }
    const text = document.createElement('span');
    text.textContent = item.label;
    optionElement.appendChild(text);
    options.appendChild(optionElement);
  });
  optionsContainer.appendChild(options);
}

function resetQueryState() {
  cancelScheduledSearchReload();
  state.bulkSelected = {};
  state.relationSearch = {};
  state.resourceSearch = '';
  state.pagination = { page: 1, size: Number(els.pageSize.value || 10), pages: 1, total: 0 };
  state.listUpdatedAt = null;
  if (els.sidebarResourceSearch) {
    els.sidebarResourceSearch.value = '';
  }
  els.search.value = '';
  els.sort.innerHTML = '';
  els.filtersForm.innerHTML = '';
  closeColumnMenu();
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
    reloadListWithStatus('Updated search.', true).catch((error) => setStatus(String(error.message || error)));
  }, 300);
}

function filteredResources() {
  const term = String(state.resourceSearch || '').trim().toLowerCase();
  const resources = state.resources.slice().sort((a, b) => {
    const orderDelta = Number(a.order || 0) - Number(b.order || 0);
    if (orderDelta !== 0) return orderDelta;
    return String(a.label || a.name || '').localeCompare(String(b.label || b.name || ''), adminLocale || undefined);
  });
  if (!term) return resources;
  return resources.filter((resource) => {
    const label = String(resource.label || '').toLowerCase();
    const name = String(resource.name || '').toLowerCase();
    const group = String(resource.group || '').toLowerCase();
    return label.includes(term) || name.includes(term) || group.includes(term);
  });
}

function groupedResources(resources) {
  const groups = [];
  const byName = new Map();
  resources.forEach((resource) => {
    const groupName = String(resource.group || 'Resources').trim() || 'Resources';
    let group = byName.get(groupName);
    if (!group) {
      group = { name: groupName, items: [] };
      byName.set(groupName, group);
      groups.push(group);
    }
    group.items.push(resource);
  });
  return groups;
}

function renderResources() {
  els.resources.innerHTML = '';
  if (els.sidebarDashboardLink) {
    const dashboardActive = !state.current && !state.errorPage;
    els.sidebarDashboardLink.classList.toggle('active', dashboardActive);
    els.sidebarDashboardLink.setAttribute('aria-current', dashboardActive ? 'page' : 'false');
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
  groupedResources(matches).forEach((group) => {
    const groupLabel = document.createElement('li');
    groupLabel.className = 'sidebar-resource-group';
    groupLabel.textContent = group.name;
    els.resources.appendChild(groupLabel);
    group.items.forEach((resource, index) => {
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
      if (resource.description) {
        button.setAttribute('title', resource.description);
      }
      button.appendChild(icon);
      button.appendChild(label);
      button.onclick = () => selectResource(resource);
      li.appendChild(button);
      els.resources.appendChild(li);
    });
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
        updateModalLayoutMetrics();
        modal.hidden = false;
        modal.classList.add('show');
     }
   }
    document.body.classList.add('modal-open');
    enableModalScrollGuard();
  }

 function focusFirstModalControl(modal) {
   if (!modal) return;
   window.requestAnimationFrame(() => {
     const target = modal.querySelector('input:not([disabled]), select:not([disabled]), textarea:not([disabled]), button:not([disabled])');
     if (target && typeof target.focus === 'function') {
       target.focus();
     }
   });
 }

 function openModalAndFocus(modal, selector) {
   openModal(modal);
   window.requestAnimationFrame(() => {
     const target = selector ? modal?.querySelector(selector) : null;
     if (target && typeof target.focus === 'function') {
       target.focus();
       return;
     }
     focusFirstModalControl(modal);
   });
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
     disableModalScrollGuard();
   }
 }

 function clearSelectedRecord() {
   if (!state.selected) {
     highlightSelectedRow();
     syncWorkspaceActionState();
     return;
   }
   state.selected = null;
   renderSelectedRecord();
   renderUpdateForm().catch((error) => setStatus(String(error.message || error)));
 }

 function closeRecordModalAndClearSelection() {
   closeModal(els.recordModal);
   clearSelectedRecord();
 }

 function closeAllModals() {
   [els.createModal, els.recordModal, els.editModal, els.confirmModal].forEach((modal) => closeModal(modal));
   clearSelectedRecord();
 }

 function formForModal(modal) {
   if (modal === els.createModal) return { form: els.createForm, scopeKey: 'create' };
   if (modal === els.editModal) return { form: els.updateForm, scopeKey: 'update' };
   return null;
 }

 function requestCloseModal(modal, options) {
   const modalForm = formForModal(modal);
   if (!modalForm || !formState(modalForm.scopeKey).dirty) {
     closeModal(modal);
     if (options?.afterClose) options.afterClose();
     return;
   }
   openConfirmDialog(
     'Discard changes',
     'This form has unsaved changes. Close it without saving?',
     () => {
       markFormClean(modalForm.form, modalForm.scopeKey);
       closeModal(els.confirmModal);
       closeModal(modal);
       const resetForm = modalForm.scopeKey === 'create' ? renderCreateForm : renderUpdateForm;
       resetForm().catch((error) => setStatus(String(error.message || error)));
       if (options?.afterClose) options.afterClose();
     },
     'Discard'
   );
 }

 function requestCloseActiveModal() {
   if (els.confirmModal && !els.confirmModal.hidden) {
     pendingConfirmCallback = null;
     closeModal(els.confirmModal);
     return;
   }
   if (els.editModal && !els.editModal.hidden) {
     requestCloseModal(els.editModal, { afterClose: clearSelectedRecord });
     return;
   }
   if (els.createModal && !els.createModal.hidden) {
     requestCloseModal(els.createModal);
     return;
   }
   if (els.recordModal && !els.recordModal.hidden) {
     closeRecordModalAndClearSelection();
   }
 }

let _actionMenuPortal = null;
let _actionMenuTrigger = null;
let _actionMenuFrame = 0;
function getActionMenuPortal() {
  if (!_actionMenuPortal) {
    _actionMenuPortal = document.createElement('div');
    _actionMenuPortal.id = 'action-menu-portal';
    document.body.appendChild(_actionMenuPortal);
  }
  return _actionMenuPortal;
}
function positionActionMenuPortal() {
  const portal = getActionMenuPortal();
  const menu = portal.firstElementChild;
  if (!menu || !_actionMenuTrigger || !_actionMenuTrigger.isConnected) {
    closeActionMenuPortal();
    return;
  }
  const rect = _actionMenuTrigger.getBoundingClientRect();
  menu.style.position = 'fixed';
  menu.style.zIndex = '1500';
  menu.style.top = (rect.bottom + 4) + 'px';
  menu.style.right = (window.innerWidth - rect.right) + 'px';
  menu.style.left = 'auto';
}
function scheduleActionMenuPositionUpdate() {
  if (!_actionMenuPortal || !_actionMenuPortal.firstElementChild || !_actionMenuTrigger) return;
  if (_actionMenuFrame) return;
  _actionMenuFrame = window.requestAnimationFrame(() => {
    _actionMenuFrame = 0;
    positionActionMenuPortal();
  });
}
function closeActionMenuPortal() {
  const portal = getActionMenuPortal();
  if (_actionMenuFrame) {
    window.cancelAnimationFrame(_actionMenuFrame);
    _actionMenuFrame = 0;
  }
  if (_actionMenuTrigger) {
    _actionMenuTrigger.setAttribute('aria-expanded', 'false');
  }
  _actionMenuTrigger = null;
  portal.innerHTML = '';
  delete portal.dataset.forRow;
}
function openActionMenuAt(triggerEl, rowId, items) {
  const portal = getActionMenuPortal();
  const isOpen = portal.dataset.forRow === String(rowId) && portal.firstChild;
  closeActionMenuPortal();
  if (isOpen) return;
  _actionMenuTrigger = triggerEl;
  _actionMenuTrigger.setAttribute('aria-expanded', 'true');
  const menu = document.createElement('div');
  menu.className = 'action-menu-list open';
  items.forEach((item) => {
    if (item.divider) {
      const hr = document.createElement('hr');
      hr.className = 'action-menu-divider';
      menu.appendChild(hr);
    } else {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'action-menu-item' + (item.className ? ' ' + item.className : '');
      const label = document.createElement('span');
      label.className = 'action-menu-item-label';
      label.textContent = item.label;
      btn.appendChild(label);
      if (item.icon === 'trash') {
        const icon = document.createElement('span');
        icon.className = 'action-menu-item-icon';
        icon.setAttribute('aria-hidden', 'true');
        icon.innerHTML = '<svg viewBox="0 0 24 24" fill="none" focusable="false"><path d="M3 6h18" stroke="currentColor" stroke-width="2" stroke-linecap="round"></path><path d="M8 6V4c0-1.1.9-2 2-2h4c1.1 0 2 .9 2 2v2" stroke="currentColor" stroke-width="2" stroke-linecap="round"></path><path d="M19 6l-1 14c-.08 1.1-.98 2-2.08 2H8.08C6.98 22 6.08 21.1 6 20L5 6" stroke="currentColor" stroke-width="2" stroke-linecap="round"></path><path d="M10 11v6M14 11v6" stroke="currentColor" stroke-width="2" stroke-linecap="round"></path></svg>';
        btn.appendChild(icon);
      }
      btn.disabled = !!item.disabled;
      btn.onclick = (e) => { e.stopPropagation(); closeActionMenuPortal(); item.onClick(); };
      menu.appendChild(btn);
    }
  });
  portal.dataset.forRow = String(rowId);
  portal.appendChild(menu);
  positionActionMenuPortal();
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
   if (els.bulkActionExport) {
     els.bulkActionExport.disabled = !selectedIDs().length || !state.current || !hasAction('list');
   }
   if (els.bulkActionDelete) {
     els.bulkActionDelete.disabled = !selectedIDs().length || !state.current || !hasAction('bulk_delete');
   }
  if (els.openSelectedEdit) {
    els.openSelectedEdit.disabled = !state.selected || !hasAction('update');
  }
   if (els.copyRecordJSON) {
     els.copyRecordJSON.disabled = !state.selected;
   }
 }

function columnVisibilityKey() {
  return state.current ? columnVisibilityStoragePrefix + state.current.name : '';
}

function readSavedColumnVisibility() {
  const key = columnVisibilityKey();
  if (!key) return {};
  try {
    return JSON.parse(localStorage.getItem(key) || '{}') || {};
  } catch (_) {
    return {};
  }
}

function saveColumnVisibility() {
  const key = columnVisibilityKey();
  if (!key) return;
  try {
    localStorage.setItem(key, JSON.stringify(state.visibleColumns || {}));
  } catch (_) {
    // ignore storage failures
  }
}

function updateColumnToggleLabel() {
  if (!els.columnToggle) return;
  const fields = state.meta?.list_fields || [];
  if (!els.columnToggle.querySelector('span')) {
    els.columnToggle.textContent = 'View';
  }
  if (!state.current || !fields.length) {
    els.columnToggle.setAttribute('aria-label', 'Choose visible columns');
    return;
  }
  const visible = visibleListFields().length;
  els.columnToggle.setAttribute('aria-label', 'Choose visible columns. ' + visible + ' of ' + fields.length + ' columns visible.');
}

function initializeColumnVisibility() {
  const fields = state.meta?.list_fields || [];
  const saved = readSavedColumnVisibility();
  const next = {};
  fields.forEach((name) => {
    next[name] = Object.prototype.hasOwnProperty.call(saved, name) ? saved[name] !== false : true;
  });
  state.visibleColumns = next;
  renderColumnMenu();
  updateColumnToggleLabel();
}

function visibleListFields() {
  const fields = state.meta?.list_fields || [];
  return fields.filter((name) => state.visibleColumns[name] !== false);
}

function closeColumnMenu() {
  if (!els.columnMenu || !els.columnToggle) return;
  els.columnMenu.hidden = true;
  els.columnToggle.setAttribute('aria-expanded', 'false');
}

function renderColumnMenu() {
  if (!els.columnMenu) return;
  const fields = state.meta?.list_fields || [];
  els.columnMenu.innerHTML = '';
  updateColumnToggleLabel();
  const title = document.createElement('p');
  title.className = 'column-menu-title';
  title.textContent = 'Toggle columns';
  els.columnMenu.appendChild(title);
  if (!fields.length) {
    const empty = document.createElement('div');
    empty.className = 'muted';
    empty.textContent = 'No list columns available.';
    els.columnMenu.appendChild(empty);
    return;
  }
  fields.forEach((name) => {
    const option = document.createElement('button');
    option.type = 'button';
    option.className = 'column-option';
    option.setAttribute('role', 'menuitemcheckbox');
    option.setAttribute('aria-checked', String(state.visibleColumns[name] !== false));

    const check = document.createElement('span');
    check.className = 'column-option-check';
    check.setAttribute('aria-hidden', 'true');
    check.textContent = state.visibleColumns[name] !== false ? '\u2713' : '';

    const text = document.createElement('span');
    text.className = 'column-option-label';
    text.textContent = fieldLabel(name);

    option.addEventListener('click', () => {
      const nextVisible = state.visibleColumns[name] === false;
      if (!nextVisible && visibleListFields().length <= 1) {
        showToast('Keep at least one table column visible.', 'info');
        return;
      }
      state.visibleColumns[name] = nextVisible;
      saveColumnVisibility();
      updateColumnToggleLabel();
      renderColumnMenu();
      renderList().catch((error) => setStatus(String(error.message || error)));
    });
    option.appendChild(check);
    option.appendChild(text);
    els.columnMenu.appendChild(option);
  });
}

function renderResourceSummary() {
  if (!state.current || !state.meta) {
    els.resourcePath.textContent = 'Sign in to open a resource workspace.';
    els.resourcePath.hidden = false;
    return;
  }
  const description = String(state.meta.description || '').trim();
  els.resourcePath.textContent = description;
  els.resourcePath.hidden = !description;
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

function searchPlaceholderLabels() {
  const meta = state.meta || {};
  const names = Array.isArray(meta.search_fields) ? meta.search_fields : [];
  if (!names.length) return [];
  const labelByFieldName = new Map((meta.fields || []).map((field) => [field.name, field.label || field.name]));
  return names
    .map((name) => String(labelByFieldName.get(name) || name).trim())
    .filter(Boolean);
}

function renderSearchPlaceholder() {
  const labels = searchPlaceholderLabels();
  els.search.placeholder = labels.length ? 'Search by ' + labels.join(', ') : 'Search current resource';
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

function resourceIcon(resource) {
  const iconByKey = {
    users: {
      icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M8 8a2.75 2.75 0 1 0 0-5.5A2.75 2.75 0 0 0 8 8Z"/><path d="M3.5 13.25a4.5 4.5 0 0 1 9 0"/></svg>'
    },
    user: {
      icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M8 8a2.75 2.75 0 1 0 0-5.5A2.75 2.75 0 0 0 8 8Z"/><path d="M3.5 13.25a4.5 4.5 0 0 1 9 0"/></svg>'
    },
    roles: {
      icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M8 2.5 3.5 4.75v3c0 2.9 1.85 5.5 4.5 6.25 2.65-.75 4.5-3.35 4.5-6.25v-3z"/><path d="m6.5 8 1 1 2-2.25"/></svg>'
    },
    shield: {
      icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M8 2.5 3.5 4.75v3c0 2.9 1.85 5.5 4.5 6.25 2.65-.75 4.5-3.35 4.5-6.25v-3z"/><path d="m6.5 8 1 1 2-2.25"/></svg>'
    },
    projects: {
      icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M3 4.5h10"/><path d="M5 2.75v3.5"/><path d="M11 2.75v3.5"/><rect x="3" y="4.5" width="10" height="8.5" rx="1.25"/><path d="M6 8h4"/><path d="M6 10.5h2.5"/></svg>'
    },
    briefcase: {
      icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M3 4.5h10"/><path d="M5 2.75v3.5"/><path d="M11 2.75v3.5"/><rect x="3" y="4.5" width="10" height="8.5" rx="1.25"/><path d="M6 8h4"/><path d="M6 10.5h2.5"/></svg>'
    },
    table: {
      icon: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect x="3" y="3.5" width="10" height="9" rx="1.25"/><path d="M6 6.5h4"/><path d="M6 9h4"/></svg>'
    }
  };
  const iconKey = String(resource?.icon || '').trim().toLowerCase();
  const nameKey = String(resource?.name || '').trim().toLowerCase();
  return (iconByKey[iconKey] || iconByKey[nameKey] || iconByKey.table).icon;
}

function dashboardTileMeta(resource, index) {
  const palette = ['#007bff', '#17a2b8', '#6f42c1', '#fd7e14', '#20c997', '#e83e8c'];
  const byName = {
    users: {
      badge: 'Core access'
    },
    roles: {
      badge: 'Permissions'
    },
    projects: {
      badge: 'Delivery'
    }
  };
  const normalizedName = String(resource?.name || '').toLowerCase();
  const meta = byName[normalizedName] || {
    badge: 'Workspace'
  };
  return {
    accent: palette[index % palette.length],
    badge: resource?.group || meta.badge,
    description: String(resource?.description || '').trim(),
    icon: resourceIcon(resource)
  };
}

function renderDashboard() {
  if (!els.dashboardShell || !els.dashboardTiles) return;
  if (state.errorPage || state.current || !state.resources.length) {
    els.dashboardShell.hidden = true;
    return;
  }
  els.dashboardShell.hidden = false;
  els.dashboardTiles.innerHTML = '';
  const tileRefs = [];
  state.resources.forEach((resource, index) => {
    const meta = dashboardTileMeta(resource, index);
    const tile = document.createElement('button');
    tile.type = 'button';
    tile.className = 'dashboard-tile small-box bg-info';
    tile.style.setProperty('--dashboard-tile-accent', meta.accent);
    tile.setAttribute('aria-label', 'Open ' + resource.label + ' workspace');
    const descriptionHTML = meta.description
      ? '<p class="dashboard-tile-description">' + escapeHTML(meta.description) + '</p>'
      : '';
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
        descriptionHTML +
      '</div>' +
      '<div class="dashboard-tile-footer">' +
        '<span class="dashboard-tile-action">Open workspace</span>' +
        '<span class="dashboard-tile-arrow" aria-hidden="true">→</span>' +
      '</div>';
    tile.onclick = () => selectResource(resource);
    els.dashboardTiles.appendChild(tile);
    tileRefs.push({ resource, tile });
  });
  loadDashboardCounts(tileRefs);
}

function setDashboardTileCount(tile, total) {
  const countEl = tile ? tile.querySelector('.dashboard-tile-count') : null;
  if (countEl) countEl.textContent = String(total ?? dashboardCountPlaceholder);
}

function loadDashboardCountsFallback(tileRefs) {
  tileRefs.forEach(({ resource, tile }) => {
    const basePath = apiBase + '/resources' + resource.path;
    request(basePath + '?page=1&size=1')
      .then((data) => setDashboardTileCount(tile, data.total))
      .catch((err) => {
        // Count is decorative; log but don't surface to user
        console.error('dashboard tile count load failed for ' + resource.name + ':', err);
      });
  });
}

async function loadDashboardCounts(tileRefs) {
  if (!tileRefs.length) return;
  try {
    const stats = await request(apiBase + '/resources/stats');
    const totals = new Map((stats.resources || []).map((item) => [item.name, item.total]));
    tileRefs.forEach(({ resource, tile }) => {
      setDashboardTileCount(tile, totals.has(resource.name) ? totals.get(resource.name) : dashboardCountPlaceholder);
    });
  } catch (err) {
    console.error('dashboard stats load failed:', err);
    loadDashboardCountsFallback(tileRefs);
  }
}

function normalizedFilterPlaceholder(field, fallback) {
  const label = String(field?.label || field?.name || '').trim();
  const placeholder = String(field?.placeholder || '').trim();
  if (placeholder && placeholder.toLowerCase() !== label.toLowerCase()) {
    return placeholder;
  }
  return fallback;
}

function lowerFirst(value) {
  return String(value || '').replace(/^./, (char) => char.toLowerCase());
}

function filterControlLabel(field) {
  const label = String(field?.label || field?.name || 'Value').trim();
  if (isDateTimeField(field)) return label + ' between';
  if (field?.component === 'checkbox') return label;
  return label + ' is';
}

function emptyFilterPlaceholder(field) {
  const label = lowerFirst(field?.label || field?.name || 'value');
  if (isDateTimeField(field)) return 'Select date';
  if (field?.component === 'checkbox') return 'All';
  return 'Any ' + label;
}

function buildFilterControl(field) {
  const wrapper = document.createElement('div');
  const componentClass = String(field.component || 'text').replace(/[^a-z0-9_-]/gi, '-').toLowerCase();
  wrapper.className = 'inline-field form-group filter-control filter-control-' + componentClass;
  const label = document.createElement('label');
  label.className = 'filter-control-label';
  label.textContent = filterControlLabel(field);
  if (isDateTimeField(field)) {
    const range = document.createElement('div');
    range.className = 'filter-date-range';
    const from = document.createElement('input');
    from.type = 'date';
    from.className = 'form-control filter-date-input';
    from.name = field.name + '__from';
    from.id = 'filter-' + field.name + '-from';
    from.setAttribute('aria-label', field.label + ' start date');
    from.setAttribute('title', field.label + ' start date');
    const separator = document.createElement('span');
    separator.className = 'filter-date-separator';
    separator.textContent = 'to';
    const to = document.createElement('input');
    to.type = 'date';
    to.className = 'form-control filter-date-input';
    to.name = field.name + '__to';
    to.id = 'filter-' + field.name + '-to';
    to.setAttribute('aria-label', field.label + ' end date');
    to.setAttribute('title', field.label + ' end date');
    [from, to].forEach((input) => {
      input.addEventListener('input', () => syncFilterFacetState(input));
      input.addEventListener('change', () => syncFilterFacetState(input));
    });
    range.appendChild(from);
    range.appendChild(separator);
    range.appendChild(to);
    wrapper.appendChild(label);
    wrapper.appendChild(range);
    els.filtersForm.appendChild(wrapper);
    syncFilterFacetState(from);
    return;
  }
  let input;
  if (field.component === 'checkbox') {
    input = document.createElement('select');
    input.className = 'custom-select';
    [['', normalizedFilterPlaceholder(field, emptyFilterPlaceholder(field))], ['true', 'Yes'], ['false', 'No']].forEach((pair) => {
      const option = document.createElement('option');
      option.value = pair[0];
      option.textContent = pair[1];
      input.appendChild(option);
    });
  } else if (field.component === 'number') {
    input = document.createElement('input');
    input.type = 'number';
    input.className = 'form-control';
    input.placeholder = normalizedFilterPlaceholder(field, emptyFilterPlaceholder(field));
  } else if (field.component === 'datetime') {
    input = document.createElement('input');
    input.type = 'datetime-local';
    input.className = 'form-control';
    input.placeholder = normalizedFilterPlaceholder(field, emptyFilterPlaceholder(field));
  } else {
    input = document.createElement('input');
    input.type = 'text';
    input.className = 'form-control';
    input.placeholder = normalizedFilterPlaceholder(field, emptyFilterPlaceholder(field));
  }
  input.name = field.name;
  input.id = 'filter-' + field.name;
  input.setAttribute('aria-label', 'Filter ' + field.label);
  input.setAttribute('title', 'Filter ' + field.label);
  label.setAttribute('for', input.id);
  input.addEventListener('input', () => syncFilterFacetState(input));
  input.addEventListener('change', () => syncFilterFacetState(input));
  wrapper.appendChild(label);
  wrapper.appendChild(input);
  els.filtersForm.appendChild(wrapper);
  syncFilterFacetState(input);
}

function syncFilterFacetState(input) {
  const wrapper = input?.closest?.('.filter-control');
  if (!wrapper) return;
  const active = Array.from(wrapper.querySelectorAll('input, select, textarea'))
    .some((control) => String(control.value || '').trim() !== '');
  wrapper.classList.toggle('is-active', active);
}

function syncFilterFacetStates() {
  if (!els.filtersForm) return;
  Array.from(els.filtersForm.elements).forEach((element) => {
    if ('value' in element) syncFilterFacetState(element);
  });
}

function renderFilterControls() {
  els.filtersForm.innerHTML = '';
  const filterFields = state.meta?.filter_fields || [];
  if (!filterFields.length) {
    els.filtersForm.hidden = true;
    return;
  }
  els.filtersForm.hidden = false;
  filterFields.forEach((name) => {
    const field = fieldMeta(name);
    if (field) buildFilterControl(field);
  });
  const reset = document.createElement('button');
  reset.id = 'resetFilters';
  reset.className = 'filter-reset-btn btn btn-default';
  reset.type = 'button';
  reset.innerHTML = '<span>Reset</span><svg viewBox="0 0 16 16" aria-hidden="true" focusable="false"><path d="m4.5 4.5 7 7"/><path d="m11.5 4.5-7 7"/></svg>';
  reset.onclick = () => {
    if (!state.current) return;
    cancelScheduledSearchReload();
    clearListFilters();
    syncResetFiltersButton();
    reloadListWithStatus('Cleared list filters.', true).catch((error) => setStatus(String(error.message || error)));
  };
  reset.hidden = true;
  els.filtersForm.appendChild(reset);
  syncResetFiltersButton();
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

function scheduleRelationSearch(field, scopeKey, searchInput, select, summary, optionsContainer, dropdown, itemsRef) {
  const key = relationStateKey(scopeKey, field);
  clearTimeout(state.relationTimers[key]);
  state.relationTimers[key] = setTimeout(async () => {
    try {
      const term = searchInput.value.trim();
      const items = await loadRelationOptions(field, term, 1, 8);
      if (itemsRef) itemsRef.items = items;
      const nextValue = resolveRelationSelection(field, items, selectedRelationValues(select, field), term);
      populateRelationSelect(field, select, items, nextValue, field.label);
      syncRelationDropdown(field, select, summary, optionsContainer, items, dropdown);
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
    searchInput.placeholder = field.placeholder || ('Search related ' + field.label);
    searchInput.value = state.relationSearch[searchKey] || '';
    searchInput.className = 'form-control';
    const select = document.createElement('select');
    select.name = field.name;
    select.className = 'custom-select';
    const multiRelation = isMultiRelationField(field);
    const dropdown = document.createElement('details');
    const dropdownSummary = document.createElement('summary');
    const dropdownMenu = document.createElement('div');
    const dropdownOptions = document.createElement('div');
    dropdown.className = 'multi-relation-dropdown';
    dropdownSummary.className = 'multi-relation-summary';
    dropdownMenu.className = 'multi-relation-menu';
    dropdownOptions.className = 'multi-relation-options';
    searchInput.classList.add('relation-search');
    dropdown.appendChild(dropdownSummary);
    dropdownMenu.appendChild(searchInput);
    dropdownMenu.appendChild(dropdownOptions);
    dropdown.appendChild(dropdownMenu);
    select.hidden = true;
    dropdown.addEventListener('toggle', () => {
      if (dropdown.open) {
        window.requestAnimationFrame(() => searchInput.focus());
      }
    });
    const help = document.createElement('div');
    help.className = 'field-help';
    help.textContent = multiRelation
      ? 'Search related records and choose one or more matching options for this field.'
      : 'Search related records and choose the best matching option for this field.';
    wrapper.appendChild(dropdown);
    wrapper.appendChild(select);
    wrapper.appendChild(help);
    const itemsRef = { items: await loadRelationOptions(field, searchInput.value.trim(), 1, 8) };
    const nextValue = resolveRelationSelection(field, itemsRef.items, value, searchInput.value);
    populateRelationSelect(field, select, itemsRef.items, nextValue, field.label);
    syncRelationDropdown(field, select, dropdownSummary, dropdownOptions, itemsRef.items, dropdown);
    searchInput.addEventListener('input', () => {
      state.relationSearch[searchKey] = searchInput.value;
      scheduleRelationSearch(field, scopeKey, searchInput, select, dropdownSummary, dropdownOptions, dropdown, itemsRef);
    });
    select.addEventListener('change', () => {
      syncRelationDropdown(field, select, dropdownSummary, dropdownOptions, itemsRef.items, dropdown);
    });
    return wrapper;
  }
  if (field.component === 'checkbox') {
    const input = document.createElement('input');
    input.type = 'checkbox';
    input.name = field.name;
    input.checked = Boolean(value);
    input.className = 'form-check-input switch-input';
    return input;
  }
  if (field.component === 'array' || field.component === 'text' || field.component === 'textarea') {
    const input = document.createElement('textarea');
    input.name = field.name;
    input.className = 'form-control';
    input.placeholder = field.placeholder || '';
    input.value = field.component === 'array'
      ? (Array.isArray(value) ? JSON.stringify(value, null, 2) : (value ? JSON.stringify(value, null, 2) : ''))
      : (value == null ? '' : String(value));
    return input;
  }
  const input = document.createElement('input');
  input.name = field.name;
  input.type = ({ email: 'email', password: 'password', number: 'number', datetime: 'datetime-local' }[field.component]) || 'text';
  input.value = value == null ? '' : String(value);
  input.placeholder = field.placeholder || '';
  input.className = 'form-control';
  return input;
}

async function renderForm(target, fieldNames, mode, values, scopeKey) {
  target.innerHTML = '';
  target.className = 'stack resource-form';
  target.noValidate = true;
  if (!state.meta || !fieldNames.length) {
    target.innerHTML = '<p class="muted">' + mode + ' is not available for this resource.</p>';
    markFormClean(target, scopeKey);
    return;
  }
  const intro = document.createElement('div');
  intro.className = 'form-section-intro';
  const introCopyWrap = document.createElement('div');
  introCopyWrap.className = 'form-section-copy-wrap';
  const introTitle = document.createElement('h4');
  introTitle.className = 'form-section-title';
  introTitle.textContent = mode === 'update' ? 'Edit form' : 'Create form';
  const introCopy = document.createElement('p');
  introCopy.className = 'form-section-copy';
  introCopy.textContent = mode === 'update'
    ? 'Review the fields below and save the changes when ready.'
    : 'Fill in the fields below and submit to create the record.';
  const introMeta = document.createElement('span');
  introMeta.className = 'form-section-meta';
  introMeta.textContent = fieldNames.length + ' field' + (fieldNames.length === 1 ? '' : 's');
  introCopyWrap.appendChild(introTitle);
  introCopyWrap.appendChild(introCopy);
  intro.appendChild(introCopyWrap);
  intro.appendChild(introMeta);
  target.appendChild(intro);
  const grid = document.createElement('div');
  grid.className = 'form-grid';
  for (const name of fieldNames) {
    const field = fieldMeta(name);
    if (!field) continue;
    const wrapper = document.createElement('div');
    wrapper.className = 'form-field-card';
    wrapper.dataset.fieldName = field.name;
    if (shouldSpanFullWidth(field)) {
      wrapper.classList.add('form-field-wide');
    }
    if (field.component === 'checkbox') {
      wrapper.classList.add('form-field-checkbox');
    }
    const control = await buildFieldControl(field, values[name], scopeKey);
    applyFieldControlState(control, field, scopeKey);
    const header = document.createElement('div');
    header.className = 'form-field-header';
    const copy = document.createElement('div');
    copy.className = 'form-field-copy';
    if (field.component !== 'checkbox') {
      const label = document.createElement('label');
      label.className = 'form-field-label';
      label.setAttribute('for', fieldControlID(scopeKey, field));
      label.textContent = field.label;
      copy.appendChild(label);
    }
    if (field.description) {
      const description = document.createElement('p');
      description.className = 'form-field-description';
      description.textContent = field.description;
      copy.appendChild(description);
    }
    if (field.help) {
      const help = document.createElement('p');
      help.className = 'form-field-description';
      help.textContent = field.help;
      copy.appendChild(help);
    }
    const meta = document.createElement('div');
    meta.className = 'form-field-meta';
    if (isFieldRequiredForForm(field, scopeKey)) {
      meta.appendChild(createFieldTag('Required', 'required'));
    }
    if (field.read_only) {
      meta.appendChild(createFieldTag('Read only', 'readonly'));
    }
    if (field.relation) {
      meta.appendChild(createFieldTag('Relation', ''));
    }
    if (copy.childNodes.length || meta.childNodes.length) {
      if (copy.childNodes.length) {
        header.appendChild(copy);
      }
      if (meta.childNodes.length) {
        header.appendChild(meta);
      }
      wrapper.appendChild(header);
    }
    const controlWrap = document.createElement('div');
    controlWrap.className = 'form-field-control';
    if (field.component === 'checkbox') {
      const toggle = document.createElement('label');
      toggle.className = 'form-toggle';
      const toggleCopy = document.createElement('span');
      toggleCopy.className = 'form-toggle-copy';
      const toggleLabel = document.createElement('span');
      toggleLabel.className = 'form-toggle-label';
      toggleLabel.textContent = field.label;
      const toggleHelp = document.createElement('span');
      toggleHelp.className = 'form-toggle-help';
      toggleHelp.textContent = field.description || 'Enable or disable this option.';
      toggleCopy.appendChild(toggleLabel);
      toggleCopy.appendChild(toggleHelp);
      toggle.appendChild(toggleCopy);
      toggle.appendChild(control);
      controlWrap.appendChild(toggle);
    } else {
      controlWrap.appendChild(control);
    }
    wrapper.appendChild(controlWrap);
    const errorEl = document.createElement('p');
    errorEl.className = 'field-error';
    errorEl.id = fieldControlID(scopeKey, field) + '-error';
    errorEl.hidden = true;
    wrapper.appendChild(errorEl);
    grid.appendChild(wrapper);
  }
  target.appendChild(grid);
  const footer = document.createElement('div');
  footer.className = 'resource-form-footer';
  const footerCopy = document.createElement('p');
  footerCopy.className = 'muted form-status';
  footerCopy.dataset.formStatus = scopeKey;
  footerCopy.textContent = mode === 'update'
    ? 'Review the changes before saving.'
    : 'Only required fields need values to create the record.';
  const submit = document.createElement('button');
  submit.type = 'submit';
  submit.textContent = mode === 'update' ? 'Update' : 'Create';
  submit.dataset.defaultText = submit.textContent;
  submit.dataset.pendingText = mode === 'update' ? 'Saving...' : 'Creating...';
  submit.className = 'btn btn-primary';
  footer.appendChild(footerCopy);
  footer.appendChild(submit);
  target.appendChild(footer);
  markFormClean(target, scopeKey);
}

async function renderCreateForm() {
  await renderForm(els.createForm, state.meta?.create_fields || [], 'create', {}, 'create');
}

async function renderUpdateForm() {
  if (!state.selected) {
    els.updateForm.innerHTML = '<p class="muted">Select a row to edit it.</p>';
    els.editHint.textContent = 'Select a row to edit.';
    markFormClean(els.updateForm, 'update');
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
    syncWorkspaceActionState();
    return;
  }
  const record = state.selected.item || {};
  const recordID = recordPrimaryKey(record);
  els.detailTitle.textContent = state.meta.label + ' #' + recordID;
  els.detailObjectBadge.textContent = 'Record overview';
  els.detail.textContent = JSON.stringify(record, null, 2);
  const detailFields = state.meta?.detail_fields || Object.keys(record);
  detailFields.forEach((name) => {
    const labelText = fieldMeta(name)?.label || name;
    const row = document.createElement('div');
    row.className = 'detail-row';
    const label = document.createElement('div');
    label.className = 'detail-label';
    label.textContent = labelText;
    const value = document.createElement('div');
    value.className = 'detail-value';
    value.appendChild(buildDetailValueContent(name, record[name]));
    row.appendChild(label);
    row.appendChild(value);
    els.detailFields.appendChild(row);
  });
  syncWorkspaceActionState();
}

async function copyTextToClipboard(text) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand('copy');
  textarea.remove();
}

async function copySelectedRecordJSON() {
  if (!state.selected) return;
  const text = JSON.stringify(state.selected.item || {}, null, 2);
  await copyTextToClipboard(text);
  showToast('Copied record JSON.', 'success');
  setStatus('Copied record JSON.');
}

async function openSelectedRecordEdit() {
  if (!state.selected || !hasAction('update')) return;
  await renderUpdateForm();
  closeModal(els.recordModal);
  openModalAndFocus(els.editModal, 'input:not([disabled]), select:not([disabled]), textarea:not([disabled])');
}

function syncBulkActionState() {
  const count = selectedIDs().length;
  els.selectedCountBadge.textContent = count + ' selected';
  if (els.bulkActionBar) {
    els.bulkActionBar.hidden = count === 0;
  }
  if (els.bulkActionCount) {
    els.bulkActionCount.textContent = String(count);
  }
  if (els.bulkActionLabel) {
    els.bulkActionLabel.textContent = (count === 1 ? 'record selected' : 'records selected');
  }
  if (els.copySelectedIDs) {
    els.copySelectedIDs.disabled = count === 0;
    els.copySelectedIDs.hidden = count === 0;
  }
  if (els.clearSelection) {
    els.clearSelection.disabled = count === 0;
    els.clearSelection.hidden = count === 0;
  }
  if (els.bulkActionDelete) {
    els.bulkActionDelete.disabled = count === 0 || !hasAction('bulk_delete');
  }
  if (els.bulkActionExport) {
    els.bulkActionExport.disabled = count === 0 || !state.current || !hasAction('list');
  }
  syncWorkspaceActionState();
}

function clearBulkSelection() {
  state.bulkSelected = {};
  syncBulkActionState();
  syncBulkRowSelection();
  setStatus('Cleared selected records.');
}

async function copyBulkSelectedIDs() {
  const ids = selectedIDs();
  if (!ids.length) return;
  await copyTextToClipboard(ids.map((id) => String(id)).join('\n'));
  showToast('Copied ' + ids.length + ' selected ID(s).', 'success');
  setStatus('Copied ' + ids.length + ' selected ID(s).');
}

function bulkDeleteSelectedRecords() {
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
        syncBulkActionState();
        syncBulkRowSelection();
        const deleted = String(result.deleted || 0);
        rewindPageIfCurrentPageEmptied(ids);
        showToast('Bulk deleted ' + deleted + ' record(s).', 'success');
        await reloadListWithStatus('Bulk deleted ' + deleted + ' record(s).', false);
      } catch (error) {
        showToast(String(error.message || error), 'danger');
        setStatus(String(error.message || error));
      }
    },
    'Delete ' + count
  );
}

function buildValidatedFormPayload(form, scopeKey) {
  clearFormErrors(form);
  const errors = [];
  (state.meta?.fields || []).forEach((field) => {
    if (!formFieldWrapper(form, field.name) || field.read_only) return;
    const value = formFieldValue(form, field);
    const empty = Array.isArray(value)
      ? value.length === 0
      : value == null || value === '';
    if (isFieldRequiredForForm(field, scopeKey) && empty) {
      errors.push({ field: field.name, message: field.label + ' is required.' });
    }
    if (field.component === 'number' && !empty && !Number.isFinite(Number(value))) {
      errors.push({ field: field.name, message: field.label + ' must be a valid number.' });
    }
  });

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
      try {
        payload[key] = value ? JSON.parse(value) : [];
        if (!Array.isArray(payload[key])) {
          errors.push({ field: key, message: field.label + ' must be a JSON array.' });
        }
      } catch (_) {
        errors.push({ field: key, message: field.label + ' must be valid JSON.' });
      }
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
  errors.forEach((error) => setFormFieldError(form, error.field, error.message));
  return {
    valid: errors.length === 0,
    payload,
    errors
  };
}

function appendListQueryParams(params) {
  if (els.search.value.trim()) {
    params.set('search', els.search.value.trim());
  }
  if (els.sort.value) {
    params.set('sort', els.sort.value);
  }
  params.set('page', String(state.pagination.page));
  params.set('size', String(state.pagination.size));
  appendFilterParams(params, { prefix: 'f.' });
}

function syncListURLState() {
  if (!state.current) return;
  updateNavigationState('replace', 'resource', state.current.name, { includeListState: true });
}

function hasListStateParams(params) {
  if (!params) return false;
  if (params.has('search') || params.has('sort') || params.has('page') || params.has('size')) return true;
  return Array.from(params.keys()).some((key) => key.startsWith('f.'));
}

function applyListStateFromURL(resourceName) {
  const params = new URLSearchParams(window.location.search || '');
  if (resourceNameFromAdminPath(currentPagePath()) !== resourceName || !hasListStateParams(params)) return false;
  els.search.value = params.get('search') || '';
  const sortValue = params.get('sort') || '';
  if (sortValue && Array.from(els.sort.options).some((option) => option.value === sortValue)) {
    els.sort.value = sortValue;
  }
  const sizeValue = params.get('size') || '';
  if (sizeValue && Array.from(els.pageSize.options).some((option) => option.value === sizeValue)) {
    els.pageSize.value = sizeValue;
    state.pagination.size = Number(sizeValue);
  }
  const pageValue = Number(params.get('page') || '1');
  state.pagination.page = Number.isFinite(pageValue) && pageValue > 0 ? pageValue : 1;
  (state.meta?.filter_fields || []).forEach((name) => {
    const field = fieldMeta(name);
    if (isDateTimeField(field)) {
      setFilterStateValue(name, {
        from: params.get('f.' + name + '__from') || '',
        to: params.get('f.' + name + '__to') || ''
      });
      return;
    }
    setFilterStateValue(name, params.get('f.' + name) || '');
  });
  syncResetFiltersButton();
  return true;
}

function listStateStorageKey(resourceName) {
  return listStateStoragePrefix + resourceName;
}

function collectCurrentListState() {
  const filters = {};
  (state.meta?.filter_fields || []).forEach((name) => {
    const value = getFilterStateValue(name);
    if (value != null) filters[name] = value;
  });
  return {
    search: els.search.value.trim(),
    sort: els.sort.value || '',
    page: state.pagination.page,
    size: state.pagination.size,
    filters
  };
}

function saveCurrentListState() {
  if (!state.current) return;
  try {
    localStorage.setItem(listStateStorageKey(state.current.name), JSON.stringify(collectCurrentListState()));
  } catch (_) {
    // ignore storage failures
  }
}

function readSavedListState(resourceName) {
  try {
    return JSON.parse(localStorage.getItem(listStateStorageKey(resourceName)) || 'null');
  } catch (_) {
    return null;
  }
}

function applyListStateSnapshot(saved) {
  if (!saved || typeof saved !== 'object') return false;
  els.search.value = String(saved.search || '');
  const sortValue = String(saved.sort || '');
  if (sortValue && Array.from(els.sort.options).some((option) => option.value === sortValue)) {
    els.sort.value = sortValue;
  } else {
    els.sort.value = '';
  }
  const sizeValue = String(saved.size || '');
  if (sizeValue && Array.from(els.pageSize.options).some((option) => option.value === sizeValue)) {
    els.pageSize.value = sizeValue;
    state.pagination.size = Number(sizeValue);
  }
  const pageValue = Number(saved.page || '1');
  state.pagination.page = Number.isFinite(pageValue) && pageValue > 0 ? pageValue : 1;
  const filters = saved.filters && typeof saved.filters === 'object' ? saved.filters : {};
  (state.meta?.filter_fields || []).forEach((name) => {
    setFilterStateValue(name, filters[name] == null ? '' : filters[name]);
  });
  syncResetFiltersButton();
  return true;
}

function applySavedListState(resourceName) {
  return applyListStateSnapshot(readSavedListState(resourceName));
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
  appendFilterParams(params, { boundary: true });
  return '?' + params.toString();
}

function activeListEntries() {
  if (!state.current) return [];
  const entries = [];
  const searchValue = els.search.value.trim();
  if (searchValue) {
    entries.push({ label: 'Search', clearable: true });
  }
  if (els.sort.value) {
    entries.push({ label: 'Sort', clearable: true });
  }
  (state.meta?.filter_fields || []).forEach((name) => {
    const value = getFilterStateValue(name);
    if (value == null) return;
    entries.push({ label: fieldLabel(name), clearable: true });
  });
  return entries;
}

function hasClearableListState() {
  return activeListEntries().some((entry) => entry.clearable);
}

function syncResetFiltersButton() {
  const reset = document.getElementById('resetFilters');
  if (!reset) return;
  reset.hidden = !hasClearableListState();
}

function renderListEmptyState(title, message, actions) {
  const empty = document.createElement('div');
  empty.className = 'empty-state';
  const heading = document.createElement('p');
  heading.className = 'empty-state-title';
  heading.textContent = title;
  const copy = document.createElement('p');
  copy.className = 'empty-state-copy';
  copy.textContent = message;
  empty.appendChild(heading);
  empty.appendChild(copy);
  const availableActions = (actions || []).filter(Boolean);
  if (availableActions.length) {
    const row = document.createElement('div');
    row.className = 'empty-state-actions';
    availableActions.forEach((action) => {
      const button = document.createElement('button');
      button.type = 'button';
      button.className = action.primary ? 'btn btn-primary' : 'secondary btn btn-default';
      button.textContent = action.label;
      button.onclick = action.onClick;
      row.appendChild(button);
    });
    empty.appendChild(row);
  }
  els.list.innerHTML = '';
  els.list.appendChild(empty);
}

function csvCell(value) {
  let text;
  if (value == null) {
    text = '';
  } else if (Array.isArray(value) || typeof value === 'object') {
    text = JSON.stringify(value);
  } else {
    text = String(value);
  }
  if (/[",\r\n]/.test(text)) {
    return '"' + text.replaceAll('"', '""') + '"';
  }
  return text;
}

function downloadCSV(filename, rows) {
  const content = '\ufeff' + rows.map((row) => row.map(csvCell).join(',')).join('\r\n');
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}

async function selectedRowsForExport(ids) {
  const currentRows = new Map(state.records.map((row) => [selectionKey(recordPrimaryKey(row)), row]));
  return Promise.all(ids.map(async (id) => {
    const currentRow = currentRows.get(selectionKey(id));
    if (currentRow) return currentRow;
    const payload = await request(currentBasePath() + '/' + encodeURIComponent(String(id)));
    return payload.item || payload;
  }));
}

async function exportSelectedRecordsCSV() {
  const ids = selectedIDs();
  if (!state.current || !ids.length || !els.bulkActionExport) return;
  const label = els.bulkActionExport.querySelector('span');
  const originalText = label ? label.textContent : els.bulkActionExport.textContent;
  const originalTooltip = els.bulkActionExport.dataset.tooltip || 'Export CSV';
  els.bulkActionExport.disabled = true;
  els.bulkActionExport.dataset.pending = 'true';
  els.bulkActionExport.dataset.tooltip = 'Exporting...';
  if (label) {
    label.textContent = 'Exporting...';
  } else {
    els.bulkActionExport.textContent = 'Exporting...';
  }
  try {
    const fields = visibleListFields();
    const rows = await selectedRowsForExport(ids);
    const csvRows = [
      fields.map((field) => fieldLabel(field)),
      ...rows.map((row) => fields.map((field) => row[field]))
    ];
    downloadCSV((state.current.name || 'resource') + '-selected.csv', csvRows);
    showToast('Exported ' + ids.length + ' selected record(s).', 'success');
    setStatus('Exported ' + ids.length + ' selected record(s).');
  } catch (error) {
    const message = String(error.message || error);
    showToast(message, 'danger');
    setStatus(message);
  } finally {
    if (label) {
      label.textContent = originalText;
    } else {
      els.bulkActionExport.textContent = originalText;
    }
    delete els.bulkActionExport.dataset.pending;
    els.bulkActionExport.dataset.tooltip = originalTooltip;
    syncWorkspaceActionState();
  }
}

function paginationSummaryText() {
  const page = Math.max(1, Number(state.pagination.page || 1));
  const pages = Math.max(1, Number(state.pagination.pages || 1));
  return 'Page ' + page + ' of ' + pages;
}

function paginationItems(page, pages) {
  const totalPages = Math.max(1, Number(pages || 1));
  const current = Math.min(Math.max(1, Number(page || 1)), totalPages);
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => index + 1);
  }
  let start = Math.max(2, current - 1);
  let end = Math.min(totalPages - 1, current + 2);
  if (current <= 3) {
    start = 2;
    end = 4;
  } else if (current >= totalPages - 2) {
    start = totalPages - 3;
    end = totalPages - 1;
  }
  const items = [1];
  if (start > 2) items.push('ellipsis-start');
  for (let pageNumber = start; pageNumber <= end; pageNumber += 1) {
    items.push(pageNumber);
  }
  if (end < totalPages - 1) items.push('ellipsis-end');
  items.push(totalPages);
  return items;
}

function renderPagination() {
  const page = Math.max(1, Number(state.pagination.page || 1));
  const pages = Math.max(1, Number(state.pagination.pages || 1));
  if (els.paginationInfo) els.paginationInfo.textContent = paginationSummaryText();
  if (els.firstPage) els.firstPage.disabled = page <= 1;
  if (els.prevPage) els.prevPage.disabled = page <= 1;
  if (els.nextPage) els.nextPage.disabled = page >= pages;
  if (els.lastPage) els.lastPage.disabled = page >= pages;
  if (!els.paginationPages) return;
  els.paginationPages.innerHTML = '';
  paginationItems(page, pages).forEach((item) => {
    if (typeof item !== 'number') {
      const ellipsis = document.createElement('span');
      ellipsis.className = 'pagination-ellipsis';
      ellipsis.textContent = '...';
      els.paginationPages.appendChild(ellipsis);
      return;
    }
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'pagination-btn pagination-page-btn';
    button.textContent = String(item);
    button.dataset.page = String(item);
    button.setAttribute('aria-label', 'Page ' + item);
    if (item === page) {
      button.classList.add('active');
      button.setAttribute('aria-current', 'page');
    }
    els.paginationPages.appendChild(button);
  });
}

function highlightSelectedRow() {
  const selectedID = state.selected ? String(recordPrimaryKey(state.selected.item)) : '';
  els.list.querySelectorAll('tbody tr[data-record-id]').forEach((row) => {
    row.classList.toggle('row-selected', row.dataset.recordId === selectedID);
  });
}

function syncBulkRowSelection() {
  const rows = Array.from(els.list.querySelectorAll('tbody tr[data-record-id]'));
  rows.forEach((row) => {
    const checked = Object.prototype.hasOwnProperty.call(state.bulkSelected, row.dataset.selectionKey);
    row.classList.toggle('row-bulk-selected', checked);
    if (checked) {
      row.dataset.state = 'selected';
    } else {
      delete row.dataset.state;
    }
    const checkbox = row.querySelector('.bulk-row-checkbox');
    if (checkbox) checkbox.checked = checked;
  });
  const selectAll = els.list.querySelector('.bulk-select-all-checkbox');
  if (selectAll) {
    const selectedCount = rows.filter((row) => Object.prototype.hasOwnProperty.call(state.bulkSelected, row.dataset.selectionKey)).length;
    selectAll.checked = rows.length > 0 && selectedCount === rows.length;
    selectAll.indeterminate = selectedCount > 0 && selectedCount < rows.length;
  }
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
  const fields = visibleListFields();
  const rows = data.items || [];
  state.records = rows;
  state.pagination = {
    page: data.page || 1,
    size: data.size || Number(els.pageSize.value || 10),
    pages: data.pages || 1,
    total: data.total || 0
  };
  state.listUpdatedAt = new Date();
  if (els.pageSize) els.pageSize.value = String(state.pagination.size);
  syncListURLState();
  saveCurrentListState();
  renderPagination();
  syncFilterFacetStates();
  syncResetFiltersButton();
  if (!fields.length) {
    renderListEmptyState('No list fields', 'This resource does not expose any list fields for the current user.');
    return;
  }
  if (!rows.length) {
    const hasActiveListState = hasClearableListState();
    const actions = [];
    if (hasActiveListState) {
      actions.push({
        label: 'Clear search and filters',
        onClick: () => {
          clearListFilters();
          syncResetFiltersButton();
          reloadListWithStatus('Cleared list state.', true).catch((error) => setStatus(String(error.message || error)));
        }
      });
    }
    if (!hasActiveListState && hasAction('create')) {
      actions.push({
        label: 'Create record',
        primary: true,
        onClick: () => openModalAndFocus(els.createModal, 'input:not([disabled]), select:not([disabled]), textarea:not([disabled])')
      });
    }
    renderListEmptyState(
      hasActiveListState ? 'No results found' : 'No records yet',
      hasActiveListState
        ? 'Try adjusting the search text or clearing the active filters.'
        : 'Create the first record for this resource when you are ready.',
      actions
    );
    return;
  }
  const listFragment = document.createDocumentFragment();
  const tableShell = document.createElement('div');
  tableShell.className = 'table-shell table-responsive p-0';
  const table = document.createElement('table');
  table.className = 'resource-table table table-bordered table-striped table-hover';
  const thead = document.createElement('thead');
  thead.className = 'thead-light';
  const headRow = document.createElement('tr');
  const bulkCell = document.createElement('th');
  bulkCell.className = 'table-select-cell';
  const selectAll = document.createElement('input');
  selectAll.type = 'checkbox';
  selectAll.className = 'form-check-input position-static bulk-select-all-checkbox';
  const selectedOnPage = rows.filter((row) => isSelectedForBulk(recordPrimaryKey(row))).length;
  selectAll.checked = rows.length > 0 && selectedOnPage === rows.length;
  selectAll.indeterminate = selectedOnPage > 0 && selectedOnPage < rows.length;
  selectAll.onchange = () => {
    rows.forEach((row) => setSelectedForBulk(recordPrimaryKey(row), selectAll.checked));
    syncBulkActionState();
    syncBulkRowSelection();
  };
  bulkCell.appendChild(selectAll);
  headRow.appendChild(bulkCell);
  const sortable = new Set(state.meta?.sort_fields || []);
  const { field: sortField, dir: sortDir } = activeSortField();
  fields.forEach((field) => {
    const th = document.createElement('th');
    const label = fieldLabel(field);
    const labelSpan = document.createElement('span');
    labelSpan.className = 'table-column-label';
    labelSpan.textContent = label;
    const keySpan = document.createElement('span');
    keySpan.className = 'table-column-key';
    keySpan.textContent = field;
    if (sortable.has(field)) {
      th.className = 'sortable-th' + (sortField === field ? ' sort-' + sortDir : '');
      th.setAttribute('aria-sort', sortField === field ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none');
      th.setAttribute('title', 'Click to sort by ' + label);
      th.setAttribute('aria-label', 'Sort by ' + label);
      const iconSpan = document.createElement('span');
      iconSpan.className = 'sort-icon';
      iconSpan.setAttribute('aria-hidden', 'true');
      iconSpan.textContent = sortField === field ? (sortDir === 'asc' ? '▲' : '▼') : '⇅';
      th.appendChild(labelSpan);
      th.appendChild(keySpan);
      th.appendChild(iconSpan);
      th.onclick = () => applySortFromHeader(field);
    } else {
      th.appendChild(labelSpan);
      th.appendChild(keySpan);
    }
    headRow.appendChild(th);
  });
  const actionHead = document.createElement('th');
  actionHead.className = 'table-actions-head';
  actionHead.textContent = 'Actions';
  headRow.appendChild(actionHead);
  thead.appendChild(headRow);
  table.appendChild(thead);
  const tbody = document.createElement('tbody');
  rows.forEach((row) => {
    const tr = document.createElement('tr');
    const id = recordPrimaryKey(row);
    tr.dataset.recordId = String(id);
    tr.dataset.selectionKey = selectionKey(id);
    tr.classList.toggle('row-bulk-selected', isSelectedForBulk(id));
    if (isSelectedForBulk(id)) tr.dataset.state = 'selected';
    const checkCell = document.createElement('td');
    checkCell.className = 'table-select-cell';
    const checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    checkbox.className = 'form-check-input position-static bulk-row-checkbox';
    checkbox.checked = isSelectedForBulk(id);
    checkbox.onchange = () => {
      setSelectedForBulk(id, checkbox.checked);
      syncBulkActionState();
      syncBulkRowSelection();
    };
    checkCell.appendChild(checkbox);
    tr.appendChild(checkCell);
    fields.forEach((field) => {
      const td = document.createElement('td');
      td.appendChild(buildTableCellContent(field, row[field]));
      tr.appendChild(td);
    });
    const actionCell = document.createElement('td');
    actionCell.className = 'table-actions-cell';
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
    trigger.setAttribute('aria-expanded', 'false');
    trigger.textContent = '···';
    trigger.onclick = (e) => {
      e.stopPropagation();
      openActionMenuAt(trigger, id, [
        { label: 'Edit', disabled: !hasAction('update'), onClick: () => selectRecord(row, { openModal: 'edit' }) },
        { divider: true },
        { label: 'Delete', className: 'danger', icon: 'trash', disabled: !hasAction('delete'), onClick: () => deleteRecordByID(id) },
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
  listFragment.appendChild(tableShell);
  els.list.appendChild(listFragment);
  highlightSelectedRow();
  syncBulkRowSelection();
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
       openModalAndFocus(els.recordModal, '#openSelectedEdit:not([disabled]), #copyRecordJSON:not([disabled])');
     }
     if (options.openModal === 'edit') {
       closeModal(els.recordModal);
       openModalAndFocus(els.editModal, 'input:not([disabled]), select:not([disabled]), textarea:not([disabled])');
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
          rewindPageIfCurrentPageEmptied([id]);
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

 function rewindPageIfCurrentPageEmptied(ids) {
  if (state.pagination.page <= 1 || !Array.isArray(ids) || !ids.length || !state.records.length) return;
  const deleted = new Set(ids.map((id) => String(id)));
  const currentPageDeleted = state.records.every((row) => deleted.has(String(recordPrimaryKey(row))));
  if (currentPageDeleted) {
    state.pagination.page -= 1;
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
    renderDashboard();
    setStatus('Loaded ' + state.resources.length + ' resources.');
    if (await restoreNavigationState(window.history.state)) {
      return;
    }
    if (requestedResourceNameFromURL()) {
      showNotFound({ history: 'none' });
      return;
    }
    if (state.resources.length) {
      await selectResource(state.resources[0], { history: 'replace' });
      return;
    }
    updateNavigationState('replace', 'dashboard');
  } catch (error) {
    showErrorPage(error?.status ? error : { status: 500 });
    setStatus(String(error.message || error), 'danger');
  }
}

async function selectResource(resource, options) {
  const navigationMode = options?.history || 'push';
  state.current = resource;
  state.selected = null;
  state.errorPage = null;
  setTopbarBreadcrumb('Dashboard', resource.label || resource.name || 'Resource');
  resetQueryState();
  renderDashboard();
  if (els.errorShell) els.errorShell.hidden = true;
  if (els.workspaceHeader) els.workspaceHeader.hidden = false;
  if (els.recordsShell) els.recordsShell.hidden = false;
  if (navigationMode !== 'none') {
    updateNavigationState(navigationMode, 'resource', resource.name);
  }
  try {
    state.meta = await request(currentBasePath() + '/meta');
    renderResources();
    els.resourceTitle.textContent = state.meta.label;
    setTopbarBreadcrumb('Dashboard', state.meta.label || resource.label || resource.name || 'Resource');
    renderResourceSummary();
    renderResourceActionSummary();
    renderSearchPlaceholder();
    renderSortOptions();
    renderFilterControls();
    initializeColumnVisibility();
    if (options?.restoreQuery !== false) {
      const restoredFromURL = applyListStateFromURL(resource.name);
      if (!restoredFromURL) {
        applySavedListState(resource.name);
      }
    }
    await Promise.all([renderCreateForm(), renderUpdateForm(), renderList()]);
    renderSelectedRecord();
    syncBulkActionState();
    syncWorkspaceActionState();
    setStatus('Loaded resource ' + resource.name + '.');
  } catch (error) {
    const message = String(error.message || error);
    if (error?.status) {
      showErrorPage(error);
    }
    showToast(message, 'danger');
    setStatus(message);
  }
}

els.token.addEventListener('input', () => {
  state.auth = emptyAuthIdentity();
  clearPersistedAuthIdentity();
  persistToken();
  if (!hasToken()) {
    resetAdminState();
  }
  renderAuthState();
});
els.loginForm.onsubmit = async (event) => {
  event.preventDefault();
  setLoginFeedback('');
  try {
    const payload = await request(__GIN_NINJA_ADMIN_AUTH_LOGIN_PATH__, {
      method: 'POST',
      body: JSON.stringify({
        email: els.loginEmail.value.trim(),
        password: els.loginPassword.value
      }),
      skipAuthRedirect: true
    });
    const extractedToken = extractLoginToken(payload);
    if (!payload || !extractedToken) {
      throw new Error('Login response did not include a token.');
    }
    state.auth = {
      ...emptyAuthIdentity(),
      name: extractLoginName(payload) || '',
      userID: extractLoginUserID(payload) || null
    };
    els.token.value = extractedToken;
    persistToken();
    persistAuthIdentity();
    els.loginPassword.value = '';
    renderAuthState();
    const profileLoaded = await refreshAuthProfile();
    if (profileLoaded === false) return;
    const successMessage = state.auth.name ? ('Signed in as ' + state.auth.name + '.') : 'Signed in successfully.';
    if (isStandaloneLoginPage()) {
      redirectToAdmin(successMessage);
      return;
    }
    setStatus(successMessage);
    await loadResources();
  } catch (error) {
    const message = extractErrorMessage(error);
    showToast(message, 'danger');
    setStatus(message, 'danger');
  }
};
const toggleLoginPassword = document.getElementById('toggleLoginPassword');
if (toggleLoginPassword && els.loginPassword) {
  toggleLoginPassword.addEventListener('click', () => {
    const visible = els.loginPassword.type === 'text';
    els.loginPassword.type = visible ? 'password' : 'text';
    toggleLoginPassword.setAttribute('aria-pressed', String(!visible));
    toggleLoginPassword.setAttribute('aria-label', visible ? 'Show password' : 'Hide password');
  });
}
els.clearToken.onclick = () => {
  logout('Signed out of the admin console.');
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

const sidebarUserMenuBtn = document.getElementById('sidebarUserMenuBtn');
const sidebarUserMenu = document.getElementById('sidebarUserMenu');
if (sidebarUserMenuBtn && sidebarUserMenu) {
  const closeSidebarUserMenu = () => {
    sidebarUserMenu.hidden = true;
    sidebarUserMenuBtn.setAttribute('aria-expanded', 'false');
  };
  sidebarUserMenuBtn.addEventListener('click', (event) => {
    event.stopPropagation();
    const open = sidebarUserMenu.hidden === false;
    sidebarUserMenu.hidden = open;
    sidebarUserMenuBtn.setAttribute('aria-expanded', String(!open));
  });
  sidebarUserMenu.addEventListener('click', (event) => {
    event.stopPropagation();
  });
  document.addEventListener('click', closeSidebarUserMenu);
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') closeSidebarUserMenu();
  });
}

// Topbar search toggle
const topbarSearchToggle = document.getElementById('topbarSearchToggle');
const topbarSearchExpand = document.getElementById('topbarSearchExpand');
const topbarSearchClose = document.getElementById('topbarSearchClose');
if (topbarSearchToggle && topbarSearchExpand) {
  topbarSearchToggle.addEventListener('click', (event) => {
    event.stopPropagation();
    if (topbarSearchExpand.classList.contains('open')) {
      closeGlobalSearchDialog({ clearInput: true });
    } else {
      openGlobalSearchDialog();
    }
  });
  topbarSearchExpand.addEventListener('click', (event) => {
    if (event.target === topbarSearchExpand) {
      closeGlobalSearchDialog({ clearInput: true });
    }
  });
}
if (topbarSearchClose) {
  topbarSearchClose.addEventListener('click', (event) => {
    event.stopPropagation();
    closeGlobalSearchDialog({ clearInput: true });
    if (topbarSearchToggle && !topbarSearchToggle.matches('[hidden]')) topbarSearchToggle.focus();
  });
}
document.addEventListener('click', (event) => {
  if (
    topbarSearchExpand &&
    topbarSearchExpand.classList.contains('open') &&
    !topbarSearchExpand.contains(event.target) &&
    event.target !== topbarSearchToggle
  ) {
    closeGlobalSearchDialog({ clearInput: true });
  }
});
document.addEventListener('keydown', (event) => {
  const isSearchShortcut = (event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k';
  if (!isSearchShortcut) return;
  event.preventDefault();
  openGlobalSearchDialog();
});

function errorPageConfig(status, message) {
  const code = Number(status || 500);
  const configs = {
    401: {
      title: 'Unauthorized Access',
      copy: 'Please log in with the appropriate credentials to access this resource.'
    },
    403: {
      title: 'Access Forbidden',
      copy: "You don't have necessary permission to view this resource."
    },
    404: {
      title: 'Oops! Page Not Found!',
      copy: "It seems like the page you're looking for does not exist or might have been removed."
    },
    500: {
      title: "Oops! Something went wrong :')",
      copy: 'We apologize for the inconvenience. Please try again later.'
    },
    503: {
      title: 'Website is under maintenance!',
      copy: "The site is not available at the moment. We'll be back online shortly."
    }
  };
  const known = configs[code];
  const fallback = known || configs[500];
  return {
    code: String(code || 500),
    title: fallback.title,
    copy: known ? fallback.copy : (String(message || '').trim() || fallback.copy)
  };
}

function showErrorPage(error, options) {
  const navigationMode = options?.history || 'none';
  const details = errorPageConfig(error?.status || error?.code || error || 500, options?.copy);
  state.current = null;
  state.meta = null;
  state.selected = null;
  state.errorPage = details;
  setTopbarBreadcrumb('Dashboard', details.code);
  resetQueryState();
  renderResources();
  renderDashboard();
  if (els.workspaceHeader) els.workspaceHeader.hidden = true;
  if (els.recordsShell) els.recordsShell.hidden = true;
  if (els.dashboardShell) els.dashboardShell.hidden = true;
  if (els.errorShell) els.errorShell.hidden = false;
  if (els.errorCode) els.errorCode.textContent = details.code;
  if (els.errorTitle) els.errorTitle.textContent = details.title;
  if (els.errorCopy) els.errorCopy.textContent = details.copy;
  els.resourceTitle.textContent = details.title;
  els.resourcePath.textContent = details.copy;
  els.resourcePath.hidden = false;
  els.detailTitle.textContent = 'No record selected';
  els.detailObjectBadge.textContent = details.code;
  els.detail.textContent = details.copy;
  els.detailFields.innerHTML = '<p class="muted">Use the sidebar or return home to continue.</p>';
  els.createForm.innerHTML = '<p class="muted">No resource is selected.</p>';
  els.updateForm.innerHTML = '<p class="muted">No resource is selected.</p>';
  els.list.innerHTML = '<div class="empty-state">' + escapeHTML(details.copy) + '</div>';
  renderResourceActionSummary();
  updateColumnToggleLabel();
  renderPagination();
  syncBulkActionState();
  syncWorkspaceActionState();
  if (navigationMode !== 'none') {
    updateNavigationState(navigationMode, 'dashboard');
  }
  setStatus(details.code + ' ' + details.title + '.', Number(details.code) >= 500 ? 'danger' : 'warning');
}

function showNotFound(options) {
  showErrorPage({ status: 404 }, options);
}

function showDashboard(options) {
  const navigationMode = options?.history || 'push';
  state.current = null;
  state.meta = null;
  state.selected = null;
  state.errorPage = null;
  setTopbarBreadcrumb('Dashboard', 'Overview');
  resetQueryState();
  renderResources();
  renderDashboard();
  if (els.workspaceHeader) els.workspaceHeader.hidden = true;
  if (els.recordsShell) els.recordsShell.hidden = true;
  if (els.errorShell) els.errorShell.hidden = true;
  els.resourceTitle.textContent = 'Admin dashboard';
  els.resourcePath.textContent = 'Choose a resource from the sidebar to load its workspace.';
  els.resourcePath.hidden = false;
  els.detailTitle.textContent = 'No record selected';
  els.detailObjectBadge.textContent = 'Dashboard';
  els.detail.textContent = 'Choose a resource from the sidebar.';
  els.detailFields.innerHTML = '<p class="muted">Select a resource to inspect records, filters, and actions.</p>';
  els.createForm.innerHTML = '<p class="muted">Select a resource to create records.</p>';
  els.updateForm.innerHTML = '<p class="muted">Select a resource to edit records.</p>';
  els.editHint.textContent = 'Select a resource to open the change form.';
  els.list.innerHTML = '<div class="empty-state">Select a resource from the sidebar to load records.</div>';
  renderResourceActionSummary();
  updateColumnToggleLabel();
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
  els.topbarSearchInput.addEventListener('focus', () => {
    openGlobalSearchDialog();
  });
  els.topbarSearchInput.addEventListener('click', () => {
    openGlobalSearchDialog();
  });
  els.topbarSearchInput.addEventListener('input', (event) => {
    openGlobalSearchDialog();
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
    if (event.key === 'ArrowDown') {
      if (moveGlobalSearchSelection(1)) event.preventDefault();
      return;
    }
    if (event.key === 'ArrowUp') {
      if (moveGlobalSearchSelection(-1)) event.preventDefault();
      return;
    }
    if (event.key === 'Enter') {
      if (openActiveGlobalSearchResult()) event.preventDefault();
      return;
    }
    if (event.key === 'Escape') {
      event.stopPropagation();
      closeGlobalSearchDialog({ clearInput: true });
    }
  });
}
// Topbar ☰ toggle: collapse / expand sidebar
const topbarToggle = document.querySelector('.topbar-toggle');
if (topbarToggle) {
  topbarToggle.addEventListener('click', (event) => {
    event.stopPropagation();
    document.body.classList.toggle('sidebar-collapsed');
    if (document.body.classList.contains('modal-open')) {
      window.requestAnimationFrame(updateModalLayoutMetrics);
    }
  });
}
const sidebarSignOut = document.getElementById('sidebarSignOut');
if (sidebarSignOut) {
  sidebarSignOut.addEventListener('click', () => {
    if (sidebarUserMenu) sidebarUserMenu.hidden = true;
    if (sidebarUserMenuBtn) sidebarUserMenuBtn.setAttribute('aria-expanded', 'false');
    els.clearToken.click();
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
if (els.topbarBreadcrumbRoot) {
  els.topbarBreadcrumbRoot.addEventListener('click', (event) => {
    event.preventDefault();
    showDashboard({ history: state.current ? 'push' : 'replace' });
  });
}
if (els.errorGoBack) {
  els.errorGoBack.addEventListener('click', () => {
    if (window.history && window.history.length > 1) {
      window.history.back();
      return;
    }
    showDashboard({ history: 'replace' });
  });
}
if (els.errorHome) {
  els.errorHome.addEventListener('click', () => {
    showDashboard({ history: 'push' });
  });
}
 els.openCreateModal.onclick = () => {
   if (els.openCreateModal.disabled) return;
   openModalAndFocus(els.createModal, 'input:not([disabled]), select:not([disabled]), textarea:not([disabled])');
 };
 els.closeCreateModal.onclick = () => requestCloseModal(els.createModal);
 els.closeRecordModal.onclick = () => closeRecordModalAndClearSelection();
 els.closeEditModal.onclick = () => requestCloseModal(els.editModal, { afterClose: clearSelectedRecord });
 if (els.openSelectedEdit) {
   els.openSelectedEdit.onclick = () => openSelectedRecordEdit().catch((error) => setStatus(String(error.message || error)));
 }
 if (els.copyRecordJSON) {
   els.copyRecordJSON.onclick = () => copySelectedRecordJSON().catch((error) => {
     showToast(String(error.message || error), 'danger');
     setStatus(String(error.message || error));
   });
 }
 [els.createModal, els.recordModal, els.editModal].forEach((modal) => {
   modal.addEventListener('click', (event) => {
     if (event.target === modal) {
       if (modal === els.recordModal) {
         closeRecordModalAndClearSelection();
       } else if (modal === els.editModal) {
         requestCloseModal(modal, { afterClose: clearSelectedRecord });
       } else {
         requestCloseModal(modal);
       }
    }
  });
 });
 document.addEventListener('keydown', (event) => {
   if (event.key === 'Escape') {
     closeActionMenuPortal();
     closeColumnMenu();
     requestCloseActiveModal();
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
   closeColumnMenu();
 });
window.addEventListener('scroll', scheduleActionMenuPositionUpdate, true);
window.addEventListener('resize', () => {
  scheduleActionMenuPositionUpdate();
  if (document.body.classList.contains('modal-open')) {
    updateModalLayoutMetrics();
  }
});
if (els.bulkActionExport) {
  els.bulkActionExport.onclick = () => exportSelectedRecordsCSV();
}
if (els.clearSelection) {
  els.clearSelection.onclick = () => clearBulkSelection();
}
if (els.bulkActionClear) {
  els.bulkActionClear.onclick = () => clearBulkSelection();
}
if (els.copySelectedIDs) {
  els.copySelectedIDs.onclick = () => copyBulkSelectedIDs().catch((error) => {
    showToast(String(error.message || error), 'danger');
    setStatus(String(error.message || error));
  });
}
 els.filtersForm.onsubmit = (event) => {
   event.preventDefault();
   cancelScheduledSearchReload();
   syncResetFiltersButton();
   reloadListWithStatus('Applied filters.', true).catch((error) => setStatus(String(error.message || error)));
 };
 els.filtersForm.addEventListener('input', () => {
   if (!state.current) return;
   syncResetFiltersButton();
 });
 els.search.addEventListener('input', () => {
   if (!state.current) return;
   syncResetFiltersButton();
   scheduleSearchReload();
 });
 els.search.onkeydown = (event) => {
   if (event.key === 'Enter') {
     event.preventDefault();
     cancelScheduledSearchReload();
     syncResetFiltersButton();
     reloadListWithStatus('Updated search.', true).catch((error) => setStatus(String(error.message || error)));
   }
 };
 els.sort.onchange = () => {
   if (!state.current) return;
   cancelScheduledSearchReload();
   syncResetFiltersButton();
   reloadListWithStatus('Updated sort.', true).catch((error) => setStatus(String(error.message || error)));
 };
els.pageSize.onchange = () => {
  state.pagination.size = Number(els.pageSize.value || 10);
  reloadListWithStatus('Updated page size.', true).catch((error) => setStatus(String(error.message || error)));
};
if (els.columnToggle && els.columnMenu) {
  els.columnToggle.addEventListener('click', (event) => {
    event.stopPropagation();
    const open = els.columnMenu.hidden;
    els.columnMenu.hidden = !open;
    els.columnToggle.setAttribute('aria-expanded', String(open));
  });
  els.columnMenu.addEventListener('click', (event) => {
    event.stopPropagation();
  });
}
 els.filtersForm.onchange = () => {
   if (!state.current) return;
   cancelScheduledSearchReload();
   syncResetFiltersButton();
   reloadListWithStatus('Updated filters.', true).catch((error) => setStatus(String(error.message || error)));
 };
function goToPaginationPage(page, statusMessage) {
  const pages = Math.max(1, Number(state.pagination.pages || 1));
  const nextPage = Math.min(Math.max(1, Number(page || 1)), pages);
  if (nextPage === state.pagination.page) return;
  state.pagination.page = nextPage;
  reloadListWithStatus(statusMessage || ('Loaded page ' + nextPage + '.'), false).catch((error) => setStatus(String(error.message || error)));
}
if (els.firstPage) {
  els.firstPage.onclick = () => goToPaginationPage(1, 'Loaded first page.');
}
if (els.prevPage) {
  els.prevPage.onclick = () => goToPaginationPage(state.pagination.page - 1, 'Loaded previous page.');
}
if (els.paginationPages) {
  els.paginationPages.addEventListener('click', (event) => {
    const button = event.target.closest('button[data-page]');
    if (!button) return;
    goToPaginationPage(Number(button.dataset.page || 1), 'Loaded page ' + button.dataset.page + '.');
  });
}
if (els.nextPage) {
  els.nextPage.onclick = () => goToPaginationPage(state.pagination.page + 1, 'Loaded next page.');
}
if (els.lastPage) {
  els.lastPage.onclick = () => goToPaginationPage(state.pagination.pages, 'Loaded last page.');
}
els.createForm.onsubmit = async (event) => {
  event.preventDefault();
  if (!state.current) return;
  const built = buildValidatedFormPayload(els.createForm, 'create');
  if (!built.valid) {
    showToast('Please fix the highlighted fields.', 'danger');
    setStatus('Please fix the highlighted fields.');
    focusFirstFormError(els.createForm);
    return;
  }
  setFormPending(els.createForm, 'create', true);
  try {
    await request(currentBasePath(), {
      method: 'POST',
      body: JSON.stringify(built.payload)
    });
    await renderCreateForm();
    closeModal(els.createModal);
    showToast('Created a new ' + state.current.name + ' record.', 'success');
    await reloadListWithStatus('Created a new ' + state.current.name + ' record.', true);
  } catch (error) {
    const message = String(error.message || error);
    applyServerErrorToForm(els.createForm, error);
    showToast(message, 'danger');
    setStatus(message);
  } finally {
    setFormPending(els.createForm, 'create', false);
  }
};
els.updateForm.onsubmit = async (event) => {
  event.preventDefault();
  if (!state.current || !state.selected) return;
  const built = buildValidatedFormPayload(els.updateForm, 'update');
  if (!built.valid) {
    showToast('Please fix the highlighted fields.', 'danger');
    setStatus('Please fix the highlighted fields.');
    focusFirstFormError(els.updateForm);
    return;
  }
  setFormPending(els.updateForm, 'update', true);
  try {
    const id = recordPrimaryKey(state.selected.item);
    await request(currentBasePath() + '/' + encodeURIComponent(String(id)), {
      method: 'PUT',
      body: JSON.stringify(built.payload)
    });
    markFormClean(els.updateForm, 'update');
    closeModal(els.editModal);
    await renderList();
    clearSelectedRecord();
    showToast('Updated record #' + id + '.', 'success');
    setStatus('Updated record #' + id + '.');
  } catch (error) {
    const message = String(error.message || error);
    applyServerErrorToForm(els.updateForm, error);
    showToast(message, 'danger');
    setStatus(message);
  } finally {
    setFormPending(els.updateForm, 'update', false);
  }
};
els.createForm.addEventListener('input', () => syncFormDirty(els.createForm, 'create'));
els.createForm.addEventListener('change', () => syncFormDirty(els.createForm, 'create'));
els.updateForm.addEventListener('input', () => syncFormDirty(els.updateForm, 'update'));
els.updateForm.addEventListener('change', () => syncFormDirty(els.updateForm, 'update'));
window.addEventListener('beforeunload', (event) => {
  if (!state.forms.create.dirty && !state.forms.update.dirty) return;
  event.preventDefault();
  event.returnValue = '';
});
if (els.bulkActionDelete) {
  els.bulkActionDelete.onclick = bulkDeleteSelectedRecords;
}
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
if (els.adminThemeToggle && els.adminThemeMenu) {
  els.adminThemeToggle.addEventListener('click', (event) => {
    event.stopPropagation();
    const open = els.adminThemeMenu.hidden;
    els.adminThemeMenu.hidden = !open;
    els.adminThemeToggle.setAttribute('aria-expanded', String(open));
  });
  els.adminThemeMenu.addEventListener('click', (event) => {
    event.stopPropagation();
  });
}
document.addEventListener('click', closeAdminThemeMenu);
document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape') closeAdminThemeMenu();
});

resetAdminState();
updatePageChrome();
renderAdminThemeMenu();
restoreAdminTheme();
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
    refreshAuthProfile()
      .then((profileLoaded) => {
        if (profileLoaded !== false && hasToken()) {
          loadResources().catch((error) => setStatus(String(error.message || error), 'danger'));
        }
      })
      .catch((error) => setStatus(String(error.message || error), 'danger'));
  }
} else {
  if (!flashMessage) {
    setStatus('Ready. Sign in to continue.');
  }
}
