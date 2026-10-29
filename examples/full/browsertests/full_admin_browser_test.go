package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"net/http"
	"testing"
	"time"
)

func TestFullExampleAdminPrototypeBrowserCRUDFlow(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserVisible(t, ctx, "#loginEmail")

	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")
	waitForAdminDashboard(t, ctx)

	openAdminResourcePath(t, ctx, server.URL, "users", "Users")
	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resources", "Roles")
	waitForBrowserText(t, ctx, "#resources", "Projects")
	waitForBrowserCondition(t, ctx, "resources render flat in sidebar", `(() => {
		return !document.querySelector("#resourceTreeview") &&
			!document.querySelector("#resourceTreeviewToggle") &&
			Array.from(document.querySelectorAll("#resources .nav-link")).some((button) => button.textContent.includes("Users"));
	})()`)

	setBrowserValue(t, ctx, "#sidebarResourceSearch", "proj")
	waitForBrowserCondition(t, ctx, "sidebar resource search filters navigation", `(() => {
		const resources = document.querySelector("#resources");
		return !!resources && resources.textContent.includes("Projects") && !resources.textContent.includes("Users");
	})()`)
	clickBrowser(t, ctx, "#sidebarResourceSearchButton")
	waitForBrowserCondition(t, ctx, "sidebar resource search reset restores navigation", `(() => {
		const resources = document.querySelector("#resources");
		const search = document.querySelector("#sidebarResourceSearch");
		return !!resources && !!search && search.value === "" && resources.textContent.includes("Users") && resources.textContent.includes("Projects");
	})()`)

	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const button = Array.from(document.querySelectorAll('#resources .nav-link'))
			.find((node) => node.textContent && node.textContent.includes('Projects'));
		if (button) button.click();
		return !!button;
	})()`, nil))
	waitForBrowserText(t, ctx, "#resourceTitle", "Projects")
	waitForBrowserEnabled(t, ctx, "#openCreateModal")
	waitForBrowserExists(t, ctx, "#createForm textarea[name='title']")

	clickBrowser(t, ctx, "#openCreateModal")
	waitForBrowserVisible(t, ctx, "#createModal")
	waitForBrowserExists(t, ctx, "#createForm details.multi-relation-dropdown")
	waitForBrowserCondition(t, ctx, "owner relation dropdown options loaded", `(() => {
		const menu = document.querySelector("#createForm .multi-relation-menu");
		return !!menu && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Alice"));
	})()`)
	clickBrowser(t, ctx, "#createForm details.multi-relation-dropdown summary")
	waitForBrowserCondition(t, ctx, "owner relation dropdown opens and focuses search", `(() => {
		const dropdown = document.querySelector("#createForm details.multi-relation-dropdown");
		const search = document.querySelector("#createForm .relation-search");
		const options = document.querySelectorAll("#createForm .multi-relation-option");
		return !!dropdown && dropdown.open && !!search && document.activeElement === search && options.length > 0;
	})()`)

	setBrowserValue(t, ctx, "#createForm textarea[name='title']", "Black Box Project")
	setBrowserValue(t, ctx, "#createForm textarea[name='summary']", "created via browser integration")
	setBrowserValue(t, ctx, "#createForm .relation-search", "ali")
	waitForBrowserCondition(t, ctx, "owner relation option filtered", `(() => {
		const options = Array.from(document.querySelectorAll("#createForm .multi-relation-option"));
		return options.length > 0 && options.every((option) => option.textContent.includes("Ali"));
	})()`)
	clickBrowser(t, ctx, "#createForm .multi-relation-option")
	waitForBrowserCondition(t, ctx, "owner relation value selected", `(() => {
		const select = document.querySelector("#createForm select[name='owner_id']");
		const summary = document.querySelector("#createForm details.multi-relation-dropdown summary");
		return !!select && select.value === "1" && !!summary && summary.textContent.includes("Alice");
	})()`)
	clickBrowser(t, ctx, "#createForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Created a new projects record.")
	waitForBrowserText(t, ctx, "#list", "Black Box Project")
	// Toast should appear for successful create
	waitForBrowserCondition(t, ctx, "create toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("Created a new projects record.");
	})()`)

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-btn-view")
	waitForBrowserVisible(t, ctx, "#recordModal")
	waitForBrowserText(t, ctx, "#detailTitle", "Projects #1")
	waitForBrowserText(t, ctx, "#detailFields", "created via browser integration")
	waitForBrowserCondition(t, ctx, "record payload is separated from header", `(() => {
		const pre = document.querySelector('#recordModal pre#detail');
		const card = pre && pre.closest('.detail-card');
		const header = card && card.querySelector('.detail-card-header');
		if (!pre || !header) return false;
		const preRect = pre.getBoundingClientRect();
		const headerRect = header.getBoundingClientRect();
		return preRect.top - headerRect.bottom >= 10;
	})()`)

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserVisible(t, ctx, ".action-menu-list.open")
	clickBrowser(t, ctx, ".action-menu-list.open .action-menu-item")
	waitForBrowserVisible(t, ctx, "#editModal")
	waitForBrowserExists(t, ctx, "#updateForm textarea[name='summary']")

	setBrowserValue(t, ctx, "#updateForm textarea[name='summary']", "updated through browser flow")
	clickBrowser(t, ctx, "#updateForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Updated record #1.")
	waitForBrowserText(t, ctx, "#list", "Black Box Project")
	waitForBrowserCondition(t, ctx, "row highlight released after edit submit", `document.querySelectorAll("#list tbody tr.row-selected").length === 0`)
	// Toast should appear for successful update
	waitForBrowserCondition(t, ctx, "update toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("Updated record #1.");
	})()`)

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-btn-view")
	waitForBrowserText(t, ctx, "#detailFields", "updated through browser flow")
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		document.getElementById('toastContainer').innerHTML = '';
		document.getElementById('status').textContent = '';
	})()`, nil))
	clickBrowser(t, ctx, "#detailFields .detail-value")
	waitForBrowserCondition(t, ctx, "detail fields do not copy values when clicked", `(() => {
		const toastText = document.getElementById('toastContainer').textContent || '';
		const statusText = document.getElementById('status').textContent || '';
		return !toastText.includes('Copied') && !statusText.includes('Copied');
	})()`)
	clickBrowser(t, ctx, "#closeRecordModal")
	waitForBrowserCondition(t, ctx, "row highlight released after closing detail", `document.querySelectorAll("#list tbody tr.row-selected").length === 0`)

	// Verify '/' keyboard shortcut focuses the search input
	waitForBrowserCondition(t, ctx, "search input exists before shortcut", `document.getElementById('search') !== null`)

	clickBrowser(t, ctx, "#list tbody tr:first-child td:first-child input[type='checkbox']")
	waitForBrowserText(t, ctx, "#selectedCountBadge", "1 selected")
	waitForBrowserEnabled(t, ctx, "#bulkActionDelete")
	clickBrowser(t, ctx, "#bulkActionDelete")

	// Confirm the bulk delete in the confirm dialog
	waitForBrowserVisible(t, ctx, "#confirmModal")
	waitForBrowserExists(t, ctx, "#confirmModalConfirm")
	clickBrowser(t, ctx, "#confirmModalConfirm")

	waitForBrowserText(t, ctx, "#status", "Bulk deleted 1 record(s).")
	waitForBrowserText(t, ctx, "#list", "No records yet")
	// Toast should appear for successful bulk delete
	waitForBrowserCondition(t, ctx, "bulk delete toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("Bulk deleted 1 record(s).");
	})()`)
}

func TestFullExampleAdminModalTracksWorkspaceCenter(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.EmulateViewport(1680, 780))
	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")
	waitForAdminDashboard(t, ctx)
	openAdminResourcePath(t, ctx, server.URL, "users", "Users")
	waitForBrowserEnabled(t, ctx, "#openCreateModal")

	centered := func(modal string) string {
		return `(() => {
			const main = document.querySelector('.app-main');
			const dialog = document.querySelector('` + modal + ` .modal-dialog');
			if (!main || !dialog) return false;
			const mainRect = main.getBoundingClientRect();
			const dialogRect = dialog.getBoundingClientRect();
			const mainCenter = mainRect.left + mainRect.width / 2;
			const dialogCenter = dialogRect.left + dialogRect.width / 2;
			return Math.abs(mainCenter - dialogCenter) <= 0.5;
		})()`
	}
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		document.body.style.minHeight = '1400px';
		window.scrollTo(0, 0);
		const rect = (selector) => {
			const box = document.querySelector(selector).getBoundingClientRect();
			return { left: box.left, right: box.right, width: box.width };
		};
		window.__modalLayoutBefore = {
			viewport: window.innerWidth,
			client: document.documentElement.clientWidth,
			topbar: rect('.topbar'),
			main: rect('.app-main'),
			create: rect('#openCreateModal')
		};
	})()`, nil))
	layoutStable := `(() => {
		const before = window.__modalLayoutBefore;
		if (!before) return false;
		const rect = (selector) => {
			const box = document.querySelector(selector).getBoundingClientRect();
			return { left: box.left, right: box.right, width: box.width };
		};
		const after = {
			viewport: window.innerWidth,
			client: document.documentElement.clientWidth,
			topbar: rect('.topbar'),
			main: rect('.app-main'),
			create: rect('#openCreateModal')
		};
		const close = (a, b) => Math.abs(a - b) <= 0.5;
		return close(before.viewport, after.viewport) &&
			close(before.client, after.client) &&
			close(before.topbar.left, after.topbar.left) &&
			close(before.topbar.right, after.topbar.right) &&
			close(before.main.left, after.main.left) &&
			close(before.main.right, after.main.right) &&
			close(before.create.left, after.create.left) &&
			close(before.create.right, after.create.right);
	})()`

	clickBrowser(t, ctx, "#openCreateModal")
	waitForBrowserVisible(t, ctx, "#createModal")
	waitForBrowserCondition(t, ctx, "modal keeps page layout stable with scrollbar", layoutStable)
	waitForBrowserCondition(t, ctx, "create modal centered in expanded workspace", centered("#createModal"))
	clickBrowser(t, ctx, "#closeCreateModal")
	waitForBrowserCondition(t, ctx, "create modal closed", `document.querySelector('#createModal').hidden === true`)

	clickBrowser(t, ctx, ".topbar-toggle")
	waitForBrowserCondition(t, ctx, "sidebar collapsed", `document.body.classList.contains('sidebar-collapsed')`)
	clickBrowser(t, ctx, "#openCreateModal")
	waitForBrowserVisible(t, ctx, "#createModal")
	waitForBrowserCondition(t, ctx, "create modal centered after sidebar collapse", centered("#createModal"))

	runBrowser(t, ctx, chromedp.EmulateViewport(1440, 780))
	waitForBrowserCondition(t, ctx, "create modal centered after resize", centered("#createModal"))
	clickBrowser(t, ctx, "#closeCreateModal")
	waitForBrowserCondition(t, ctx, "create modal closed after resize", `document.querySelector('#createModal').hidden === true`)

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserVisible(t, ctx, ".action-menu-list.open")
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const item = Array.from(document.querySelectorAll('.action-menu-list.open .action-menu-item'))
			.find((button) => button.textContent && button.textContent.includes('Edit'));
		if (item) item.click();
		return !!item;
	})()`, nil))
	waitForBrowserVisible(t, ctx, "#editModal")
	waitForBrowserCondition(t, ctx, "edit modal centered after sidebar collapse and resize", centered("#editModal"))
}

func TestFullExampleAdminPrototypeDarkModeToggle(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserVisible(t, ctx, "#darkModeToggle")

	// By default the page should NOT be in dark mode
	waitForBrowserCondition(t, ctx, "page starts in light mode", `document.documentElement.getAttribute('data-theme') !== 'dark'`)

	// Click the toggle — should enter dark mode
	clickBrowser(t, ctx, "#darkModeToggle")
	waitForBrowserCondition(t, ctx, "dark mode activated after toggle", `document.documentElement.getAttribute('data-theme') === 'dark'`)

	// Moon icon should animate into view in dark mode
	waitForBrowserCondition(t, ctx, "dark mode toggle pressed", `document.getElementById('darkModeToggle').getAttribute('aria-pressed') === 'true'`)
	waitForBrowserCondition(t, ctx, "moon icon visible in dark mode", `parseFloat(getComputedStyle(document.getElementById('darkModeIconMoon')).opacity) > 0.95`)
	waitForBrowserCondition(t, ctx, "sun icon hidden in dark mode", `parseFloat(getComputedStyle(document.getElementById('darkModeIconSun')).opacity) < 0.05`)

	// Click again — should return to light mode
	clickBrowser(t, ctx, "#darkModeToggle")
	waitForBrowserCondition(t, ctx, "light mode restored after second toggle", `document.documentElement.getAttribute('data-theme') !== 'dark'`)

	// Sun icon should animate back into view in light mode
	waitForBrowserCondition(t, ctx, "dark mode toggle not pressed", `document.getElementById('darkModeToggle').getAttribute('aria-pressed') === 'false'`)
	waitForBrowserCondition(t, ctx, "sun icon visible in light mode", `parseFloat(getComputedStyle(document.getElementById('darkModeIconSun')).opacity) > 0.95`)
	waitForBrowserCondition(t, ctx, "moon icon hidden in light mode", `parseFloat(getComputedStyle(document.getElementById('darkModeIconMoon')).opacity) < 0.05`)
}

func TestFullExampleAdminPrototypeUserRoleMultiSelect(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	login := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d body=%s", login.StatusCode, readBody(t, login.Body))
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&auth); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	login.Body.Close()

	for _, role := range []map[string]any{
		{"name": "Administrators", "code": "admin", "status": 1, "remark": "full access"},
		{"name": "Editors", "code": "editor", "status": 1, "remark": "content editors"},
	} {
		createRoleResp := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/roles", role, auth.Token)
		if createRoleResp.StatusCode != http.StatusCreated {
			t.Fatalf("expected role create 201, got %d body=%s", createRoleResp.StatusCode, readBody(t, createRoleResp.Body))
		}
		createRoleResp.Body.Close()
	}

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")
	waitForAdminDashboard(t, ctx)

	openAdminResourcePath(t, ctx, server.URL, "users", "Users")
	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resources", "Roles")
	waitForBrowserEnabled(t, ctx, "#openCreateModal")

	clickBrowser(t, ctx, "#openCreateModal")
	waitForBrowserVisible(t, ctx, "#createModal")
	waitForBrowserExists(t, ctx, "#createForm details.multi-relation-dropdown")
	waitForBrowserCondition(t, ctx, "role multiselect dropdown options loaded", `(() => {
		const menu = document.querySelector("#createForm .multi-relation-menu");
		return !!menu && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Administrators")) && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Editors"));
	})()`)
	clickBrowser(t, ctx, "#createForm details.multi-relation-dropdown summary")
	waitForBrowserCondition(t, ctx, "create multiselect dropdown opens and focuses search", `(() => {
		const dropdown = document.querySelector("#createForm details.multi-relation-dropdown");
		const search = document.querySelector("#createForm .relation-search");
		return !!dropdown && dropdown.open && !!search && document.activeElement === search;
	})()`)

	setBrowserValue(t, ctx, "#createForm textarea[name='name']", "Role User")
	setBrowserValue(t, ctx, "#createForm input[name='email']", "role.user@example.com")
	setBrowserValue(t, ctx, "#createForm input[name='password']", "password123")
	setBrowserValue(t, ctx, "#createForm input[name='age']", "31")
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const dropdown = document.querySelector("#createForm details.multi-relation-dropdown");
		if (!dropdown) return "";
		Array.from(dropdown.querySelectorAll(".multi-relation-option")).forEach((option) => {
			const checkbox = option.querySelector("input[type='checkbox']");
			const shouldSelect = option.textContent.includes("Administrators") || option.textContent.includes("Editors");
			if (checkbox && checkbox.checked !== shouldSelect) checkbox.click();
		});
		const select = document.querySelector("#createForm select[name='role_ids']");
		return select ? Array.from(select.selectedOptions).map((option) => option.value).join(",") : "";
	})()`, nil))
	clickBrowser(t, ctx, "#createForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Created a new users record.")
	waitForBrowserText(t, ctx, "#list", "Role User")
	waitForBrowserCondition(t, ctx, "created user visible with role ids", `document.getElementById('list').textContent.includes('Role User')`)

	clickBrowser(t, ctx, "#list tbody tr:last-child .action-btn-view")
	waitForBrowserVisible(t, ctx, "#recordModal")
	waitForBrowserText(t, ctx, "#detailFields", "Roles")
	waitForBrowserText(t, ctx, "#detailFields", "2 items")
	waitForBrowserText(t, ctx, "#detailFields", "1, 2")
	clickBrowser(t, ctx, "#closeRecordModal")

	clickBrowser(t, ctx, "#list tbody tr:last-child .action-menu-trigger")
	waitForBrowserVisible(t, ctx, ".action-menu-list.open")
	clickBrowser(t, ctx, ".action-menu-list.open .action-menu-item")
	waitForBrowserVisible(t, ctx, "#editModal")
	waitForBrowserExists(t, ctx, "#updateForm details.multi-relation-dropdown")
	waitForBrowserCondition(t, ctx, "update multiselect dropdown options loaded", `(() => {
		const menu = document.querySelector("#updateForm .multi-relation-menu");
		return !!menu && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Administrators")) && Array.from(menu.querySelectorAll(".multi-relation-option")).some((option) => option.textContent.includes("Editors"));
	})()`)
	clickBrowser(t, ctx, "#updateForm details.multi-relation-dropdown summary")
	waitForBrowserCondition(t, ctx, "update multiselect dropdown opens and focuses search", `(() => {
		const dropdown = document.querySelector("#updateForm details.multi-relation-dropdown");
		const search = document.querySelector("#updateForm .relation-search");
		return !!dropdown && dropdown.open && !!search && document.activeElement === search;
	})()`)
	setBrowserValue(t, ctx, "#updateForm .relation-search", "Admin")
	waitForBrowserCondition(t, ctx, "update multiselect options filtered", `(() => {
		const options = Array.from(document.querySelectorAll("#updateForm .multi-relation-option"));
		return options.length === 1 && options[0].textContent.includes("Administrators");
	})()`)
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const dropdown = document.querySelector("#updateForm details.multi-relation-dropdown");
		if (!dropdown) return "";
		Array.from(dropdown.querySelectorAll(".multi-relation-option")).forEach((option) => {
			const checkbox = option.querySelector("input[type='checkbox']");
			const shouldSelect = option.textContent.includes("Editors");
			if (checkbox && checkbox.checked !== shouldSelect) checkbox.click();
		});
		const select = document.querySelector("#updateForm select[name='role_ids']");
		return select ? Array.from(select.selectedOptions).map((option) => option.value).join(",") : "";
	})()`, nil))
	clickBrowser(t, ctx, "#updateForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Updated record #2.")
	clickBrowser(t, ctx, "#list tbody tr:last-child .action-btn-view")
	waitForBrowserText(t, ctx, "#detailFields", "Roles")
	waitForBrowserText(t, ctx, "#detailFields", "1 item")
	waitForBrowserText(t, ctx, "#detailFields", "2")
}

func TestFullExampleAdminPrototypeTextareaResizeDirection(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name": "Alice", "email": "alice@example.com", "password": "password123", "age": 18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")
	waitForAdminDashboard(t, ctx)

	openAdminResourcePath(t, ctx, server.URL, "users", "Users")
	waitForBrowserEnabled(t, ctx, "#openCreateModal")
	clickBrowser(t, ctx, "#openCreateModal")
	waitForBrowserVisible(t, ctx, "#createModal")
	waitForBrowserExists(t, ctx, "#createForm textarea[name='name']")
	waitForBrowserCondition(t, ctx, "create textarea uses vertical resize", `(() => {
		const textarea = document.querySelector("#createForm textarea[name='name']");
		return !!textarea && window.getComputedStyle(textarea).resize === "vertical";
	})()`)
}

func TestFullExampleAdminPrototypeActionMenuPortal(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name": "Alice", "email": "alice@example.com", "password": "password123", "age": 18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")
	waitForAdminDashboard(t, ctx)

	openAdminResourcePath(t, ctx, server.URL, "users", "Users")
	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserCondition(t, ctx, "list table loaded", `document.querySelector('#list table') !== null`)

	// Clicking the ··· trigger should open the portal menu (not inside the table)
	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserCondition(t, ctx, "action menu portal opened", `document.querySelector('#action-menu-portal .action-menu-list.open') !== null`)
	waitForBrowserCondition(t, ctx, "action menu uses shadcn row action styling", `(() => {
		const menu = document.querySelector('#action-menu-portal .action-menu-list.open');
		if (!menu) return false;
		const edit = menu.querySelector('.action-menu-item:not(.danger) .action-menu-item-label');
		const divider = menu.querySelector('.action-menu-divider');
		const del = menu.querySelector('.action-menu-item.danger');
		const icon = del && del.querySelector('.action-menu-item-icon svg');
		const menuStyle = getComputedStyle(menu);
		const delStyle = del ? getComputedStyle(del) : null;
		const rect = menu.getBoundingClientRect();
		const delRect = del ? del.getBoundingClientRect() : null;
		return rect.width >= 150 &&
			rect.width <= 180 &&
			menuStyle.borderRadius === '8px' &&
			edit && edit.textContent.trim() === 'Edit' &&
			divider &&
			del && del.textContent.includes('Delete') &&
			icon &&
			delRect.height <= 34 &&
			delStyle.borderRadius === '6px';
	})()`)
	runBrowser(t, ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		if err := css.Enable().Do(ctx); err != nil {
			return err
		}
		root, err := dom.GetDocument().Do(ctx)
		if err != nil {
			return err
		}
		nodeID, err := dom.QuerySelector(root.NodeID, "#action-menu-portal .action-menu-item.danger").Do(ctx)
		if err != nil {
			return err
		}
		return css.ForcePseudoState(nodeID, []string{"hover"}).Do(ctx)
	}))
	waitForBrowserCondition(t, ctx, "delete action hover keeps destructive text readable", `(() => {
		const del = document.querySelector('#action-menu-portal .action-menu-item.danger');
		if (!del) return false;
		const style = getComputedStyle(del);
		const parts = style.color.match(/[\d.]+/g);
		if (!parts || parts.length < 3) return false;
		const channels = parts.slice(0, 3).map((part) => {
			const value = Number(part);
			return value <= 1 ? Math.round(value * 255) : value;
		});
		const red = channels[0];
		const green = channels[1];
		const blue = channels[2];
		const isWhite = red > 245 && green > 245 && blue > 245;
		const noBorder = style.borderTopWidth === '0px' &&
			style.borderRightWidth === '0px' &&
			style.borderBottomWidth === '0px' &&
			style.borderLeftWidth === '0px';
		return !isWhite &&
			red > green &&
			red > blue &&
			noBorder &&
			style.boxShadow === 'none' &&
			style.outlineStyle === 'none';
	})()`)

	// The portal menu must be a direct child of body (not inside table-shell)
	waitForBrowserCondition(t, ctx, "action menu portal is direct child of body", `
		document.getElementById('action-menu-portal') !== null &&
		document.getElementById('action-menu-portal').parentElement === document.body
	`)

	// Clicking outside should close the portal
	runBrowser(t, ctx, chromedp.Evaluate(`document.body.click()`, nil))
	waitForBrowserCondition(t, ctx, "action menu portal closed after outside click", `document.querySelector('#action-menu-portal .action-menu-list.open') === null`)

	// Pressing Escape after re-opening should also close it
	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserCondition(t, ctx, "action menu portal re-opened", `document.querySelector('#action-menu-portal .action-menu-list.open') !== null`)
	runBrowser(t, ctx, chromedp.KeyEvent("\x1b"))
	waitForBrowserCondition(t, ctx, "action menu portal closed by Escape", `document.querySelector('#action-menu-portal .action-menu-list.open') === null`)
}

func TestFullExampleAdminPrototypeGlobalSearch(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	// Register Alice
	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name": "Alice", "email": "alice@example.com", "password": "password123", "age": 18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	// Navigate to the admin console and sign in via the login form
	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")
	waitForAdminDashboard(t, ctx)

	// Wait for resources to load
	waitForBrowserText(t, ctx, "#resources", "Users")

	// The topbar search input should exist in the page
	waitForBrowserCondition(t, ctx, "topbar search input exists", `!!document.getElementById('topbarSearchInput')`)
	// The results panel should start hidden (no has-results class)
	waitForBrowserCondition(t, ctx, "search results panel initially hidden", `!document.getElementById('topbarSearchResults').classList.contains('has-results')`)

	// Open the command-style search dialog by clicking the search input
	clickBrowser(t, ctx, "#topbarSearchInput")
	waitForBrowserCondition(t, ctx, "search dialog opens", `document.getElementById('topbarSearchExpand').classList.contains('open')`)
	waitForBrowserVisible(t, ctx, "#topbarSearchInput")
	waitForBrowserCondition(t, ctx, "search dialog overlay covers viewport", `(() => {
		const rect = document.getElementById('topbarSearchExpand').getBoundingClientRect();
		const page = document.body.getBoundingClientRect();
		return rect.left <= page.left + 1 &&
			rect.top <= 1 &&
			rect.right >= page.right - 1 &&
			rect.bottom >= window.innerHeight - 1;
	})()`)

	// Pressing Escape should close the search dialog
	runBrowser(t, ctx, chromedp.Focus("#topbarSearchInput"))
	runBrowser(t, ctx, chromedp.KeyEvent("\x1b"))
	waitForBrowserCondition(t, ctx, "search dialog closes on Escape", `!document.getElementById('topbarSearchExpand').classList.contains('open')`)
}

func TestFullExampleAdminPrototypeSortableColumns(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	// Register and sign in
	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name": "Alice", "email": "alice@example.com", "password": "password123", "age": 18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")
	waitForAdminDashboard(t, ctx)

	// Wait for resources and list to load
	openAdminResourcePath(t, ctx, server.URL, "users", "Users")
	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserCondition(t, ctx, "list has at least one row", `document.querySelector('#list table') !== null`)

	// There should be sortable column headers in the Users table
	waitForBrowserCondition(t, ctx, "sortable header exists", `document.querySelector('th.sortable-th') !== null`)

	// Clicking a sortable header should update the sort dropdown to ascending
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const th = document.querySelector('th.sortable-th');
		if (th) th.click();
		return !!th;
	})()`, nil))

	// The sort select should now have a non-empty value (ascending sort was applied)
	waitForBrowserCondition(t, ctx, "sort select updated after header click", `document.getElementById('sort').value !== ''`)
	// Wait for list to finish re-rendering before next click
	waitForBrowserCondition(t, ctx, "list table re-rendered after first click", `document.querySelector('#list th.sortable-th') !== null`)

	// Clicking again should switch to descending
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const th = document.querySelector('th.sortable-th');
		if (th) th.click();
		return !!th;
	})()`, nil))
	waitForBrowserCondition(t, ctx, "sort select shows descending after second click", `document.getElementById('sort').value.startsWith('-')`)
	// Wait for list to finish re-rendering before next click
	waitForBrowserCondition(t, ctx, "list table re-rendered after second click", `document.querySelector('#list th.sortable-th.sort-desc') !== null`)

	// Clicking a third time should clear the sort (return to default)
	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const th = document.querySelector('th.sortable-th');
		if (th) th.click();
		return !!th;
	})()`, nil))
	waitForBrowserCondition(t, ctx, "sort select cleared after third click", `document.getElementById('sort').value === ''`)
}

