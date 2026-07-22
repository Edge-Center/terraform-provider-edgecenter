package converter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

type schemaTree struct {
	attrs  map[string]*schema.Schema
	blocks map[string]*schemaTree
}

func buildSchemaTree(m map[string]*schema.Schema) *schemaTree {
	t := &schemaTree{attrs: map[string]*schema.Schema{}, blocks: map[string]*schemaTree{}}
	for name, s := range m {
		if pureComputed(s) {
			continue
		}
		if res, ok := s.Elem.(*schema.Resource); ok && (s.Type == schema.TypeList || s.Type == schema.TypeSet) {
			t.blocks[name] = buildSchemaTree(res.Schema)
			continue
		}
		t.attrs[name] = s
	}

	return t
}

func (t *schemaTree) clone() *schemaTree {
	c := &schemaTree{attrs: map[string]*schema.Schema{}, blocks: map[string]*schemaTree{}}
	for k, v := range t.attrs {
		c.attrs[k] = v
	}
	for k, v := range t.blocks {
		c.blocks[k] = v.clone()
	}
	return c
}

func (t *schemaTree) resolveParent(parts []string) (*schemaTree, error) {
	cur := t
	for _, p := range parts {
		next, ok := cur.blocks[p]
		if !ok {
			return nil, fmt.Errorf("block %s not found", p)
		}
		cur = next
	}

	return cur, nil
}

func settable(s *schema.Schema) bool {
	return s.Required || s.Optional
}

func (t *schemaTree) hasSettableAttr(name string) bool {
	s, ok := t.attrs[name]
	return ok && settable(s)
}

func pureComputed(s *schema.Schema) bool {
	return s.Computed && !s.Optional && !s.Required
}

func simulateOps(t *testing.T, source string, ops []Op, cur, v2 *schemaTree, registry map[string]*schema.Resource) {
	t.Helper()
	fail := func(op Op, format string, args ...any) {
		t.Errorf("%s: op %s %s: %s", source, op.Op, op.Path, fmt.Sprintf(format, args...))
	}
	v2parent := func(op Op, parts []string) *schemaTree {
		p, err := v2.resolveParent(parts)
		if err != nil {
			fail(op, "v2 side: %v", err)
			return nil
		}
		return p
	}
	for _, op := range ops {
		parts := op.pathParts()
		last := parts[len(parts)-1]
		parent, err := cur.resolveParent(parts[:len(parts)-1])
		if err != nil {
			fail(op, "v1 side: %v", err)
			continue
		}
		switch op.Op {
		case OpRename:
			if blk, ok := parent.blocks[last]; ok {
				delete(parent.blocks, last)
				parent.blocks[op.To] = blk
				if p2 := v2parent(op, parts[:len(parts)-1]); p2 != nil {
					if _, ok := p2.blocks[op.To]; !ok {
						fail(op, "v2 has no block %s", op.To)
					}
				}
				continue
			}
			s, ok := parent.attrs[last]
			if !ok {
				fail(op, "v1 has no attribute %s", last)
				continue
			}
			delete(parent.attrs, last)
			parent.attrs[op.To] = s
			if p2 := v2parent(op, parts[:len(parts)-1]); p2 != nil {
				if !p2.hasSettableAttr(op.To) {
					fail(op, "v2 has no settable attribute %s", op.To)
				}
			}
		case OpDrop, OpTodo:
			if _, ok := parent.blocks[last]; ok {
				delete(parent.blocks, last)
				continue
			}
			if _, ok := parent.attrs[last]; !ok {
				fail(op, "v1 has no %s", last)
				continue
			}
			delete(parent.attrs, last)
		case OpRequire:
			if p2 := v2parent(op, parts[:len(parts)-1]); p2 != nil {
				if s, ok := p2.attrs[last]; !ok || !s.Required {
					fail(op, "%s is not a required v2 attribute", last)
				}
			}
		case OpSet:
			if p2 := v2parent(op, parts[:len(parts)-1]); p2 != nil {
				if !p2.hasSettableAttr(last) {
					fail(op, "v2 has no settable attribute %s", last)
				}
			}
		case OpSplit:
			blk, ok := parent.blocks[last]
			if !ok {
				fail(op, "v1 has no block %s", last)
				continue
			}
			delete(parent.blocks, last)
			targets := map[string]bool{op.ElseTo: true}
			for _, cs := range op.Cases {
				targets[cs.To] = true
			}
			for target := range targets {
				parent.blocks[target] = blk.clone()
				if _, ok := v2.blocks[target]; !ok {
					fail(op, "v2 has no block %s", target)
				}
			}
		case OpEnsureOne:
			blk, ok := parent.blocks[last]
			if !ok {
				fail(op, "v1 has no block %s", last)
				continue
			}
			if op.RankBy != "" {
				if _, ok := blk.attrs[op.RankBy]; !ok {
					fail(op, "rank_by %s not in v1 block", op.RankBy)
				}
			}
			if p2, err := v2.resolveParent(parts); err != nil {
				fail(op, "v2 side: %v", err)
			} else if !p2.hasSettableAttr(op.Attr) {
				fail(op, "v2 block has no settable attribute %s", op.Attr)
			}
		case OpExtract:
			blk, ok := parent.blocks[last]
			if !ok {
				fail(op, "v1 has no block %s", last)
				continue
			}
			delete(parent.blocks, last)
			target, ok := registry[op.NewType]
			if !ok {
				fail(op, "resource %s not registered", op.NewType)
				continue
			}
			tt := buildSchemaTree(target.Schema)
			if !tt.hasSettableAttr(op.ParentRefAttr) {
				fail(op, "%s has no settable attribute %s", op.NewType, op.ParentRefAttr)
			}
			for _, a := range op.CopyParentAttrs {
				if !cur.hasSettableAttr(a) && !tt.hasSettableAttr(a) {
					fail(op, "copy attr %s missing", a)
				}
			}
			for name := range op.AttrTodos {
				if _, ok := blk.attrs[name]; !ok {
					fail(op, "attr_todos entry %s not in v1 block", name)
				}
			}
			for name, s := range blk.attrs {
				if pureComputed(s) {
					continue
				}
				if _, todo := op.AttrTodos[name]; todo {
					continue
				}
				if !tt.hasSettableAttr(name) {
					fail(op, "v1 block attribute %s has no settable counterpart in %s", name, op.NewType)
				}
			}
		}
	}
}

