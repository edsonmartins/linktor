package flows

import "testing"

// TestFlowJSONBuilder_MultiScreen proves a valid multi-screen flow builds
// without the historical "duplicate screen ID" error (screens used to be
// counted twice during validation).
func TestFlowJSONBuilder_MultiScreen(t *testing.T) {
	b := NewFlowJSONBuilder()
	b.Screen("SCREEN_A").SetTitle("A").AddBody("first")
	b.Screen("SCREEN_B").SetTitle("B").SetTerminal(true).AddBody("second")

	flow, err := b.Build()
	if err != nil {
		t.Fatalf("Build failed for a valid multi-screen flow: %v", err)
	}
	if len(flow.Screens) != 2 {
		t.Fatalf("expected 2 screens, got %d", len(flow.Screens))
	}

	ids := map[string]bool{}
	for _, s := range flow.Screens {
		id, _ := s["id"].(string)
		if ids[id] {
			t.Fatalf("screen %q appears more than once", id)
		}
		ids[id] = true
	}
	if !ids["SCREEN_A"] || !ids["SCREEN_B"] {
		t.Fatalf("missing expected screen ids: %v", ids)
	}
}

// TestContactFormTemplate_Builds ensures the shipped FlowJSON templates are
// actually usable (they previously always failed validation).
func TestContactFormTemplate_Builds(t *testing.T) {
	if _, err := ContactFormTemplate().Build(); err != nil {
		t.Fatalf("ContactFormTemplate did not build: %v", err)
	}
	if _, err := LeadCaptureTemplate("Offer", "desc").Build(); err != nil {
		t.Fatalf("LeadCaptureTemplate did not build: %v", err)
	}
}

// TestFlowJSONBuilder_DuplicateStillRejected makes sure the fix did not disable
// duplicate detection.
func TestFlowJSONBuilder_DuplicateStillRejected(t *testing.T) {
	b := NewFlowJSONBuilder()
	b.Screen("DUP").AddBody("one")
	b.Screen("DUP").AddBody("two")

	if _, err := b.Build(); err == nil {
		t.Fatalf("expected duplicate screen ID error, got nil")
	}
}
