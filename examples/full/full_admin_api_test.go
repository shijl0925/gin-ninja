package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestFullExampleAdminAPIUsersAndProjectPermissions(t *testing.T) {
	client := newFullTestClient(t)

	for _, user := range []map[string]any{
		{
			"name":     "Alice",
			"email":    "alice@example.com",
			"password": "password123",
			"age":      18,
		},
		{
			"name":     "Bob",
			"email":    "bob@example.com",
			"password": "password123",
			"age":      22,
		},
	} {
		register := doFullJSON(t, client, http.MethodPost, "/api/v1/auth/register", user, "")
		if register.StatusCode != http.StatusCreated {
			t.Fatalf("expected register 201, got %d body=%s", register.StatusCode, readBody(t, register))
		}
	}

	login := func(email string) string {
		t.Helper()
		resp := doFullJSON(t, client, http.MethodPost, "/api/v1/auth/login", map[string]any{
			"email":    email,
			"password": "password123",
		}, "")
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected login 201 for %s, got %d body=%s", email, resp.StatusCode, readBody(t, resp))
		}
		var auth struct {
			Token string `json:"token"`
		}
		if err := resp.DecodeJSON(&auth); err != nil {
			t.Fatalf("decode login for %s: %v", email, err)
		}
		if auth.Token == "" {
			t.Fatalf("expected login token for %s", email)
		}
		return auth.Token
	}

	aliceToken := login("alice@example.com")
	bobToken := login("bob@example.com")

	for _, role := range []map[string]any{
		{
			"name":   "Administrators",
			"code":   "admin",
			"status": 1,
			"remark": "full access",
		},
		{
			"name":   "Editors",
			"code":   "editor",
			"status": 1,
			"remark": "content editors",
		},
	} {
		createRoleResp := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/roles", role, aliceToken)
		if createRoleResp.StatusCode != http.StatusCreated {
			t.Fatalf("expected role create 201, got %d body=%s", createRoleResp.StatusCode, readBody(t, createRoleResp))
		}
	}

	resourceIndexResp := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources", nil, aliceToken)
	if resourceIndexResp.StatusCode != http.StatusOK {
		t.Fatalf("expected admin resource index 200, got %d", resourceIndexResp.StatusCode)
	}
	var resourceIndex struct {
		Resources []struct {
			Name  string `json:"name"`
			Label string `json:"label"`
			Path  string `json:"path"`
		} `json:"resources"`
	}
	if err := resourceIndexResp.DecodeJSON(&resourceIndex); err != nil {
		t.Fatalf("decode resource index: %v", err)
	}
	if len(resourceIndex.Resources) != 3 {
		t.Fatalf("expected 3 admin resources, got %+v", resourceIndex.Resources)
	}
	resourcePaths := map[string]string{}
	for _, resource := range resourceIndex.Resources {
		resourcePaths[resource.Name] = resource.Path
	}
	if resourcePaths["users"] != "/users" || resourcePaths["roles"] != "/roles" || resourcePaths["projects"] != "/projects" {
		t.Fatalf("unexpected admin resources: %+v", resourceIndex.Resources)
	}

	usersMetaResp := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/users/meta", nil, aliceToken)
	if usersMetaResp.StatusCode != http.StatusOK {
		t.Fatalf("expected users metadata 200, got %d", usersMetaResp.StatusCode)
	}
	var usersMeta struct {
		Actions      []string `json:"actions"`
		CreateFields []string `json:"create_fields"`
		UpdateFields []string `json:"update_fields"`
		Fields       []struct {
			Name      string `json:"name"`
			Type      string `json:"type"`
			Component string `json:"component"`
			Relation  *struct {
				Resource   string `json:"resource"`
				LabelField string `json:"label_field"`
			} `json:"relation"`
		} `json:"fields"`
	}
	if err := usersMetaResp.DecodeJSON(&usersMeta); err != nil {
		t.Fatalf("decode users metadata: %v", err)
	}
	actionSet := map[string]bool{}
	for _, action := range usersMeta.Actions {
		actionSet[action] = true
	}
	for _, action := range []string{"list", "detail", "create", "update", "delete", "bulk_delete"} {
		if !actionSet[action] {
			t.Fatalf("expected action %q in users metadata, got %+v", action, usersMeta.Actions)
		}
	}
	if !strings.Contains(strings.Join(usersMeta.CreateFields, ","), "password") || !strings.Contains(strings.Join(usersMeta.UpdateFields, ","), "password") {
		t.Fatalf("expected password field in users metadata create/update fields, got %+v", usersMeta)
	}
	if !strings.Contains(strings.Join(usersMeta.CreateFields, ","), "role_ids") || !strings.Contains(strings.Join(usersMeta.UpdateFields, ","), "role_ids") {
		t.Fatalf("expected role_ids field in users metadata create/update fields, got %+v", usersMeta)
	}
	var roleIDsFieldFound bool
	for _, field := range usersMeta.Fields {
		if field.Name == "role_ids" && field.Type == "array" && field.Component == "select" && field.Relation != nil && field.Relation.Resource == "roles" && field.Relation.LabelField == "name" {
			roleIDsFieldFound = true
		}
	}
	if !roleIDsFieldFound {
		t.Fatalf("expected role_ids relation metadata, got %+v", usersMeta.Fields)
	}

	roleOptionsResp := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/users/fields/role_ids/options?search=adm", nil, aliceToken)
	if roleOptionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected role relation selector 200, got %d", roleOptionsResp.StatusCode)
	}
	var roleOptions struct {
		Items []struct {
			Value float64 `json:"value"`
			Label string  `json:"label"`
		} `json:"items"`
	}
	if err := roleOptionsResp.DecodeJSON(&roleOptions); err != nil {
		t.Fatalf("decode role options: %v", err)
	}
	if len(roleOptions.Items) != 1 || roleOptions.Items[0].Value != 1 || roleOptions.Items[0].Label != "Administrators" {
		t.Fatalf("unexpected role relation selector payload: %+v", roleOptions.Items)
	}

	createUserResp := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/users", map[string]any{
		"name":     "  Carol Admin  ",
		"email":    "  CAROL@EXAMPLE.COM ",
		"password": "password123",
		"age":      27,
		"is_admin": true,
		"role_ids": []int{1, 2},
	}, aliceToken)
	if createUserResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected admin user create 201, got %d body=%s", createUserResp.StatusCode, readBody(t, createUserResp))
	}
	var createdUser struct {
		Item map[string]any `json:"item"`
	}
	if err := createUserResp.DecodeJSON(&createdUser); err != nil {
		t.Fatalf("decode created user: %v", err)
	}
	if createdUser.Item["name"] != "Carol Admin" || createdUser.Item["email"] != "carol@example.com" {
		t.Fatalf("expected normalized created user payload, got %+v", createdUser.Item)
	}
	if createdUser.Item["is_admin"] != true {
		t.Fatalf("expected created user to preserve is_admin=true, got %+v", createdUser.Item)
	}
	roleIDs, ok := createdUser.Item["role_ids"].([]any)
	if !ok || len(roleIDs) != 2 || roleIDs[0] != float64(1) || roleIDs[1] != float64(2) {
		t.Fatalf("expected created user role_ids [1 2], got %+v", createdUser.Item["role_ids"])
	}
	if _, ok := createdUser.Item["password"]; ok {
		t.Fatalf("expected password to stay hidden in admin response, got %+v", createdUser.Item)
	}
	if createdUser.Item["id"] != float64(3) {
		t.Fatalf("expected created user id 3, got %+v", createdUser.Item)
	}

	_ = login("carol@example.com")

	updateUserResp := doFullJSON(t, client, http.MethodPut, "/api/v1/admin/resources/users/3", map[string]any{
		"name":     "  Carol Updated  ",
		"email":    "  CAROL.UPDATED@EXAMPLE.COM ",
		"age":      28,
		"role_ids": []int{2},
	}, aliceToken)
	if updateUserResp.StatusCode != http.StatusOK {
		t.Fatalf("expected admin user update 200, got %d body=%s", updateUserResp.StatusCode, readBody(t, updateUserResp))
	}
	var updatedUser struct {
		Item map[string]any `json:"item"`
	}
	if err := updateUserResp.DecodeJSON(&updatedUser); err != nil {
		t.Fatalf("decode updated user: %v", err)
	}
	if updatedUser.Item["name"] != "Carol Updated" || updatedUser.Item["email"] != "carol.updated@example.com" {
		t.Fatalf("expected normalized updated user payload, got %+v", updatedUser.Item)
	}
	updatedRoleIDs, ok := updatedUser.Item["role_ids"].([]any)
	if !ok || len(updatedRoleIDs) != 1 || updatedRoleIDs[0] != float64(2) {
		t.Fatalf("expected updated user role_ids [2], got %+v", updatedUser.Item["role_ids"])
	}

	_ = login("carol.updated@example.com")

	invalidUserResp := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/users", map[string]any{
		"name":  "No Password",
		"email": "nopassword@example.com",
		"age":   19,
	}, aliceToken)
	if invalidUserResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected admin user validation 400, got %d body=%s", invalidUserResp.StatusCode, readBody(t, invalidUserResp))
	}
	invalidUserBody := readBody(t, invalidUserResp)
	if !strings.Contains(invalidUserBody, "password") || !strings.Contains(invalidUserBody, "required") {
		t.Fatalf("expected missing-password validation message, got %s", invalidUserBody)
	}

	createAliceProject := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/projects", map[string]any{
		"title":    "Alice Private Project",
		"summary":  "owned by alice",
		"owner_id": 1,
	}, aliceToken)
	if createAliceProject.StatusCode != http.StatusCreated {
		t.Fatalf("expected alice project create 201, got %d body=%s", createAliceProject.StatusCode, readBody(t, createAliceProject))
	}
	var aliceProject struct {
		Item map[string]any `json:"item"`
	}
	if err := createAliceProject.DecodeJSON(&aliceProject); err != nil {
		t.Fatalf("decode alice project: %v", err)
	}

	createBobProject := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/projects", map[string]any{
		"title":    "Bob Visible Project",
		"summary":  "owned by bob",
		"owner_id": 2,
	}, bobToken)
	if createBobProject.StatusCode != http.StatusCreated {
		t.Fatalf("expected bob project create 201, got %d body=%s", createBobProject.StatusCode, readBody(t, createBobProject))
	}
	var bobProject struct {
		Item map[string]any `json:"item"`
	}
	if err := createBobProject.DecodeJSON(&bobProject); err != nil {
		t.Fatalf("decode bob project: %v", err)
	}

	bobListResp := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects", nil, bobToken)
	if bobListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected bob project list 200, got %d", bobListResp.StatusCode)
	}
	bobListBody := readBody(t, bobListResp)
	if strings.Contains(bobListBody, "Alice Private Project") || !strings.Contains(bobListBody, "Bob Visible Project") {
		t.Fatalf("expected bob project list to be row-scoped, got %s", bobListBody)
	}

	aliceProjectID := int(aliceProject.Item["id"].(float64))
	bobProjectID := int(bobProject.Item["id"].(float64))

	bobReadsAliceResp := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects/"+strconv.Itoa(aliceProjectID), nil, bobToken)
	if bobReadsAliceResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected bob to get 404 for alice project detail, got %d body=%s", bobReadsAliceResp.StatusCode, readBody(t, bobReadsAliceResp))
	}

	bobUpdatesAliceResp := doFullJSON(t, client, http.MethodPut, "/api/v1/admin/resources/projects/"+strconv.Itoa(aliceProjectID), map[string]any{
		"title": "blocked",
	}, bobToken)
	if bobUpdatesAliceResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected bob to get 404 for alice project update, got %d body=%s", bobUpdatesAliceResp.StatusCode, readBody(t, bobUpdatesAliceResp))
	}

	bobBulkDeleteResp := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/projects/bulk-delete", map[string]any{
		"ids": []int{aliceProjectID, bobProjectID},
	}, bobToken)
	if bobBulkDeleteResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected bob bulk delete 201, got %d body=%s", bobBulkDeleteResp.StatusCode, readBody(t, bobBulkDeleteResp))
	}
	var bobBulkDelete struct {
		Deleted int64 `json:"deleted"`
	}
	if err := bobBulkDeleteResp.DecodeJSON(&bobBulkDelete); err != nil {
		t.Fatalf("decode bob bulk delete: %v", err)
	}
	if bobBulkDelete.Deleted != 1 {
		t.Fatalf("expected bob bulk delete to remove only his own project, got %+v", bobBulkDelete)
	}

	aliceProjectStillThereResp := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects/"+strconv.Itoa(aliceProjectID), nil, aliceToken)
	if aliceProjectStillThereResp.StatusCode != http.StatusOK {
		t.Fatalf("expected alice project to remain after bob bulk delete, got %d body=%s", aliceProjectStillThereResp.StatusCode, readBody(t, aliceProjectStillThereResp))
	}
	aliceProjectStillThereBody := readBody(t, aliceProjectStillThereResp)
	if !strings.Contains(aliceProjectStillThereBody, "Alice Private Project") {
		t.Fatalf("expected alice project detail after bob bulk delete, got %s", aliceProjectStillThereBody)
	}
}

