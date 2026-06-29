package services

import (
	"testing"

	"expense-splitter/types"
)

func TestDecideGroupRole(t *testing.T) {
	admin := &types.Principal{IsGlobalAdmin: true}
	user := &types.Principal{IsGlobalAdmin: false}

	cases := []struct {
		name    string
		p       *types.Principal
		m       types.Membership
		found   bool
		need    types.MembershipRole
		allowed bool
	}{
		{"global admin, no membership, needs admin", admin, types.Membership{}, false, types.RoleGroupAdmin, true},
		{"global admin, no membership, needs member", admin, types.Membership{}, false, types.RoleMember, true},
		{"global admin, requested membership, needs admin", admin, types.Membership{Role: types.RoleMember, Status: types.MembershipRequested}, true, types.RoleGroupAdmin, true},

		{"approved member, needs member", user, types.Membership{Role: types.RoleMember, Status: types.MembershipApproved}, true, types.RoleMember, true},
		{"approved member, needs admin", user, types.Membership{Role: types.RoleMember, Status: types.MembershipApproved}, true, types.RoleGroupAdmin, false},
		{"approved group-admin, needs admin", user, types.Membership{Role: types.RoleGroupAdmin, Status: types.MembershipApproved}, true, types.RoleGroupAdmin, true},
		{"approved group-admin, needs member", user, types.Membership{Role: types.RoleGroupAdmin, Status: types.MembershipApproved}, true, types.RoleMember, true},

		{"not a member, needs member", user, types.Membership{}, false, types.RoleMember, false},
		{"not a member, needs admin", user, types.Membership{}, false, types.RoleGroupAdmin, false},

		{"requested member, needs member", user, types.Membership{Role: types.RoleMember, Status: types.MembershipRequested}, true, types.RoleMember, false},
		{"rejected member, needs member", user, types.Membership{Role: types.RoleMember, Status: types.MembershipRejected}, true, types.RoleMember, false},
		{"requested group-admin, needs admin", user, types.Membership{Role: types.RoleGroupAdmin, Status: types.MembershipRequested}, true, types.RoleGroupAdmin, false},
		{"rejected group-admin, needs admin", user, types.Membership{Role: types.RoleGroupAdmin, Status: types.MembershipRejected}, true, types.RoleGroupAdmin, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := decideGroupRole(c.p, c.m, c.found, c.need)
			if c.allowed && err != nil {
				t.Fatalf("expected allow, got deny: %v", err)
			}
			if !c.allowed && err == nil {
				t.Fatalf("expected deny, got allow")
			}
		})
	}
}

func TestRequireGlobalAdmin(t *testing.T) {
	a := NewAuthorizer(nil, nil)

	if err := a.RequireGlobalAdmin(&types.Principal{IsGlobalAdmin: true}); err != nil {
		t.Fatalf("global admin should pass: %v", err)
	}
	if err := a.RequireGlobalAdmin(&types.Principal{IsGlobalAdmin: false}); err == nil {
		t.Fatalf("non-admin should be forbidden")
	}
}

func TestRequireVerified(t *testing.T) {
	a := NewAuthorizer(nil, nil)

	if err := a.RequireVerified(&types.Principal{VerificationStatus: types.VerificationVerified}); err != nil {
		t.Fatalf("verified should pass: %v", err)
	}
	for _, s := range []types.VerificationStatus{types.VerificationRegistered, types.VerificationPending, types.VerificationRejected} {
		if err := a.RequireVerified(&types.Principal{VerificationStatus: s}); err == nil {
			t.Fatalf("status %q should be forbidden", s)
		}
	}
}
