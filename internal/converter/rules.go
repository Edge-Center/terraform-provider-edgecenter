package converter

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed rules/*.yaml
var rulesFS embed.FS

const (
	OpRename    = "rename"
	OpDrop      = "drop"
	OpTodo      = "todo"
	OpRequire   = "require"
	OpSet       = "set"
	OpSplit     = "split"
	OpEnsureOne = "ensure_one"
	OpExtract   = "extract"
)

type TypePair struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type SplitCase struct {
	Equals *string `yaml:"equals"`
	Absent bool    `yaml:"absent"`
	To     string  `yaml:"to"`
}

type Op struct {
	Op              string            `yaml:"op"`
	Path            string            `yaml:"path"`
	To              string            `yaml:"to"`
	Note            string            `yaml:"note"`
	Value           string            `yaml:"value"`
	IfAbsent        bool              `yaml:"if_absent"`
	WhenEquals      *string           `yaml:"when_equals"`
	By              string            `yaml:"by"`
	Cases           []SplitCase       `yaml:"cases"`
	ElseTo          string            `yaml:"else_to"`
	Attr            string            `yaml:"attr"`
	RankBy          string            `yaml:"rank_by"`
	NewType         string            `yaml:"new_type"`
	ParentRefAttr   string            `yaml:"parent_ref_attr"`
	CopyParentAttrs []string          `yaml:"copy_parent_attrs"`
	ImportID        string            `yaml:"import_id"`
	AttrTodos       map[string]string `yaml:"attr_todos"`
}

type Refs struct {
	Rename map[string]string `yaml:"rename"`
	Warn   map[string]string `yaml:"warn"`
}

type Rules struct {
	Resource TypePair `yaml:"resource"`
	Data     TypePair `yaml:"data"`
	ImportID string   `yaml:"import_id"`
	Ops      []Op     `yaml:"ops"`
	Refs     Refs     `yaml:"refs"`
	DataRefs Refs     `yaml:"data_refs"`

	source string
}

func (o Op) pathParts() []string {
	if o.Path == "" {
		return nil
	}
	return strings.Split(o.Path, ".")
}

func (o Op) validate() error {
	switch o.Op {
	case OpRename:
		if o.Path == "" || o.To == "" {
			return fmt.Errorf("rename needs path and to")
		}
	case OpDrop, OpTodo, OpRequire:
		if o.Path == "" {
			return fmt.Errorf("%s needs path", o.Op)
		}
		if o.Op != OpDrop && o.Note == "" {
			return fmt.Errorf("%s %s needs note", o.Op, o.Path)
		}
	case OpSet:
		if o.Path == "" || o.Value == "" {
			return fmt.Errorf("set needs path and value")
		}
		if !o.IfAbsent {
			return fmt.Errorf("set %s supports only if_absent mode", o.Path)
		}
	case OpSplit:
		if o.Path == "" || o.By == "" || len(o.Cases) == 0 || o.ElseTo == "" || o.Note == "" {
			return fmt.Errorf("split %s needs path, by, cases, else_to and note", o.Path)
		}
		for _, c := range o.Cases {
			if c.To == "" || (c.Equals == nil && !c.Absent) {
				return fmt.Errorf("split %s has an incomplete case", o.Path)
			}
		}
	case OpEnsureOne:
		if o.Path == "" || o.Attr == "" || o.Value == "" || o.Note == "" {
			return fmt.Errorf("ensure_one %s needs path, attr, value and note", o.Path)
		}
	case OpExtract:
		if o.Path == "" || o.NewType == "" || o.ParentRefAttr == "" || o.ImportID == "" || o.Note == "" {
			return fmt.Errorf("extract %s needs path, new_type, parent_ref_attr, import_id and note", o.Path)
		}
	default:
		return fmt.Errorf("unknown op %q", o.Op)
	}

	return nil
}

func (r *Rules) validate() error {
	if r.Resource.From == "" || r.Resource.To == "" {
		return fmt.Errorf("%s: resource.from and resource.to are required", r.source)
	}
	if r.ImportID == "" {
		return fmt.Errorf("%s: import_id is required", r.source)
	}
	if (r.Data.From == "") != (r.Data.To == "") {
		return fmt.Errorf("%s: data.from and data.to must be set together", r.source)
	}
	for _, op := range r.Ops {
		if err := op.validate(); err != nil {
			return fmt.Errorf("%s: %w", r.source, err)
		}
	}

	return nil
}

func LoadRules() ([]*Rules, error) {
	return loadRulesFrom(rulesFS, "rules")
}

func loadRulesFrom(fsys fs.FS, dir string) ([]*Rules, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read rules dir: %w", err)
	}
	var out []*Rules
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		raw, err := fs.ReadFile(fsys, dir+"/"+e.Name())
		if err != nil {
			return nil, fmt.Errorf("read rules file: %w", err)
		}
		r := &Rules{source: e.Name()}
		dec := yaml.NewDecoder(strings.NewReader(string(raw)))
		dec.KnownFields(true)
		if err := dec.Decode(r); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if err := r.validate(); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].source < out[j].source })
	if len(out) == 0 {
		return nil, fmt.Errorf("no rule files found")
	}
	byType := map[string]string{}
	for _, r := range out {
		if prev, ok := byType[r.Resource.From]; ok {
			return nil, fmt.Errorf("%s: resource %s already mapped in %s", r.source, r.Resource.From, prev)
		}
		byType[r.Resource.From] = r.source
	}

	return out, nil
}
