package repo

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/andyballingall/json-schema-manager/internal/config"
)

// absPath is a variable for filepath.Abs to allow mocking in tests.
var absPath = filepath.Abs

// CLIGitter is the concrete implementation of Gitter using the git CLI.
type CLIGitter struct {
	cfg *config.Config
}

// NewCLIGitter creates a new CLIGitter instance.
func NewCLIGitter(cfg *config.Config) *CLIGitter {
	return &CLIGitter{cfg: cfg}
}

// getEnvConfig looks up the EnvConfig for the given environment.
func (g *CLIGitter) getEnvConfig(env config.Env) (*config.EnvConfig, error) {
	return g.cfg.EnvConfig(env)
}

// tagPrefix returns the tag prefix for the given environment.
func (g *CLIGitter) tagPrefix(env config.Env) string {
	return fmt.Sprintf("%s/%s", JSMDeployTagPrefix, env)
}

// GetLatestAnchor finds the latest deployment tag for an environment.
// If no tag is found, it returns the repository's initial commit.
func (g *CLIGitter) GetLatestAnchor(env config.Env) (Revision, error) {
	if _, err := g.getEnvConfig(env); err != nil {
		return "", err
	}

	tagPattern := fmt.Sprintf("%s/*", g.tagPrefix(env))

	// --abbrev=0 finds the closest reachable tag matching the pattern
	cmd := exec.Command("git", "describe", "--tags", "--match", tagPattern, "--abbrev=0")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// Fallback: Get the root commit (Day Zero)
		revCmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
		revOut, rErr := revCmd.Output()
		if rErr != nil {
			return "", fmt.Errorf("could not find git history: %w", rErr)
		}
		return Revision(strings.TrimSpace(string(revOut))), nil
	}

	return Revision(strings.TrimSpace(out.String())), nil
}

// getGitRoot finds the top-level directory of the git repository.
func (g *CLIGitter) getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to find git root: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// TagDeploymentSuccess creates and pushes a new environment-specific deployment tag.
func (g *CLIGitter) TagDeploymentSuccess(env config.Env) (string, error) {
	if _, err := g.getEnvConfig(env); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102-150405")
	tagName := fmt.Sprintf("%s/%s", g.tagPrefix(env), timestamp)

	// 1. Create the local annotated tag
	tagCmd := exec.Command("git", "tag", "-a", tagName, "-m", fmt.Sprintf("Successful JSM deployment to %s", env))
	if err := tagCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create git tag: %w", err)
	}

	// 2. Push the tag to origin
	pushCmd := exec.Command("git", "push", "origin", tagName)
	if err := pushCmd.Run(); err != nil {
		return tagName, fmt.Errorf("failed to push git tag to origin: %w", err)
	}

	return tagName, nil
}

// GetSchemaChanges identifies files with the given suffix changed between the anchor and HEAD.
func (g *CLIGitter) GetSchemaChanges(anchor Revision, sourceDir, suffix string) ([]Change, error) {
	absSourceDir, err := absPath(sourceDir)
	if err != nil {
		return nil, err
	}

	root, err := g.getGitRoot()
	if err != nil {
		return nil, err
	}

	//nolint:gosec // CMD arguments are internal and path is absolute
	cmd := exec.Command("git", "diff", "--name-status", anchor.String(), "--", absSourceDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w (output: %s)", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	changes := make([]Change, 0, len(lines))

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		path := fields[1]
		if !strings.HasSuffix(path, suffix) {
			continue
		}

		// git diff returns paths relative to the repo root.
		// Resolve these to absolute paths so they are correctly handled regardless of CWD.
		absPath := filepath.Join(root, path)

		changes = append(changes, Change{
			Path:  absPath,
			IsNew: fields[0] == "A",
		})
	}
	return changes, nil
}
