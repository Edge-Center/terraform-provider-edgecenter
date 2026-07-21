package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Options struct {
	Dir            string
	StatePath      string
	MigrationsPath string
}

type Result struct {
	Report  *Report
	Changed map[string][]byte
}

func Run(opts Options) (*Result, error) {
	rules, err := LoadRules()
	if err != nil {
		return nil, err
	}
	resources := map[string]*Rules{}
	datas := map[string]*Rules{}
	v1types := map[string]bool{}
	for _, r := range rules {
		resources[r.Resource.From] = r
		v1types[r.Resource.From] = true
		if r.Data.From != "" {
			datas[r.Data.From] = r
		}
	}

	entries, err := os.ReadDir(opts.Dir)
	if err != nil {
		return nil, fmt.Errorf("read config dir: %w", err)
	}
	rep := &Report{Migrations: filepath.Base(opts.MigrationsPath)}
	var files []*sourceFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".tf.json") {
			rep.warn(name, 0, "", "json configuration files are not converted")
			continue
		}
		if !strings.HasSuffix(name, ".tf") || name == filepath.Base(opts.MigrationsPath) {
			continue
		}
		path := filepath.Join(opts.Dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		f, err := parseSourceFile(name, raw)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })

	names := newNameIndex()
	moduleSeen := false
	for _, f := range files {
		for _, blk := range f.body.Blocks {
			if blk.Type == kindResource && len(blk.Labels) == 2 {
				names.add(blk.Labels[0], blk.Labels[1])
			}
			if blk.Type == "module" {
				moduleSeen = true
			}
		}
	}
	if moduleSeen {
		rep.warn("", 0, "", "module calls are not converted, run the converter in each module directory and adjust the migration block addresses manually")
	}

	var convs []*blockConv
	blockedByFile := map[*sourceFile][]span{}
	for _, f := range files {
		for _, blk := range f.body.Blocks {
			if len(blk.Labels) != 2 {
				continue
			}
			var r *Rules
			var kind string
			switch blk.Type {
			case kindResource:
				if r = resources[blk.Labels[0]]; r != nil {
					kind = kindResource
				}
			case kindData:
				if r = datas[blk.Labels[0]]; r != nil {
					kind = kindData
				}
			}
			if r == nil {
				continue
			}
			c := newBlockConv(f, r, kind, blk, rep, names)
			c.convert()
			convs = append(convs, c)
			blockedByFile[f] = append(blockedByFile[f], c.blocked...)
			rep.Converted = append(rep.Converted, ConvertedItem{
				File:    f.path,
				Line:    blk.TypeRange.Start.Line,
				Kind:    kind,
				OldAddr: c.oldAddr,
				NewAddr: c.newAddr,
			})
		}
	}

	if len(convs) == 0 {
		return &Result{Report: rep, Changed: map[string][]byte{}}, nil
	}

	extracts := map[string]string{}
	for _, c := range convs {
		for _, ex := range c.extractions {
			extracts[extractKey(ex.parentType, ex.parentName, ex.path, ex.idx)] = ex.newType + "." + ex.newName
		}
	}
	for _, f := range files {
		rewriteRefs(f, resources, datas, extracts, blockedByFile[f], rep)
	}

	var state *stateIndex
	if opts.StatePath != "" {
		state, err = loadState(opts.StatePath, v1types)
		if err != nil {
			return nil, err
		}
		for _, addr := range state.modular {
			rep.warn("", 0, addr, "v1 resource lives in a child module, not migrated")
		}
	} else {
		rep.warn("", 0, "", "no state file given, import ids are placeholders, run with -state or fill them manually")
	}

	var migEntries []migrationEntry
	for _, c := range convs {
		if c.kind != kindResource {
			continue
		}
		st := state.lookup(c.rules.Resource.From, c.name)
		migEntries = append(migEntries, migrationEntry{conv: c, state: st, rules: c.rules})
		for _, ex := range c.extractions {
			migEntries = append(migEntries, migrationEntry{conv: c, state: st, extract: ex})
		}
	}

	changed := map[string][]byte{}
	for _, f := range files {
		if len(f.edits) == 0 {
			continue
		}
		out, err := f.apply()
		if err != nil {
			return nil, err
		}
		changed[filepath.Join(opts.Dir, f.path)] = out
	}
	if len(migEntries) > 0 {
		changed[opts.MigrationsPath] = []byte(buildMigrations(migEntries, rep))
	} else {
		rep.Migrations = ""
	}

	stateOnlyWarn(state, resources, convs, rep)

	return &Result{Report: rep, Changed: changed}, nil
}

func WriteResult(res *Result) error {
	paths := make([]string, 0, len(res.Changed))
	for p := range res.Changed {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if err := os.WriteFile(p, res.Changed[p], 0o600); err != nil {
			return fmt.Errorf("write %s: %w", p, err)
		}
	}

	return nil
}

func stateOnlyWarn(state *stateIndex, resources map[string]*Rules, convs []*blockConv, rep *Report) {
	if state == nil {
		return
	}
	inConfig := map[string]bool{}
	for _, c := range convs {
		if c.kind == kindResource {
			inConfig[c.rules.Resource.From+"."+c.name] = true
		}
	}
	var addrs []string
	for addr := range state.byAddr {
		typ := strings.SplitN(addr, ".", 2)[0]
		if resources[typ] != nil && !inConfig[addr] {
			addrs = append(addrs, addr)
		}
	}
	sort.Strings(addrs)
	for _, addr := range addrs {
		rep.warn("", 0, addr, "v1 resource exists in state but not in the configuration, not migrated")
	}
}
