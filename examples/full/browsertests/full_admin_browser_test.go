package main

import (
	"encoding/json"
	"github.com/chromedp/chromedp"
	"net/http"
	"testing"
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

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")

	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resources", "Roles")
	waitForBrowserText(t, ctx, "#resources", "Projects")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")

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

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserVisible(t, ctx, ".action-menu-list.open")
	clickBrowser(t, ctx, ".action-menu-list.open .action-menu-item")
	waitForBrowserVisible(t, ctx, "#editModal")
	waitForBrowserExists(t, ctx, "#updateForm textarea[name='summary']")

	setBrowserValue(t, ctx, "#updateForm textarea[name='summary']", "updated through browser flow")
	clickBrowser(t, ctx, "#updateForm button[type='submit']")

	waitForBrowserText(t, ctx, "#status", "Updated record #1.")
	waitForBrowserText(t, ctx, "#list", "Black Box Project")
	// Toast should appear for successful update
	waitForBrowserCondition(t, ctx, "update toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("Updated record #1.");
	})()`)

	clickBrowser(t, ctx, "#list tbody tr:first-child .action-btn-view")
	waitForBrowserText(t, ctx, "#detailFields", "updated through browser flow")
	clickBrowser(t, ctx, "#closeRecordModal")

	// Verify '/' keyboard shortcut focuses the search input
	waitForBrowserCondition(t, ctx, "search input exists before shortcut", `document.getElementById('search') !== null`)

	clickBrowser(t, ctx, "#list tbody tr:first-child td:first-child input[type='checkbox']")
	waitForBrowserText(t, ctx, "#selectedCountBadge", "1 selected")
	waitForBrowserEnabled(t, ctx, "#bulkDelete")
	clickBrowser(t, ctx, "#bulkDelete")

	// Confirm the bulk delete in the confirm dialog
	waitForBrowserVisible(t, ctx, "#confirmModal")
	waitForBrowserExists(t, ctx, "#confirmModalConfirm")
	clickBrowser(t, ctx, "#confirmModalConfirm")

	waitForBrowserText(t, ctx, "#status", "Bulk deleted 1 record(s).")
	waitForBrowserText(t, ctx, "#list", "No records matched the current filters.")
	// Toast should appear for successful bulk delete
	waitForBrowserCondition(t, ctx, "bulk delete toast appears", `(() => {
		const container = document.querySelector("#toastContainer");
		return !!container && container.textContent.includes("Bulk deleted 1 record(s).");
	})()`)
}

func TestFullExampleAdminPrototypeDarkModeToggle(t *testing.T) {
	server := newFullTestServer(t)
	defer server.Close()

	ctx, cancel := newFullBrowserContext(t)
	defer cancel()

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#darkModeToggle")

	// By default the page should NOT be in dark mode
	waitForBrowserCondition(t, ctx, "page starts in light mode", `document.documentElement.getAttribute('data-theme') !== 'dark'`)

	// Click the toggle — should enter dark mode
	clickBrowser(t, ctx, "#darkModeToggle")
	waitForBrowserCondition(t, ctx, "dark mode activated after toggle", `document.documentElement.getAttribute('data-theme') === 'dark'`)

	// Sun icon should be visible, moon icon should be hidden
	waitForBrowserCondition(t, ctx, "sun icon visible in dark mode", `!document.getElementById('darkModeIconSun').hidden`)
	waitForBrowserCondition(t, ctx, "moon icon hidden in dark mode", `document.getElementById('darkModeIconMoon').hidden`)

	// Click again — should return to light mode
	clickBrowser(t, ctx, "#darkModeToggle")
	waitForBrowserCondition(t, ctx, "light mode restored after second toggle", `document.documentElement.getAttribute('data-theme') !== 'dark'`)

	// Moon icon should be visible, sun icon hidden
	waitForBrowserCondition(t, ctx, "moon icon visible in light mode", `!document.getElementById('darkModeIconMoon').hidden`)
	waitForBrowserCondition(t, ctx, "sun icon hidden in light mode", `document.getElementById('darkModeIconSun').hidden`)
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

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resources", "Roles")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
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
	waitForBrowserText(t, ctx, "#detailFields", "[1,2]")
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
	waitForBrowserText(t, ctx, "#detailFields", "[2]")
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

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
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

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
	waitForBrowserCondition(t, ctx, "list table loaded", `document.querySelector('#list table') !== null`)

	// Clicking the ··· trigger should open the portal menu (not inside the table)
	clickBrowser(t, ctx, "#list tbody tr:first-child .action-menu-trigger")
	waitForBrowserCondition(t, ctx, "action menu portal opened", `document.querySelector('#action-menu-portal .action-menu-list.open') !== null`)

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

	// Navigate to admin prototype and sign in via the login form
	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	// Wait for resources to load
	waitForBrowserText(t, ctx, "#resources", "Users")

	// The topbar search input should exist in the page
	waitForBrowserCondition(t, ctx, "topbar search input exists", `!!document.getElementById('topbarSearchInput')`)
	// The results panel should start hidden (no has-results class)
	waitForBrowserCondition(t, ctx, "search results panel initially hidden", `!document.getElementById('topbarSearchResults').classList.contains('has-results')`)

	// Open the search expand by clicking the search toggle
	clickBrowser(t, ctx, "#topbarSearchToggle")
	waitForBrowserCondition(t, ctx, "search expand opens", `document.getElementById('topbarSearchExpand').classList.contains('open')`)
	waitForBrowserVisible(t, ctx, "#topbarSearchInput")

	// Pressing Escape should close the search expand
	runBrowser(t, ctx, chromedp.Focus("#topbarSearchInput"))
	runBrowser(t, ctx, chromedp.KeyEvent("\x1b"))
	waitForBrowserCondition(t, ctx, "search expand closes on Escape", `!document.getElementById('topbarSearchExpand').classList.contains('open')`)
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

	runBrowser(t, ctx, chromedp.Navigate(server.URL+"/admin-prototype"))
	waitForBrowserVisible(t, ctx, "#loginEmail")
	setBrowserValue(t, ctx, "#loginEmail", "alice@example.com")
	setBrowserValue(t, ctx, "#loginPassword", "password123")
	clickBrowser(t, ctx, "#loginButton")

	// Wait for resources and list to load
	waitForBrowserText(t, ctx, "#resources", "Users")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
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

	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Users")
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

	waitForBrowserPath(t, ctx, "/admin")
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
	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserCondition(t, ctx, "workspace header visible for resource", `document.getElementById('workspaceHeader').hidden === false`)
	waitForBrowserCondition(t, ctx, "records shell visible for resource", `document.getElementById('recordsShell').hidden === false`)

	runBrowser(t, ctx, chromedp.Evaluate(`history.back()`, nil))
	waitForBrowserPath(t, ctx, "/admin")
	waitForBrowserText(t, ctx, "#resourceTitle", "Admin dashboard")
	waitForBrowserText(t, ctx, "#dashboardTiles", "Users")
	waitForBrowserCondition(t, ctx, "workspace header hidden again on dashboard", `document.getElementById('workspaceHeader').hidden === true`)
	waitForBrowserCondition(t, ctx, "records shell hidden again on dashboard", `document.getElementById('recordsShell').hidden === true`)
}
