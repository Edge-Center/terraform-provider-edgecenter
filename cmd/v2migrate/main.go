package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Edge-Center/terraform-provider-edgecenter/internal/converter"
)

func main() {
	dir := flag.String("dir", ".", "directory with the terraform configuration to convert")
	state := flag.String("state", "", "path to terraform.tfstate, defaults to <dir>/terraform.tfstate when present, for remote backends run terraform state pull first")
	migrations := flag.String("migrations", "", "path of the generated state migration file, defaults to <dir>/v2-migrate.tf")
	reportPath := flag.String("report", "", "write the report to this file in addition to stdout")
	dryRun := flag.Bool("dry-run", false, "print the report without writing any files")
	flag.Parse()

	if err := run(*dir, *state, *migrations, *reportPath, *dryRun); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(dir, state, migrations, reportPath string, dryRun bool) error {
	if state == "" {
		def := filepath.Join(dir, "terraform.tfstate")
		if _, err := os.Stat(def); err == nil {
			state = def
		}
	}
	if migrations == "" {
		migrations = filepath.Join(dir, "v2-migrate.tf")
	}
	if _, err := os.Stat(migrations); err == nil {
		return fmt.Errorf("migration file %s already exists, apply and delete it or pass -migrations", migrations)
	}

	res, err := converter.Run(converter.Options{
		Dir:            dir,
		StatePath:      state,
		MigrationsPath: migrations,
	})
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}

	text := res.Report.Render()
	fmt.Fprint(os.Stdout, text)
	if reportPath != "" {
		if err := os.WriteFile(reportPath, []byte(text), 0o600); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
	}

	if dryRun {
		if len(res.Changed) > 0 {
			fmt.Fprintln(os.Stdout, "\nDry run, files that would change:")
			paths := make([]string, 0, len(res.Changed))
			for p := range res.Changed {
				paths = append(paths, p)
			}
			sort.Strings(paths)
			for _, p := range paths {
				fmt.Fprintln(os.Stdout, "  "+p)
			}
		}

		return nil
	}
	if err := converter.WriteResult(res); err != nil {
		return fmt.Errorf("write result: %w", err)
	}
	if len(res.Changed) > 0 {
		fmt.Fprintf(os.Stdout, "\n%d file(s) written.\n", len(res.Changed))
	}

	return nil
}