func TestFullExampleAdminAPIProjectCRUDAndRelations(t *testing.T) {
	client := newFullTestClient(t)

	register := doFullJSON(t, client, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"name":     "Alice",
		"email":    "alice@example.com",
		"password": "password123",
		"age":      18,
	}, "")
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", register.StatusCode)
	}

	login := doFullJSON(t, client, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email":    "alice@example.com",
		"password": "password123",
	}, "")
	if login.StatusCode != http.StatusCreated {
		t.Fatalf("expected login 201, got %d", login.StatusCode)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if err := login.DecodeJSON(&auth); err != nil {
		t.Fatalf("decode login: %v", err)
	}

	projectMeta := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects/meta", nil, auth.Token)
	if projectMeta.StatusCode != http.StatusOK {
		t.Fatalf("expected project metadata 200, got %d", projectMeta.StatusCode)
	}
	var meta struct {
		Fields []struct {
			Name      string `json:"name"`
			Component string `json:"component"`
			Relation  *struct {
				Resource   string `json:"resource"`
				LabelField string `json:"label_field"`
			} `json:"relation"`
		} `json:"fields"`
	}
	if err := projectMeta.DecodeJSON(&meta); err != nil {
		t.Fatalf("decode project metadata: %v", err)
	}
	var ownerFieldFound bool
	for _, field := range meta.Fields {
		if field.Name == "owner_id" && field.Component == "select" && field.Relation != nil && field.Relation.Resource == "users" && field.Relation.LabelField == "name" {
			ownerFieldFound = true
		}
	}
	if !ownerFieldFound {
		t.Fatalf("expected owner_id relation metadata, got %+v", meta.Fields)
	}

	options := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects/fields/owner_id/options?search=ali", nil, auth.Token)
	if options.StatusCode != http.StatusOK {
		t.Fatalf("expected relation selector 200, got %d", options.StatusCode)
	}
	var optionsPayload struct {
		Items []struct {
			Value float64 `json:"value"`
			Label string  `json:"label"`
		} `json:"items"`
	}
	if err := options.DecodeJSON(&optionsPayload); err != nil {
		t.Fatalf("decode options: %v", err)
	}
	if len(optionsPayload.Items) != 1 || optionsPayload.Items[0].Value != 1 || optionsPayload.Items[0].Label != "Alice" {
		t.Fatalf("unexpected relation selector payload: %+v", optionsPayload.Items)
	}

	optionsByID := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects/fields/owner_id/options?search=1", nil, auth.Token)
	if optionsByID.StatusCode != http.StatusOK {
		t.Fatalf("expected relation selector by id 200, got %d", optionsByID.StatusCode)
	}
	var optionsByIDPayload struct {
		Items []struct {
			Value float64 `json:"value"`
			Label string  `json:"label"`
		} `json:"items"`
	}
	if err := optionsByID.DecodeJSON(&optionsByIDPayload); err != nil {
		t.Fatalf("decode options by id: %v", err)
	}
	if len(optionsByIDPayload.Items) != 1 || optionsByIDPayload.Items[0].Value != 1 || optionsByIDPayload.Items[0].Label != "Alice" {
		t.Fatalf("unexpected relation selector by id payload: %+v", optionsByIDPayload.Items)
	}

	missingOptionsByID := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects/fields/owner_id/options?search=999", nil, auth.Token)
	if missingOptionsByID.StatusCode != http.StatusOK {
		t.Fatalf("expected missing relation selector by id 200, got %d", missingOptionsByID.StatusCode)
	}
	var missingOptionsByIDPayload struct {
		Items []struct {
			Value float64 `json:"value"`
			Label string  `json:"label"`
		} `json:"items"`
	}
	if err := missingOptionsByID.DecodeJSON(&missingOptionsByIDPayload); err != nil {
		t.Fatalf("decode missing options by id: %v", err)
	}
	if len(missingOptionsByIDPayload.Items) != 0 {
		t.Fatalf("expected empty relation selector by missing id payload: %+v", missingOptionsByIDPayload.Items)
	}

	createProject := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/projects", map[string]any{
		"title":    "First Project",
		"summary":  "admin ui demo",
		"owner_id": 1,
	}, auth.Token)
	if createProject.StatusCode != http.StatusCreated {
		t.Fatalf("expected create project 201, got %d body=%s", createProject.StatusCode, readBody(t, createProject))
	}

	projectList := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects", nil, auth.Token)
	if projectList.StatusCode != http.StatusOK {
		t.Fatalf("expected project list 200, got %d", projectList.StatusCode)
	}
	projectListBody := readBody(t, projectList)
	if !strings.Contains(projectListBody, "First Project") {
		t.Fatalf("expected created project in list, got %s", projectListBody)
	}

	createProjectTwo := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/projects", map[string]any{
		"title":    "A Project",
		"summary":  "admin ui demo",
		"owner_id": 1,
	}, auth.Token)
	if createProjectTwo.StatusCode != http.StatusCreated {
		t.Fatalf("expected second create project 201, got %d body=%s", createProjectTwo.StatusCode, readBody(t, createProjectTwo))
	}

	sortedProjects := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects?search=Project&sort=-title", nil, auth.Token)
	if sortedProjects.StatusCode != http.StatusOK {
		t.Fatalf("expected sorted project list 200, got %d", sortedProjects.StatusCode)
	}
	sortedProjectsBody := readBody(t, sortedProjects)
	firstProjectIndex := strings.Index(sortedProjectsBody, "First Project")
	secondProjectIndex := strings.Index(sortedProjectsBody, "A Project")
	if firstProjectIndex == -1 || secondProjectIndex == -1 || firstProjectIndex > secondProjectIndex {
		t.Fatalf("expected sorted projects in descending title order, got %s", sortedProjectsBody)
	}

	pagedProjects := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects?page=2&size=1&sort=title", nil, auth.Token)
	if pagedProjects.StatusCode != http.StatusOK {
		t.Fatalf("expected paged project list 200, got %d", pagedProjects.StatusCode)
	}
	var paged struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
		Page  int              `json:"page"`
		Size  int              `json:"size"`
		Pages int              `json:"pages"`
	}
	if err := pagedProjects.DecodeJSON(&paged); err != nil {
		t.Fatalf("decode paged projects: %v", err)
	}
	if paged.Total != 2 || paged.Page != 2 || paged.Size != 1 || paged.Pages != 2 || len(paged.Items) != 1 {
		t.Fatalf("unexpected paged project payload: %+v", paged)
	}
	if paged.Items[0]["title"] != "First Project" {
		t.Fatalf("expected second page to contain First Project, got %+v", paged.Items)
	}

	sortedProjectsByID := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects?sort=-id", nil, auth.Token)
	if sortedProjectsByID.StatusCode != http.StatusOK {
		t.Fatalf("expected id-sorted project list 200, got %d", sortedProjectsByID.StatusCode)
	}
	var sortedByID struct {
		Items []map[string]any `json:"items"`
	}
	if err := sortedProjectsByID.DecodeJSON(&sortedByID); err != nil {
		t.Fatalf("decode id-sorted projects: %v", err)
	}
	if len(sortedByID.Items) < 2 || sortedByID.Items[0]["id"] != float64(2) || sortedByID.Items[1]["id"] != float64(1) {
		t.Fatalf("unexpected id-sorted projects payload: %+v", sortedByID.Items)
	}

	filteredProjectsByID := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects?id=2", nil, auth.Token)
	if filteredProjectsByID.StatusCode != http.StatusOK {
		t.Fatalf("expected id-filtered project list 200, got %d", filteredProjectsByID.StatusCode)
	}
	var filteredByID struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := filteredProjectsByID.DecodeJSON(&filteredByID); err != nil {
		t.Fatalf("decode id-filtered projects: %v", err)
	}
	if filteredByID.Total != 1 || len(filteredByID.Items) != 1 || filteredByID.Items[0]["id"] != float64(2) {
		t.Fatalf("unexpected id-filtered projects payload: %+v", filteredByID)
	}

	updateProject := doFullJSON(t, client, http.MethodPut, "/api/v1/admin/resources/projects/1", map[string]any{
		"title":    "Renamed Project",
		"summary":  "updated via admin api",
		"owner_id": 1,
	}, auth.Token)
	if updateProject.StatusCode != http.StatusOK {
		t.Fatalf("expected update project 200, got %d body=%s", updateProject.StatusCode, readBody(t, updateProject))
	}

	projectDetail := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects/1", nil, auth.Token)
	if projectDetail.StatusCode != http.StatusOK {
		t.Fatalf("expected project detail 200, got %d", projectDetail.StatusCode)
	}
	projectDetailBody := readBody(t, projectDetail)
	if !strings.Contains(projectDetailBody, "Renamed Project") {
		t.Fatalf("expected updated project detail, got %s", projectDetailBody)
	}

	partialUpdate := doFullJSON(t, client, http.MethodPut, "/api/v1/admin/resources/projects/2", map[string]any{
		"summary": "bulk edit compatible partial update",
	}, auth.Token)
	if partialUpdate.StatusCode != http.StatusOK {
		t.Fatalf("expected partial update 200, got %d body=%s", partialUpdate.StatusCode, readBody(t, partialUpdate))
	}

	projectDetailTwo := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects/2", nil, auth.Token)
	if projectDetailTwo.StatusCode != http.StatusOK {
		t.Fatalf("expected second project detail 200, got %d", projectDetailTwo.StatusCode)
	}
	projectDetailTwoBody := readBody(t, projectDetailTwo)
	if !strings.Contains(projectDetailTwoBody, "bulk edit compatible partial update") {
		t.Fatalf("expected partial update summary in second project detail, got %s", projectDetailTwoBody)
	}

	deleteProject := doFullJSON(t, client, http.MethodDelete, "/api/v1/admin/resources/projects/1", nil, auth.Token)
	if deleteProject.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete project 204, got %d body=%s", deleteProject.StatusCode, readBody(t, deleteProject))
	}

	projectListAfterDelete := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects", nil, auth.Token)
	if projectListAfterDelete.StatusCode != http.StatusOK {
		t.Fatalf("expected project list after delete 200, got %d", projectListAfterDelete.StatusCode)
	}
	projectListAfterDeleteBody := readBody(t, projectListAfterDelete)
	if strings.Contains(projectListAfterDeleteBody, "Renamed Project") {
		t.Fatalf("expected deleted project to be absent, got %s", projectListAfterDeleteBody)
	}

	bulkDeleteProjects := doFullJSON(t, client, http.MethodPost, "/api/v1/admin/resources/projects/bulk-delete", map[string]any{
		"ids": []int{2},
	}, auth.Token)
	if bulkDeleteProjects.StatusCode != http.StatusCreated {
		t.Fatalf("expected bulk delete project 201, got %d body=%s", bulkDeleteProjects.StatusCode, readBody(t, bulkDeleteProjects))
	}
	var bulkDelete struct {
		Deleted int64 `json:"deleted"`
	}
	if err := bulkDeleteProjects.DecodeJSON(&bulkDelete); err != nil {
		t.Fatalf("decode bulk delete: %v", err)
	}
	if bulkDelete.Deleted != 1 {
		t.Fatalf("expected one bulk deleted project, got %+v", bulkDelete)
	}

	projectListAfterBulkDelete := doFullJSON(t, client, http.MethodGet, "/api/v1/admin/resources/projects?search=Project", nil, auth.Token)
	if projectListAfterBulkDelete.StatusCode != http.StatusOK {
		t.Fatalf("expected project list after bulk delete 200, got %d", projectListAfterBulkDelete.StatusCode)
	}
	projectListAfterBulkDeleteBody := readBody(t, projectListAfterBulkDelete)
	if strings.Contains(projectListAfterBulkDeleteBody, "A Project") {
		t.Fatalf("expected bulk deleted project to be absent, got %s", projectListAfterBulkDeleteBody)
	}
}
