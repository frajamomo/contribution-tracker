package domain

import "testing"

func TestGetPlatformUsername_ReturnsMappedValue(t *testing.T) {
	u := User{
		Username:          "alice",
		PlatformUsernames: map[GitPlatform]string{PlatformGitHub: "alice-gh"},
	}

	got := u.GetPlatformUsername(PlatformGitHub)
	if got != "alice-gh" {
		t.Errorf("expected alice-gh, got %s", got)
	}
}

func TestGetPlatformUsername_FallsBackToUsername(t *testing.T) {
	u := User{
		Username:          "alice",
		PlatformUsernames: map[GitPlatform]string{PlatformGitHub: "alice-gh"},
	}

	got := u.GetPlatformUsername(PlatformGitLab)
	if got != "alice" {
		t.Errorf("expected alice, got %s", got)
	}
}

func TestGetPlatformUsername_NilMap(t *testing.T) {
	u := User{Username: "alice"}

	got := u.GetPlatformUsername(PlatformGitHub)
	if got != "alice" {
		t.Errorf("expected alice, got %s", got)
	}
}
