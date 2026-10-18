package content

import (
	"bufio"
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ApplyGitDates sets Updated from each file's last commit when front matter
// does not set it. Without git or a repository, pages are left unchanged.
func ApplyGitDates(contentRoot string, pages []Page) {
	dates := gitLastModified(contentRoot)
	if len(dates) == 0 {
		return
	}
	for i := range pages {
		if !pages[i].Updated.IsZero() {
			continue
		}
		if date, ok := dates[pages[i].RelativePath]; ok {
			pages[i].Updated = date
		}
	}
}

func gitLastModified(contentRoot string) map[string]time.Time {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}
	cmd := exec.Command("git", "-c", "core.quotePath=false", "log", "--format=@@%cI", "--name-only", "--relative", "--", ".")
	cmd.Dir = contentRoot
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	dates := map[string]time.Time{}
	var current time.Time
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "":
		case strings.HasPrefix(line, "@@"):
			current, _ = time.Parse(time.RFC3339, strings.TrimPrefix(line, "@@"))
		case !current.IsZero():
			// git log lists newest commits first, so keep the first date seen.
			path := filepath.ToSlash(line)
			if _, seen := dates[path]; !seen {
				dates[path] = current
			}
		}
	}
	return dates
}
