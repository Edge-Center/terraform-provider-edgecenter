package converter

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

func runFixture(t *testing.T, name string) map[string][]byte {
	t.Helper()
	dir := filepath.Join("testdata", name, "in")
	statePath := ""
	if _, err := os.Stat(filepath.Join(dir, "terraform.tfstate")); err == nil {
		statePath = filepath.Join(dir, "terraform.tfstate")
	}
	res, err := Run(Options{
		Dir:            dir,
		StatePath:      statePath,
		MigrationsPath: filepath.Join(dir, "v2-migrate.tf"),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]byte{}
	for p, b := range res.Changed {
		got[filepath.Base(p)] = b
	}
	got["report.md"] = []byte(res.Report.Render())

	return got
}

func TestGolden(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"basic", "edge", "nostate"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := runFixture(t, name)
			wantDir := filepath.Join("testdata", name, "want")

			if *update {
				if err := os.RemoveAll(wantDir); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(wantDir, 0o750); err != nil {
					t.Fatal(err)
				}
				for f, b := range got {
					if err := os.WriteFile(filepath.Join(wantDir, f), b, 0o600); err != nil {
						t.Fatal(err)
					}
				}

				return
			}

			entries, err := os.ReadDir(wantDir)
			if err != nil {
				t.Fatal(err)
			}
			wantFiles := map[string]bool{}
			for _, e := range entries {
				wantFiles[e.Name()] = true
				want, err := os.ReadFile(filepath.Join(wantDir, e.Name()))
				if err != nil {
					t.Fatal(err)
				}
				gotBytes, ok := got[e.Name()]
				if !ok {
					t.Errorf("%s: expected in output, missing", e.Name())
					continue
				}
				if string(gotBytes) != string(want) {
					t.Errorf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", e.Name(), gotBytes, want)
				}
			}
			var extra []string
			for f := range got {
				if !wantFiles[f] {
					extra = append(extra, f)
				}
			}
			sort.Strings(extra)
			if len(extra) > 0 {
				t.Errorf("unexpected output files: %s", strings.Join(extra, ", "))
			}
		})
	}
}
