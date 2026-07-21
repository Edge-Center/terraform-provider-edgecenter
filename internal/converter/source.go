package converter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

const todoMarker = "# TODO(v2migrate): "

type edit struct {
	start int
	end   int
	text  string
}

type sourceFile struct {
	path  string
	data  []byte
	body  *hclsyntax.Body
	edits []edit
}

func parseSourceFile(path string, data []byte) (*sourceFile, error) {
	f, diags := hclsyntax.ParseConfig(data, path, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse %s: %s", path, diags.Error())
	}
	return &sourceFile{path: path, data: data, body: f.Body.(*hclsyntax.Body)}, nil
}

func (s *sourceFile) src(rng hcl.Range) string {
	return string(s.data[rng.Start.Byte:rng.End.Byte])
}

func (s *sourceFile) replace(rng hcl.Range, text string) {
	s.edits = append(s.edits, edit{start: rng.Start.Byte, end: rng.End.Byte, text: text})
}

func (s *sourceFile) insert(off int, text string) {
	s.edits = append(s.edits, edit{start: off, end: off, text: text})
}

func (s *sourceFile) lineStart(off int) int {
	for off > 0 && s.data[off-1] != '\n' {
		off--
	}
	return off
}

func (s *sourceFile) lineEnd(off int) int {
	for off < len(s.data) && s.data[off] != '\n' {
		off++
	}
	if off < len(s.data) {
		off++
	}
	return off
}

func (s *sourceFile) indentAt(off int) string {
	start := s.lineStart(off)
	end := start
	for end < len(s.data) && (s.data[end] == ' ' || s.data[end] == '\t') {
		end++
	}
	return string(s.data[start:end])
}

func (s *sourceFile) onlyNodeOnLines(start, end int) bool {
	ls := s.lineStart(start)
	for i := ls; i < start; i++ {
		if s.data[i] != ' ' && s.data[i] != '\t' {
			return false
		}
	}
	for i := end; i < len(s.data); i++ {
		c := s.data[i]
		if c == '\n' {
			break
		}
		if c != ' ' && c != '\t' && c != ',' {
			return false
		}
	}

	return true
}

func (s *sourceFile) deleteLines(rng hcl.Range) {
	start, end := rng.Start.Byte, rng.End.Byte
	if s.onlyNodeOnLines(start, end) {
		s.edits = append(s.edits, edit{start: s.lineStart(start), end: s.lineEnd(end)})
		return
	}
	s.edits = append(s.edits, edit{start: start, end: end})
}

func (s *sourceFile) commentOut(rng hcl.Range, note string) {
	start := s.lineStart(rng.Start.Byte)
	end := rng.End.Byte
	if le := s.lineEnd(end); le > end && strings.TrimSpace(string(s.data[end:le])) == "" {
		end = le
	}
	indent := s.indentAt(rng.Start.Byte)
	var b strings.Builder
	b.WriteString(indent + todoMarker + note + "\n")
	seg := string(s.data[start:end])
	if !strings.HasSuffix(seg, "\n") {
		seg += "\n"
	}
	for _, line := range strings.SplitAfter(seg, "\n") {
		if line == "" {
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			b.WriteString(line)
			continue
		}
		b.WriteString(line[:len(line)-len(trimmed)] + "# " + trimmed)
	}
	s.edits = append(s.edits, edit{start: start, end: end, text: b.String()})
}

func (s *sourceFile) apply() ([]byte, error) {
	edits := append([]edit(nil), s.edits...)
	sort.SliceStable(edits, func(i, j int) bool {
		if edits[i].start != edits[j].start {
			return edits[i].start < edits[j].start
		}
		return edits[i].end < edits[j].end
	})
	for i := 1; i < len(edits); i++ {
		if edits[i].start < edits[i-1].end {
			return nil, fmt.Errorf("%s: conflicting edits at byte %d", s.path, edits[i].start)
		}
	}
	var b strings.Builder
	pos := 0
	for _, e := range edits {
		b.Write(s.data[pos:e.start])
		b.WriteString(e.text)
		pos = e.end
	}
	b.Write(s.data[pos:])
	out := []byte(b.String())
	if _, diags := hclsyntax.ParseConfig(out, s.path, hcl.InitialPos); diags.HasErrors() {
		return nil, fmt.Errorf("internal error: converted %s does not parse: %s", s.path, diags.Error())
	}

	return out, nil
}