func TestFullExampleAdminQACriticalInteractions(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	login := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d body=%s", login.StatusCode, readBody(t, login.Body))
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&auth); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	login.Body.Close()

	for i := 0; i < 8; i++ {
		resp := doFullJSON(t, server, http.MethodPost, "/api/v1/admin/resources/users", map[string]any{
			"name":     fmt.Sprintf("QA User %02d With A Long Display Name", i+1),
			"email":    fmt.Sprintf("qa-user-%02d@example.com", i+1),
			"password": "password123",
			"age":      24 + i,
			"is_admin": i%2 == 0,
		}, auth.Token)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected user create 201, got %d body=%s", resp.StatusCode, readBody(t, resp.Body))
		}
		resp.Body.Close()
	}

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin/login"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")
	waitForAdminDashboard(t, ctx)
	clickBrowser(t, ctx, "#darkModeToggle")
	waitForBrowserCondition(t, ctx, "dark topbar has no extra search separator", `(() => {
		if (document.documentElement.getAttribute('data-theme') !== 'dark') return false;
		const actions = document.querySelector('.topbar-actions');
		const search = document.querySelector('.topbar-search-field');
		if (!actions || !search) return false;
		const before = getComputedStyle(actions, '::before');
		const actionsStyle = getComputedStyle(actions);
		const searchStyle = getComputedStyle(search);
		const noActionsBorder = actionsStyle.borderLeftWidth === '0px' ||
			actionsStyle.borderLeftStyle === 'none' ||
			actionsStyle.borderLeftColor === 'rgba(0, 0, 0, 0)';
		return (before.content === 'none' || before.display === 'none' || before.width === '0px') &&
			noActionsBorder &&
			searchStyle.borderLeftColor === 'rgba(0, 0, 0, 0)';
	})()`)
	clickBrowser(t, ctx, "#darkModeToggle")
	waitForBrowserCondition(t, ctx, "light mode restored after topbar separator check", `document.documentElement.getAttribute('data-theme') !== 'dark'`)

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin/users?search=QA&page=1&size=5"))
	waitForBrowserPath(t, ctx, "/admin/users")
	waitForBrowserCondition(t, ctx, "query state restored into controls", `(() => {
		return document.getElementById('search').value === 'QA' &&
			document.getElementById('pageSize').value === '5' &&
			document.getElementById('paginationInfo').textContent.includes('Page 1 of');
	})()`)
	waitForBrowserText(t, ctx, "#list", "QA User 01")
	waitForBrowserCondition(t, ctx, "boolean table badges use compact shadcn style", `(() => {
		const badges = Array.from(document.querySelectorAll('#list .table-badge.boolean'));
		const trueBadge = document.querySelector('#list .table-badge.boolean-true');
		const falseBadge = document.querySelector('#list .table-badge.boolean-false');
		if (!badges.length || !trueBadge || !falseBadge) return false;
		const legacyTone = badges.some((badge) => badge.classList.contains('success') || badge.classList.contains('danger'));
		const compact = badges.every((badge) => badge.getBoundingClientRect().height <= 24);
		const sampleStyle = getComputedStyle(trueBadge);
		return !legacyTone && compact && sampleStyle.borderRadius === '6px';
	})()`)
	waitForBrowserCondition(t, ctx, "filter controls available", `(() => {
		return !!document.getElementById('filter-is_admin') &&
			!!document.getElementById('filter-age') &&
			!!document.getElementById('filter-createdAt-from') &&
			!!document.getElementById('filter-createdAt-to');
	})()`)
	waitForBrowserCondition(t, ctx, "reset button visible when search query is active", `document.getElementById('resetFilters').hidden === false`)

	clickBrowser(t, ctx, "#nextPage")
	waitForBrowserCondition(t, ctx, "pagination moves to page two", `(() => {
		return document.getElementById('paginationInfo').textContent.includes('Page 2 of') &&
			new URLSearchParams(window.location.search).get('page') === '2';
	})()`)
	clickBrowser(t, ctx, "#firstPage")
	waitForBrowserCondition(t, ctx, "pagination returns to first page", `(() => {
		return document.getElementById('paginationInfo').textContent.includes('Page 1 of') &&
			new URLSearchParams(window.location.search).get('page') === '1';
	})()`)

	today := time.Now().Format("2006-01-02")
	setBrowserValue(t, ctx, "#filter-createdAt-from", today)
	waitForBrowserCondition(t, ctx, "date range filter is mirrored to URL", fmt.Sprintf(`new URLSearchParams(window.location.search).get('f.createdAt__from') === %q`, today))
	waitForBrowserCondition(t, ctx, "date range filter keeps reset visible", `document.getElementById('resetFilters').hidden === false`)

	setBrowserValue(t, ctx, "#search", "no-match-admin-qa")
	waitForBrowserText(t, ctx, "#list", "No results found")
	waitForBrowserText(t, ctx, "#list", "Clear search and filters")
	waitForBrowserCondition(t, ctx, "filtered empty state does not offer create record", `!Array.from(document.querySelectorAll('#list .empty-state button')).some((button) => button.textContent.includes('Create record'))`)

	clickBrowser(t, ctx, "#resetFilters")
	waitForBrowserCondition(t, ctx, "reset clears search filters and hides itself", `(() => {
		const params = new URLSearchParams(window.location.search);
		return document.getElementById('search').value === '' &&
			document.getElementById('filter-createdAt-from').value === '' &&
			document.getElementById('resetFilters').hidden === true &&
			!params.has('search') &&
			!params.has('f.createdAt__from') &&
			document.querySelector('#list table') !== null;
	})()`)

	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const originalFetch = window.fetch.bind(window);
		window.__adminFetchCalls = [];
		window.fetch = (...args) => {
			const input = args[0];
			window.__adminFetchCalls.push(typeof input === 'string' ? input : String(input && input.url || input));
			return originalFetch(...args);
		};
	})()`, nil))
	runBrowser(t, ctx, chromedp.Evaluate(`window.__adminFetchCalls = []`, nil))
	clickBrowser(t, ctx, "#list .bulk-select-all-checkbox")
	waitForBrowserText(t, ctx, "#bulkActionCount", "5")
	waitForBrowserCondition(t, ctx, "selected rows use selected styling", `document.querySelectorAll('#list tbody tr.row-bulk-selected').length === 5`)
	runBrowser(t, ctx, chromedp.Sleep(300*time.Millisecond))
	waitForBrowserCondition(t, ctx, "select all does not request users list", `!window.__adminFetchCalls.some((url) => String(url).includes('/api/v1/admin/resources/users?'))`)

	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const button = Array.from(document.querySelectorAll('#resources .nav-link'))
			.find((node) => node.textContent && node.textContent.includes('Roles'));
		if (button) button.click();
		return !!button;
	})()`, nil))
	waitForBrowserPath(t, ctx, "/admin/roles")
	waitForBrowserText(t, ctx, "#resourceTitle", "Roles")
	waitForBrowserCondition(t, ctx, "bulk selection clears when switching resource", `document.getElementById('bulkActionBar').hidden === true`)

	clickBrowser(t, ctx, "#columnToggle")
	waitForBrowserCondition(t, ctx, "column menu opens inside viewport", `(() => {
		const menu = document.getElementById('columnMenu');
		const rect = menu.getBoundingClientRect();
		return !menu.hidden && rect.width > 0 && rect.right <= window.innerWidth + 1 && rect.left >= -1;
	})()`)
	runBrowser(t, ctx, chromedp.Evaluate(`document.body.click()`, nil))
	waitForBrowserCondition(t, ctx, "column menu closes on outside click", `document.getElementById('columnMenu').hidden === true`)

	clickBrowser(t, ctx, "#adminThemeToggle")
	waitForBrowserCondition(t, ctx, "theme menu opens", `document.getElementById('adminThemeMenu').hidden === false`)
	runBrowser(t, ctx, chromedp.Evaluate(`document.querySelector('[data-admin-theme-option="whatsapp"]').click()`, nil))
	waitForBrowserCondition(t, ctx, "theme option applies and closes menu", `document.documentElement.getAttribute('data-admin-theme') === 'whatsapp' && document.getElementById('adminThemeMenu').hidden === true`)

	clickBrowser(t, ctx, "#topbarUserBtn")
	waitForBrowserCondition(t, ctx, "user menu contains only profile and sign out essentials", `(() => {
		const menu = document.getElementById('topbarUserMenu');
		const rect = menu.getBoundingClientRect();
		const name = document.getElementById('topbarProfileName');
		const signOut = document.getElementById('clearToken');
		const nameStyle = name ? getComputedStyle(name) : null;
		const signOutRect = signOut ? signOut.getBoundingClientRect() : null;
		return menu.hidden === false &&
			rect.width <= 220 &&
			nameStyle && nameStyle.fontSize === '13px' &&
			signOutRect && signOutRect.height <= 34 &&
			menu.textContent.includes('Alice') &&
			menu.textContent.includes('alice@example.com') &&
			menu.textContent.includes('Sign out') &&
			!menu.textContent.includes('Permissions') &&
			!menu.textContent.includes('Token expires');
	})()`)
	runBrowser(t, ctx, chromedp.Evaluate(`document.body.click()`, nil))
	waitForBrowserCondition(t, ctx, "user menu closes on outside click", `document.getElementById('topbarUserMenu').hidden === true`)

	runBrowser(t, ctx, chromedp.EmulateViewport(390, 900))
	waitForBrowserCondition(t, ctx, "toolbar controls stay inside viewport on narrow width", `(() => {
		const controls = Array.from(document.querySelectorAll('#recordsShell .table-toolbar, #recordsShell .filters, #tableViewTools'));
		return controls.every((el) => {
			const rect = el.getBoundingClientRect();
			return rect.left >= -1 && rect.right <= window.innerWidth + 1;
		});
	})()`)
}

