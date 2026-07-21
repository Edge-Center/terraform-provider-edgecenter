package converter

import (
	"fmt"
	"sort"
	"strings"
)

type Finding struct {
	File string
	Line int
	Addr string
	Text string
}

type ConvertedItem struct {
	File    string
	Line    int
	Kind    string
	OldAddr string
	NewAddr string
}

type ExtractedItem struct {
	File       string
	ParentAddr string
	NewAddr    string
}

type Report struct {
	Converted   []ConvertedItem
	Extracted   []ExtractedItem
	Todos       []Finding
	Warns       []Finding
	Infos       []Finding
	RemovedN    int
	ImportsN    int
	Migrations  string
	Placeholder bool
}

func (r *Report) todo(file string, line int, addr, text string) {
	r.Todos = append(r.Todos, Finding{File: file, Line: line, Addr: addr, Text: text})
}

func (r *Report) warn(file string, line int, addr, text string) {
	r.Warns = append(r.Warns, Finding{File: file, Line: line, Addr: addr, Text: text})
}

func (r *Report) info(file string, line int, addr, text string) {
	r.Infos = append(r.Infos, Finding{File: file, Line: line, Addr: addr, Text: text})
}

func sortFindings(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].File != fs[j].File {
			return fs[i].File < fs[j].File
		}
		return fs[i].Line < fs[j].Line
	})
}

func writeFindings(b *strings.Builder, fs []Finding) {
	sortFindings(fs)
	for _, f := range fs {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		parts := make([]string, 0, 2)
		if loc != "" {
			parts = append(parts, loc)
		}
		if f.Addr != "" {
			parts = append(parts, f.Addr)
		}
		fmt.Fprintf(b, "- %s: %s\n", strings.Join(parts, " "), f.Text)
	}
}

func (r *Report) Render() string {
	var b strings.Builder
	b.WriteString("# v1 to v2 conversion report\n\n")

	if len(r.Converted) == 0 {
		b.WriteString("No v1 resources or data sources found, nothing to convert.\n")
		return b.String()
	}

	b.WriteString("## Converted\n\n")
	for _, c := range r.Converted {
		fmt.Fprintf(&b, "- %s -> %s (%s:%d)\n", c.OldAddr, c.NewAddr, c.File, c.Line)
	}
	b.WriteString("\n")

	if len(r.Extracted) > 0 {
		b.WriteString("## Extracted resources\n\n")
		for _, e := range r.Extracted {
			fmt.Fprintf(&b, "- %s nested block -> %s (%s)\n", e.ParentAddr, e.NewAddr, e.File)
		}
		b.WriteString("\n")
	}

	if r.Migrations != "" {
		b.WriteString("## State migration\n\n")
		fmt.Fprintf(&b, "- %s: %d removed block(s), %d import block(s)\n", r.Migrations, r.RemovedN, r.ImportsN)
		if r.Placeholder {
			b.WriteString("- some import ids could not be resolved from state and contain <placeholders>, fill them before applying\n")
		}
		b.WriteString("\n")
	}

	if len(r.Todos) > 0 {
		b.WriteString("## Manual attention required (TODO markers in config)\n\n")
		writeFindings(&b, r.Todos)
		b.WriteString("\n")
	}

	if len(r.Warns) > 0 {
		b.WriteString("## Warnings\n\n")
		writeFindings(&b, r.Warns)
		b.WriteString("\n")
	}

	if len(r.Infos) > 0 {
		b.WriteString("## Mechanical changes\n\n")
		writeFindings(&b, r.Infos)
		b.WriteString("\n")
	}

	b.WriteString("## Next steps\n\n")
	b.WriteString("1. Review the rewritten manifests and every TODO(v2migrate) marker.\n")
	b.WriteString("2. Make sure the provider version in required_providers supports the V2 resources, then run terraform init -upgrade.\n")
	b.WriteString("3. Run terraform plan -out=v2-migrate.tfplan and check it only imports and forgets resources, no destroy and no create.\n")
	b.WriteString("4. Run terraform apply v2-migrate.tfplan.\n")
	fmt.Fprintf(&b, "5. Delete %s and run terraform plan again, it must show no changes.\n", r.Migrations)
	b.WriteString("6. If the plan wants to replace an instance interface, move is_default = true to the interface terraform reports as default.\n")

	return b.String()
}
