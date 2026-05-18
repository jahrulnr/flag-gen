// nolint
package middleware

import "testing"

func TestDashboardHasPermissionAllowsAdminWildcardForDashboardActions(t *testing.T) {
	if !dashboardHasPermission([]string{"*:read", "*:write", "admin:*"}, "flags:evaluate") {
		t.Fatal("admin wildcard should allow dashboard flag evaluation")
	}
}

func TestDashboardHasPermissionKeepsScopedEvaluatePermission(t *testing.T) {
	if !dashboardHasPermission([]string{"*:read", "flags:evaluate"}, "flags:evaluate") {
		t.Fatal("explicit evaluate permission should allow flag evaluation")
	}
}

func TestDashboardHasPermissionRejectsReadOnlyEvaluate(t *testing.T) {
	if dashboardHasPermission([]string{"*:read"}, "flags:evaluate") {
		t.Fatal("read-only permission should not allow flag evaluation")
	}
}
