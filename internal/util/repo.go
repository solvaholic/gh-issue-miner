package util

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// DetectRepo returns repository in owner/repo form. Preference order:
// 1. provided flagRepo (non-empty)
// 2. GH_REPO env var
// 3. parse `git config --get remote.origin.url`
func DetectRepo(flagRepo string) (string, error) {
	if flagRepo != "" {
		return flagRepo, nil
	}
	if g := os.Getenv("GH_REPO"); g != "" {
		return g, nil
	}

	// try git
	out, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return "", errors.New("could not detect repository: set --repo or GH_REPO, or run inside a git repo with origin remote")
	}
	s := strings.TrimSpace(string(out))
	repo := parseRemoteURL(s)
	if repo == "" {
		return "", errors.New("could not parse remote URL to owner/repo; specify --repo")
	}
	return repo, nil
}

var reSSH = regexp.MustCompile(`^git@[^:]+:([^/]+)/(.+?)(?:\.git)?$`)
var reHTTPS = regexp.MustCompile(`^https?://[^/]+/([^/]+)/(.+?)(?:\.git)?$`)

func parseRemoteURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if m := reSSH.FindStringSubmatch(s); len(m) == 3 {
		return m[1] + "/" + strings.TrimSuffix(m[2], ".git")
	}
	if m := reHTTPS.FindStringSubmatch(s); len(m) == 3 {
		return m[1] + "/" + strings.TrimSuffix(m[2], ".git")
	}
	// support ssh://git@github.com/owner/repo.git
	if strings.HasPrefix(s, "ssh://") {
		// remove protocol
		parts := strings.SplitN(strings.TrimPrefix(s, "ssh://"), "/", 3)
		if len(parts) >= 3 {
			return parts[1] + "/" + strings.TrimSuffix(parts[2], ".git")
		}
	}
	return ""
}
