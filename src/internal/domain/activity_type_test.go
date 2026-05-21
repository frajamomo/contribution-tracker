package domain

import "testing"

func TestActivityType_Equality(t *testing.T) {
	a := ActivityType{Name: "COMMIT", DisplayName: "Commit"}
	b := ActivityTypeCommit

	if a != b {
		t.Error("expected equal ActivityType structs to be equal")
	}

	c := ActivityType{Name: "CUSTOM", DisplayName: "Custom Type"}
	if a == c {
		t.Error("expected different ActivityType structs to be not equal")
	}
}

func TestGitPlatform_Equality(t *testing.T) {
	a := GitPlatform{Name: "GITHUB"}
	if a != PlatformGitHub {
		t.Error("expected equal GitPlatform structs to be equal")
	}

	b := GitPlatform{Name: "BITBUCKET"}
	if a == b {
		t.Error("expected different GitPlatform structs to be not equal")
	}
}
