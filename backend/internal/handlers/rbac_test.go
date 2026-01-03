package handlers

import "net/http"

import "testing"

func TestRequiredRole(t *testing.T) {
	tests := []struct {
		path     string
		method   string
		expected string
	}{
		{"/api/v1/dashboard", http.MethodGet, roleViewer},
		{"/api/v1/messages/reply", http.MethodPost, roleMember},
		{"/api/v1/llm/providers", http.MethodGet, roleAdmin},
		{"/api/v1/team/users", http.MethodPost, roleAdmin},
		{"/api/v1/workflows", http.MethodGet, roleManager},
		{"/api/v1/webhooks/incoming", http.MethodPost, ""},
	}

	for _, test := range tests {
		if got := RequiredRole(test.path, test.method); got != test.expected {
			t.Fatalf("RequiredRole(%s, %s)=%s, expected %s", test.path, test.method, got, test.expected)
		}
	}
}

func TestRoleRankOrder(t *testing.T) {
	if roleRank[roleOwner] <= roleRank[roleAdmin] {
		t.Fatal("owner should rank above admin")
	}
	if roleRank[roleAdmin] <= roleRank[roleManager] {
		t.Fatal("admin should rank above manager")
	}
	if roleRank[roleManager] <= roleRank[roleMember] {
		t.Fatal("manager should rank above member")
	}
	if roleRank[roleMember] <= roleRank[roleViewer] {
		t.Fatal("member should rank above viewer")
	}
}
