package entity

import "testing"

func TestRoleHasManageImpliesAllActions(t *testing.T) {
	r := Role{Permissions: []Permission{{Resource: ResourceCampaigns, Action: ActionManage}}}
	for _, action := range []string{ActionRead, ActionCreate, ActionUpdate, ActionDelete} {
		if !r.Has(ResourceCampaigns, action) {
			t.Fatalf("manage should imply %s", action)
		}
	}
	if r.Has(ResourceUsers, ActionRead) {
		t.Fatal("manage on campaigns must not grant users")
	}
}

func TestRoleHasWildcardOwner(t *testing.T) {
	r := Role{Permissions: SystemRolePermissions(SystemRoleOwner)}
	if !r.Has(ResourceUsers, ActionDelete) || !r.Has(ResourceAudit, ActionRead) {
		t.Fatal("owner wildcard should grant everything")
	}
}

func TestRoleHasSpecificAction(t *testing.T) {
	r := Role{Permissions: []Permission{{Resource: ResourceContacts, Action: ActionRead}}}
	if !r.Has(ResourceContacts, ActionRead) {
		t.Fatal("should grant the exact permission")
	}
	if r.Has(ResourceContacts, ActionDelete) {
		t.Fatal("must not grant an ungranted action")
	}
}

func TestAgentRoleIsLimited(t *testing.T) {
	r := Role{Permissions: SystemRolePermissions(SystemRoleAgent)}
	if !r.Has(ResourceConversations, ActionRead) {
		t.Fatal("agent should read conversations")
	}
	if r.Has(ResourceUsers, ActionCreate) {
		t.Fatal("agent must not manage users")
	}
	if r.Has(ResourceCampaigns, ActionCreate) {
		t.Fatal("agent must not create campaigns")
	}
}

func TestSupervisorCanManageCampaignsButOnlyReadAnalytics(t *testing.T) {
	r := Role{Permissions: SystemRolePermissions(SystemRoleSupervisor)}
	if !r.Has(ResourceCampaigns, ActionCreate) {
		t.Fatal("supervisor should manage campaigns")
	}
	if !r.Has(ResourceAnalytics, ActionRead) {
		t.Fatal("supervisor should read analytics")
	}
	if r.Has(ResourceAnalytics, ActionDelete) {
		t.Fatal("supervisor must not delete analytics")
	}
}