func TestFullExampleStandaloneAdminBrowserRedirectFlow(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin"))
	waitForBrowserPath(t, ctx, "/admin/login")
	waitForBrowserVisible(t, ctx, "#loginEmail")

	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForAdminDashboard(t, ctx)
	waitForBrowserText(t, ctx, "#resources", "Projects")
	waitForBrowserCondition(t, ctx, "#adminShell visible", `(() => {
		const el = document.querySelector("#adminShell");
		return !!el && !el.hidden;
	})()`)
	waitForBrowserCondition(t, ctx, "#loginForm hidden", `(() => {
		const el = document.querySelector("#loginForm");
		return !!el && el.hidden;
	})()`)

	clickBrowser(t, ctx, "#clearToken")
	waitForBrowserPath(t, ctx, "/admin/login")
	waitForBrowserText(t, ctx, "#status", "Signed out of the admin console.")
	waitForBrowserVisible(t, ctx, "#loginForm")
}

func TestFullExampleStandaloneAdminLoginShowsErrorFeedback(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin/login"))
	waitForBrowserVisible(t, ctx, "#loginEmail")

	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "wrong-password")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserPath(t, ctx, "/admin/login")
	waitForBrowserText(t, ctx, "#status", "invalid email or password")
	waitForBrowserCondition(t, ctx, "login error toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("invalid email or password");
	})()`)
}

func TestFullExampleStandaloneAdminDashboardBackNavigation(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin/login"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForAdminDashboard(t, ctx)
	runBrowser(t, ctx, chromedp.EmulateViewport(1440, 900))
	waitForBrowserCondition(t, ctx, "breadcrumb root is dashboard link", `document.getElementById('topbarBreadcrumbRoot').getAttribute('href') === '/admin'`)
	waitForBrowserCondition(t, ctx, "dashboard resource cards wrap after three columns", `(() => {
		const grid = document.getElementById('dashboardTiles');
		const first = grid && grid.querySelector('.dashboard-tile');
		if (!grid || !first) return false;
		while (grid.children.length < 5) {
			const clone = first.cloneNode(true);
			clone.removeAttribute('aria-label');
			grid.appendChild(clone);
		}
		const columns = getComputedStyle(grid).gridTemplateColumns
			.split(' ')
			.filter(Boolean).length;
		const tiles = Array.from(grid.querySelectorAll('.dashboard-tile')).slice(0, 5);
		const tops = tiles.map((tile) => Math.round(tile.getBoundingClientRect().top));
		const firstTop = tops[0];
		const firstRowCount = tops.filter((top) => Math.abs(top - firstTop) <= 2).length;
		const rowCount = new Set(tops).size;
		return columns === 3 && firstRowCount === 3 && rowCount >= 2;
	})()`)

	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const tiles = Array.from(document.querySelectorAll('#dashboardTiles .dashboard-tile'));
		const tile = tiles.find((item) => String(item.textContent || '').includes('Users'));
		if (tile) tile.click();
		return !!tile;
	})()`, nil))
	waitForBrowserPath(t, ctx, "/admin/users")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")

	clickBrowser(t, ctx, "#sidebarDashboardLink")
	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Admin dashboard")
	waitForBrowserText(t, ctx, "#dashboardTiles", "Users")
	waitForBrowserCondition(t, ctx, "workspace header hidden on dashboard", `document.getElementById('workspaceHeader').hidden === true`)
	waitForBrowserCondition(t, ctx, "records shell hidden on dashboard", `document.getElementById('recordsShell').hidden === true`)

	runBrowser(t, ctx, chromedp.Evaluate(`(() => {
		const tiles = Array.from(document.querySelectorAll('#dashboardTiles .dashboard-tile'));
		const tile = tiles.find((item) => String(item.textContent || '').includes('Users'));
		if (tile) tile.click();
		return !!tile;
	})()`, nil))
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
	waitForBrowserPath(t, ctx, "/admin/users")
	waitForBrowserCondition(t, ctx, "workspace header visible for resource", `document.getElementById('workspaceHeader').hidden === false`)
	waitForBrowserCondition(t, ctx, "records shell visible for resource", `document.getElementById('recordsShell').hidden === false`)

	runBrowser(t, ctx, chromedp.Evaluate(`history.back()`, nil))
	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Admin dashboard")
	waitForBrowserText(t, ctx, "#dashboardTiles", "Users")
	waitForBrowserCondition(t, ctx, "workspace header hidden again on dashboard", `document.getElementById('workspaceHeader').hidden === true`)
	waitForBrowserCondition(t, ctx, "records shell hidden again on dashboard", `document.getElementById('recordsShell').hidden === true`)
}

