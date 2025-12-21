package campaign

import "testing"

func TestCampaignSpec_IsLauncherEnabled_DefaultsToTrue(t *testing.T) {
	spec := &CampaignSpec{ID: "test"}
	if !spec.IsLauncherEnabled() {
		t.Fatal("expected launcher to be enabled by default")
	}

	spec = &CampaignSpec{ID: "test", Launcher: &CampaignLauncherConfig{}}
	if !spec.IsLauncherEnabled() {
		t.Fatal("expected launcher to be enabled when launcher.enabled is omitted")
	}
}

func TestCampaignSpec_IsLauncherEnabled_CanDisable(t *testing.T) {
	falsePtr := boolPtr(false)
	spec := &CampaignSpec{ID: "test", Launcher: &CampaignLauncherConfig{Enabled: falsePtr}}
	if spec.IsLauncherEnabled() {
		t.Fatal("expected launcher to be disabled when launcher.enabled is false")
	}
}
