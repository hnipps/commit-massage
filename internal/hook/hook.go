package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nicholls-inc/commit-massage/internal/log"
)

const marker = "commit-massage"

const hookName = "prepare-commit-msg"

func hooksDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--git-path", "hooks").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git not installed)")
	}

	dir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(dir) {
		toplevel, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return "", fmt.Errorf("could not determine repository root")
		}
		dir = filepath.Join(strings.TrimSpace(string(toplevel)), dir)
	}

	return dir, nil
}

// Install writes the prepare-commit-msg hook script. If force is false,
// it refuses to overwrite an existing hook file.
func Install(force bool) error {
	s := log.Start("Locating git hooks directory...")
	dir, err := hooksDir()
	if err != nil {
		s.Fail("Not a git repository")
		return err
	}
	s.Stop("Found hooks directory")

	hookPath := filepath.Join(dir, hookName)

	if !force {
		s = log.Start("Checking for existing hook...")
		if _, err := os.Stat(hookPath); err == nil {
			s.Fail("Hook already exists")
			return fmt.Errorf("hook already exists at %s (use --force to overwrite)", hookPath)
		}
		s.Stop("No conflicting hook found")
	}

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	binPath, err = filepath.EvalSymlinks(binPath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	script := fmt.Sprintf("#!/bin/sh\n# %s: AI-generated conventional commits via Ollama\nexec %s generate \"$1\" \"$2\"\n", marker, binPath)

	s = log.Start("Writing hook script...")
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.Fail("Could not create hooks directory")
		return fmt.Errorf("could not create hooks directory: %w", err)
	}

	if err := os.WriteFile(hookPath, []byte(script), 0755); err != nil {
		s.Fail("Could not write hook")
		return fmt.Errorf("could not write hook: %w", err)
	}
	s.Stop(fmt.Sprintf("Installed hook at %s", hookPath))

	return nil
}

// Uninstall removes the prepare-commit-msg hook, but only if it was
// installed by commit-massage (identified by the marker comment).
func Uninstall() error {
	s := log.Start("Locating git hooks directory...")
	dir, err := hooksDir()
	if err != nil {
		s.Fail("Not a git repository")
		return err
	}
	s.Stop("Found hooks directory")

	hookPath := filepath.Join(dir, hookName)

	s = log.Start("Verifying hook ownership...")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.Fail("No hook found")
			return fmt.Errorf("no prepare-commit-msg hook found")
		}
		s.Fail("Could not read hook")
		return fmt.Errorf("could not read hook: %w", err)
	}

	if !strings.Contains(string(data), marker) {
		s.Fail("Hook not owned by commit-massage")
		return fmt.Errorf("hook at %s was not installed by commit-massage, refusing to remove", hookPath)
	}
	s.Stop("Hook belongs to commit-massage")

	s = log.Start("Removing hook...")
	if err := os.Remove(hookPath); err != nil {
		s.Fail("Could not remove hook")
		return fmt.Errorf("could not remove hook: %w", err)
	}
	s.Stop(fmt.Sprintf("Removed hook from %s", hookPath))

	return nil
}