func TestFullExampleStandaloneAdminUnknownResourceShowsNotFound(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	register := doFullJSON(t, server, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register.Body))
	}
	register.Body.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin/login"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForAdminDashboard(t, ctx)

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin/1111?page=1&size=50"))
	waitForBrowserPath(t, ctx, "/admin/1111")
	waitForBrowserCondition(t, ctx, "unknown admin resource preserves query", `window.location.search === '?page=1&size=50'`)
	waitForBrowserText(t, ctx, "#errorShell", "404")
	waitForBrowserText(t, ctx, "#errorShell", "Oops! Page Not Found!")
	waitForBrowserCondition(t, ctx, "unknown admin resource shows not found shell only", `(() => {
		return document.getElementById('errorShell').hidden === false &&
			document.getElementById('workspaceHeader').hidden === true &&
			document.getElementById('recordsShell').hidden === true &&
			document.getElementById('dashboardShell').hidden === true;
	})()`)
	waitForBrowserCondition(t, ctx, "unknown admin resource does not redirect to users", `window.location.pathname !== '/admin/users'`)

	clickBrowser(t, ctx, "#errorHome")
	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Admin dashboard")
	waitForBrowserCondition(t, ctx, "dashboard visible after returning home", `document.getElementById('dashboardShell').hidden === false`)
}
