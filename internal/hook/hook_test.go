package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a temp directory with a git repo and changes into it.
// It returns the repo path and a cleanup function that restores the original
// working directory and removes the temp dir.
func initTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	dir := t.TempDir()

	if out, err := exec.Command("git", "init", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to temp repo: %v", err)
	}

	cleanup := func() {
		os.Chdir(origDir)
	}

	return dir, cleanup
}

func hookPath(repoDir string) string {
	return filepath.Join(repoDir, ".git", "hooks", "prepare-commit-msg")
}

func TestInstallCreatesHookWithCorrectPermissions(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Install(false); err != nil {
		t.Fatalf("Install: %v", err)
	}

	hp := hookPath(dir)
	info, err := os.Stat(hp)
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0755 {
		t.Errorf("expected permissions 0755, got %04o", perm)
	}
}

func TestInstallCreatesHookContainingMarker(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Install(false); err != nil {
		t.Fatalf("Install: %v", err)
	}

	data, err := os.ReadFile(hookPath(dir))
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, marker) {
		t.Errorf("hook does not contain marker %q:\n%s", marker, content)
	}
	if !strings.HasPrefix(content, "#!/bin/sh") {
		t.Errorf("hook does not start with shebang:\n%s", content)
	}
	if !strings.Contains(content, "generate") {
		t.Errorf("hook does not contain 'generate' command:\n%s", content)
	}
}

func TestUninstallRemovesHook(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Install(false); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := Uninstall(); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, err := os.Stat(hookPath(dir)); !os.IsNotExist(err) {
		t.Errorf("expected hook to be removed, but it still exists")
	}
}

func TestUninstallFailsGracefullyWhenNoHookExists(t *testing.T) {
	_, cleanup := initTestRepo(t)
	defer cleanup()

	err := Uninstall()
	if err == nil {
		t.Fatal("expected error when no hook exists, got nil")
	}
	if !strings.Contains(err.Error(), "no prepare-commit-msg hook") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUninstallRefusesToRemoveUnownedHook(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	hp := hookPath(dir)
	if err := os.MkdirAll(filepath.Dir(hp), 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	if err := os.WriteFile(hp, []byte("#!/bin/sh\necho 'custom hook'\n"), 0755); err != nil {
		t.Fatalf("write custom hook: %v", err)
	}

	err := Uninstall()
	if err == nil {
		t.Fatal("expected error when hook is not owned, got nil")
	}
	if !strings.Contains(err.Error(), "not installed by") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Verify the hook was not removed
	if _, err := os.Stat(hp); os.IsNotExist(err) {
		t.Error("unowned hook was removed, but should have been preserved")
	}
}

func TestInstallWithoutForceFailsWhenHookExists(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	if err := Install(false); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	err := Install(false)
	if err == nil {
		t.Fatal("expected error on second install without force, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Verify the original hook still exists
	if _, err := os.Stat(hookPath(dir)); err != nil {
		t.Errorf("hook should still exist: %v", err)
	}
}

func TestInstallWithForceOverwritesExistingHook(t *testing.T) {
	dir, cleanup := initTestRepo(t)
	defer cleanup()

	hp := hookPath(dir)
	if err := os.MkdirAll(filepath.Dir(hp), 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	if err := os.WriteFile(hp, []byte("#!/bin/sh\necho 'old hook'\n"), 0755); err != nil {
		t.Fatalf("write old hook: %v", err)
	}

	if err := Install(true); err != nil {
		t.Fatalf("Install with force: %v", err)
	}

	data, err := os.ReadFile(hp)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}

	if !strings.Contains(string(data), marker) {
		t.Errorf("hook was not overwritten; missing marker:\n%s", string(data))
	}
}