func checkRemaining(t *testing.T, source, where string, cur, v2 *schemaTree, path string) {
	t.Helper()
	for name, s := range cur.attrs {
		if pureComputed(s) {
			continue
		}
		full := strings.TrimPrefix(path+"."+name, ".")
		if !v2.hasSettableAttr(name) {
			t.Errorf("%s: %s: v1 attribute %s has no settable v2 counterpart and no rule handles it", source, where, full)
		}
	}
	for name, blk := range cur.blocks {
		full := strings.TrimPrefix(path+"."+name, ".")
		v2blk, ok := v2.blocks[name]
		if !ok {
			t.Errorf("%s: %s: v1 block %s has no v2 counterpart and no rule handles it", source, where, full)
			continue
		}
		checkRemaining(t, source, where, blk, v2blk, full)
	}
}

func TestRulesMatchProviderSchemas(t *testing.T) {
	t.Parallel()
	p := provider.Provider()
	rules, err := LoadRules()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rules {
		v1res, ok := p.ResourcesMap[r.Resource.From]
		if !ok {
			t.Fatalf("%s: resource %s not registered", r.source, r.Resource.From)
		}
		v2res, ok := p.ResourcesMap[r.Resource.To]
		if !ok {
			t.Fatalf("%s: resource %s not registered", r.source, r.Resource.To)
		}
		if v2res.Importer == nil {
			t.Errorf("%s: resource %s has no importer, state migration is impossible", r.source, r.Resource.To)
		}
		cur := buildSchemaTree(v1res.Schema)
		v2 := buildSchemaTree(v2res.Schema)
		simulateOps(t, r.source, r.Ops, cur, v2, p.ResourcesMap)
		checkRemaining(t, r.source, "resource "+r.Resource.From, cur, v2, "")

		if r.Data.From == "" {
			continue
		}
		v1data, ok := p.DataSourcesMap[r.Data.From]
		if !ok {
			t.Fatalf("%s: data source %s not registered", r.source, r.Data.From)
		}
		v2data, ok := p.DataSourcesMap[r.Data.To]
		if !ok {
			t.Fatalf("%s: data source %s not registered", r.source, r.Data.To)
		}
		curD := buildSchemaTree(v1data.Schema)
		v2D := buildSchemaTree(v2data.Schema)
		simulateDataOps(t, r.source, r.Ops, curD, v2D)
		checkRemaining(t, r.source, "data "+r.Data.From, curD, v2D, "")
	}
}

func simulateDataOps(t *testing.T, source string, ops []Op, cur, v2 *schemaTree) {
	t.Helper()
	for _, op := range ops {
		parts := op.pathParts()
		last := parts[len(parts)-1]
		parent, err := cur.resolveParent(parts[:len(parts)-1])
		if err != nil {
			continue
		}
		switch op.Op {
		case OpRename:
			if s, ok := parent.attrs[last]; ok {
				delete(parent.attrs, last)
				parent.attrs[op.To] = s
				if p2, err := v2.resolveParent(parts[:len(parts)-1]); err != nil || !p2.hasSettableAttr(op.To) {
					t.Errorf("%s: data op rename %s: v2 data source has no settable attribute %s", source, op.Path, op.To)
				}
			}
			if blk, ok := parent.blocks[last]; ok {
				delete(parent.blocks, last)
				parent.blocks[op.To] = blk
			}
		case OpDrop, OpTodo:
			delete(parent.attrs, last)
			delete(parent.blocks, last)
		case OpSet:
			if len(parts) == 1 {
				if !v2.hasSettableAttr(last) {
					t.Errorf("%s: data op set %s: v2 data source has no settable attribute %s", source, op.Path, last)
				}
			}
		case OpSplit, OpExtract:
			delete(parent.blocks, last)
		}
	}
}
