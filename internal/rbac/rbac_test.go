package rbac

import "testing"

func TestAuthorizerCan(t *testing.T) {
	authz, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err := authz.AddRoleForUser("u1", "admin"); err != nil {
		t.Fatal(err)
	}
	if err := authz.SetRolePermissions("admin", []string{"/api/v1/forms:*", "/api/v1/reports:read"}); err != nil {
		t.Fatal(err)
	}
	if !authz.Can("u1", "/api/v1/forms", "delete") {
		t.Fatal("expected admin to manage forms")
	}
	if authz.Can("u1", "/api/v1/data-sources", "delete") {
		t.Fatal("expected data-source delete to be denied")
	}
}
