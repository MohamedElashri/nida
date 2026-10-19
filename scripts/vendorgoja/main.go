// Command vendorgoja keeps third_party/goja, Nida's patched copy of the goja
// JavaScript engine, in sync with upstream.
//
// The copy is upstream goja at the version go.mod requires, without tests,
// with third_party/goja.patch applied. go.mod points goja at the copy with a
// replace directive.
//
//	go run ./scripts/vendorgoja                  rebuild the copy from go.mod's version
//	go run ./scripts/vendorgoja -update master   move to a newer upstream commit
//	go run ./scripts/vendorgoja -check           fail if the copy differs from upstream + patch
//	go run ./scripts/vendorgoja -outdated        print the newest upstream version, if newer
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	module    = "github.com/dop251/goja"
	destDir   = "third_party/goja"
	patchFile = "third_party/goja.patch"
)

// skipDirs and skipFile drop what Nida does not build: tests, test data,
// upstream CI, and the goja command-line tool. skipDirs holds paths
// relative to the module root.
var skipDirs = map[string]bool{".github": true, "testdata": true, "goja": true}

func skipFile(name string) bool {
	return strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".sh") || strings.HasPrefix(name, ".")
}

func main() {
	update := flag.String("update", "", "upstream version or branch to move to, such as master")
	check := flag.Bool("check", false, "verify the copy instead of writing it")
	outdated := flag.Bool("outdated", false, "print the newest upstream version if it is newer than go.mod's")
	flag.Parse()
	if *check && *update != "" {
		fmt.Fprintln(os.Stderr, "vendorgoja: -check and -update cannot be combined")
		os.Exit(2)
	}
	if err := run(*update, *check, *outdated); err != nil {
		fmt.Fprintln(os.Stderr, "vendorgoja:", err)
		os.Exit(1)
	}
}

func run(update string, check, outdated bool) error {
	pinned, err := requiredVersion()
	if err != nil {
		return err
	}
	if outdated {
		latest, err := download("master")
		if err != nil {
			return err
		}
		if latest.Version != pinned {
			fmt.Println(latest.Version)
		}
		return nil
	}

	version := pinned
	if update != "" {
		version = update
	}
	mod, err := download(version)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "vendorgoja-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	if err := copyTree(mod.Dir, tmp); err != nil {
		return err
	}
	if err := applyPatch(tmp); err != nil {
		return fmt.Errorf("%s no longer applies to goja %s: %w\nupdate the patch against the new version, then run again", patchFile, mod.Version, err)
	}

	if check {
		diffs, err := compareTrees(tmp, destDir)
		if err != nil {
			return err
		}
		if len(diffs) > 0 {
			return fmt.Errorf("%s differs from goja %s with %s applied:\n  %s\nrun: go run ./scripts/vendorgoja",
				destDir, pinned, patchFile, strings.Join(diffs, "\n  "))
		}
		fmt.Printf("%s matches goja %s with %s applied\n", destDir, pinned, patchFile)
		return nil
	}

	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	if err := copyTree(tmp, destDir); err != nil {
		return err
	}
	if mod.Version != pinned {
		if err := goCmd(nil, "mod", "edit", "-require="+module+"@"+mod.Version); err != nil {
			return err
		}
		if err := goCmd(nil, "mod", "tidy"); err != nil {
			return err
		}
	}
	fmt.Printf("wrote %s from goja %s with %s applied\n", destDir, mod.Version, patchFile)
	return nil
}

type moduleInfo struct {
	Version string
	Dir     string
	Error   string
}

func download(version string) (moduleInfo, error) {
	var out bytes.Buffer
	cmd := exec.Command("go", "mod", "download", "-json", module+"@"+version)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	runErr := cmd.Run()
	var info moduleInfo
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		if runErr != nil {
			return info, runErr
		}
		return info, err
	}
	if info.Error != "" {
		return info, errors.New(info.Error)
	}
	return info, runErr
}

func requiredVersion() (string, error) {
	var out bytes.Buffer
	if err := goCmd(&out, "mod", "edit", "-json"); err != nil {
		return "", err
	}
	var gomod struct {
		Require []struct{ Path, Version string }
		Replace []struct {
			Old struct{ Path string }
			New struct{ Path string }
		}
	}
	if err := json.Unmarshal(out.Bytes(), &gomod); err != nil {
		return "", err
	}
	replaced := false
	for _, r := range gomod.Replace {
		if r.Old.Path == module && r.New.Path == "./"+destDir {
			replaced = true
		}
	}
	if !replaced {
		return "", fmt.Errorf("go.mod must contain: replace %s => ./%s", module, destDir)
	}
	for _, r := range gomod.Require {
		if r.Path == module {
			return r.Version, nil
		}
	}
	return "", fmt.Errorf("go.mod does not require %s", module)
}

func goCmd(stdout *bytes.Buffer, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr
	if stdout != nil {
		cmd.Stdout = stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	return cmd.Run()
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[filepath.ToSlash(rel)] || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if skipFile(d.Name()) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
}

func applyPatch(dir string) error {
	patch, err := filepath.Abs(patchFile)
	if err != nil {
		return err
	}
	// Outside a git repository, git apply works like patch(1).
	cmd := exec.Command("git", "apply", "--verbose", patch)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CEILING_DIRECTORIES="+filepath.Dir(dir))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func compareTrees(want, got string) ([]string, error) {
	wantFiles, err := listFiles(want)
	if err != nil {
		return nil, err
	}
	gotFiles, err := listFiles(got)
	if err != nil {
		return nil, err
	}
	var diffs []string
	for name, data := range wantFiles {
		have, ok := gotFiles[name]
		switch {
		case !ok:
			diffs = append(diffs, "missing "+name)
		case !bytes.Equal(data, have):
			diffs = append(diffs, "changed "+name)
		}
	}
	for name := range gotFiles {
		if _, ok := wantFiles[name]; !ok {
			diffs = append(diffs, "extra "+name)
		}
	}
	sort.Strings(diffs)
	return diffs, nil
}

func listFiles(root string) (map[string][]byte, error) {
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		files[filepath.ToSlash(rel)] = data
		return err
	})
	return files, err
}
