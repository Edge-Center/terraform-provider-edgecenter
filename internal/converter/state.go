package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type stateInstance struct {
	IndexKey   any            `json:"index_key"`
	Attributes map[string]any `json:"attributes"`
}

type stateResource struct {
	Module    string          `json:"module"`
	Mode      string          `json:"mode"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Instances []stateInstance `json:"instances"`
}

type stateFile struct {
	Version   int             `json:"version"`
	Resources []stateResource `json:"resources"`
}

type stateIndex struct {
	byAddr  map[string]*stateResource
	modular []string
}

func loadState(path string, v1types map[string]bool) (*stateIndex, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var sf stateFile
	if err := dec.Decode(&sf); err != nil {
		return nil, fmt.Errorf("parse state %s: %w", path, err)
	}
	if sf.Version != 4 {
		return nil, fmt.Errorf("unsupported state version %d in %s, run terraform state pull with terraform >= 1.7", sf.Version, path)
	}
	idx := &stateIndex{byAddr: map[string]*stateResource{}}
	for i := range sf.Resources {
		r := &sf.Resources[i]
		if r.Mode != "managed" {
			continue
		}
		if r.Module != "" {
			if v1types[r.Type] {
				idx.modular = append(idx.modular, r.Module+"."+r.Type+"."+r.Name)
			}
			continue
		}
		idx.byAddr[r.Type+"."+r.Name] = r
	}

	return idx, nil
}

func (idx *stateIndex) lookup(typ, name string) *stateResource {
	if idx == nil {
		return nil
	}
	return idx.byAddr[typ+"."+name]
}

func attrValue(attrs map[string]any, path string) (string, bool) {
	var cur any = attrs
	for _, part := range strings.Split(path, ".") {
		switch v := cur.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				return "", false
			}
			cur = next
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(v) {
				return "", false
			}
			cur = v[i]
		default:
			return "", false
		}
	}
	switch v := cur.(type) {
	case string:
		if v == "" {
			return "", false
		}
		return v, true
	case json.Number:
		return v.String(), true
	case bool:
		return strconv.FormatBool(v), true
	case nil:
		return "", false
	}

	return "", false
}

func expandImportID(tmpl string, attrs map[string]any) (string, []string) {
	var missing []string
	var b strings.Builder
	rest := tmpl
	for {
		i := strings.IndexByte(rest, '{')
		if i < 0 {
			b.WriteString(rest)
			break
		}
		j := strings.IndexByte(rest[i:], '}')
		if j < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:i])
		name := rest[i+1 : i+j]
		if attrs != nil {
			if v, ok := attrValue(attrs, name); ok {
				b.WriteString(v)
				rest = rest[i+j+1:]
				continue
			}
		}
		missing = append(missing, name)
		b.WriteString("<" + name + ">")
		rest = rest[i+j+1:]
	}

	return b.String(), missing
}
