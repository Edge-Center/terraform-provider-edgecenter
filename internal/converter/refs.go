package converter

import (
	"math/big"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type refRewriter struct {
	f         *sourceFile
	resources map[string]*Rules
	datas     map[string]*Rules
	extracts  map[string]string
	rep       *Report
	blocked   []span
	warned    map[string]bool
}

func extractKey(v1type, name, path string, idx int) string {
	return v1type + "." + name + "." + path + "." + strconv.Itoa(idx)
}

func rewriteRefs(f *sourceFile, resources, datas map[string]*Rules, extracts map[string]string, blocked []span, rep *Report) {
	rw := &refRewriter{
		f:         f,
		resources: resources,
		datas:     datas,
		extracts:  extracts,
		rep:       rep,
		blocked:   blocked,
		warned:    map[string]bool{},
	}
	for _, blk := range f.body.Blocks {
		switch blk.Type {
		case "removed", "import", "moved":
			continue
		}
		rw.walkBlock(blk)
	}
	for _, attr := range f.body.Attributes {
		rw.walkExpr(attr.Expr)
	}
}

func (rw *refRewriter) walkBlock(blk *hclsyntax.Block) {
	for _, attr := range blk.Body.Attributes {
		rw.walkExpr(attr.Expr)
	}
	for _, child := range blk.Body.Blocks {
		rw.walkBlock(child)
	}
}

type traversalCollector struct {
	list []*hclsyntax.ScopeTraversalExpr
}

func (t *traversalCollector) Enter(node hclsyntax.Node) hcl.Diagnostics {
	if e, ok := node.(*hclsyntax.ScopeTraversalExpr); ok {
		t.list = append(t.list, e)
	}
	return nil
}

func (t *traversalCollector) Exit(hclsyntax.Node) hcl.Diagnostics { return nil }

func (rw *refRewriter) walkExpr(expr hclsyntax.Expression) {
	col := &traversalCollector{}
	hclsyntax.Walk(expr, col)
	for _, e := range col.list {
		rw.handleTraversal(e.Traversal)
	}
}

func (rw *refRewriter) isBlocked(off int) bool {
	for _, s := range rw.blocked {
		if off >= s.start && off < s.end {
			return true
		}
	}
	return false
}

func (rw *refRewriter) handleTraversal(trav hcl.Traversal) {
	if len(trav) < 2 {
		return
	}
	root, ok := trav[0].(hcl.TraverseRoot)
	if !ok || rw.isBlocked(root.SrcRange.Start.Byte) {
		return
	}
	if rules, found := rw.resources[root.Name]; found {
		nameSeg, okName := trav[1].(hcl.TraverseAttr)
		if okName && rw.rewriteExtractedRef(trav, root, nameSeg) {
			return
		}
		rw.replaceSegment(root.SrcRange, rules.Resource.To)
		addr := rules.Resource.To
		if okName {
			addr += "." + nameSeg.Name
		}
		rw.handleAttrSegment(trav, 2, rules.Refs, addr)

		return
	}
	if root.Name != "data" {
		return
	}
	typeSeg, ok := trav[1].(hcl.TraverseAttr)
	if !ok {
		return
	}
	if rules, found := rw.datas[typeSeg.Name]; found {
		rw.replaceSegment(typeSeg.SrcRange, rules.Data.To)
		addr := "data." + rules.Data.To
		if len(trav) > 2 {
			if nameSeg, okName := trav[2].(hcl.TraverseAttr); okName {
				addr += "." + nameSeg.Name
			}
		}
		rw.handleAttrSegment(trav, 3, rules.DataRefs, addr)
	}
}

func (rw *refRewriter) rewriteExtractedRef(trav hcl.Traversal, root hcl.TraverseRoot, nameSeg hcl.TraverseAttr) bool {
	if len(trav) < 4 || len(rw.extracts) == 0 {
		return false
	}
	pathSeg, ok := trav[2].(hcl.TraverseAttr)
	if !ok {
		return false
	}
	idxSeg, ok := trav[3].(hcl.TraverseIndex)
	if !ok {
		return false
	}
	idx, ok := indexInt(idxSeg)
	if !ok {
		return false
	}
	addr, found := rw.extracts[extractKey(root.Name, nameSeg.Name, pathSeg.Name, idx)]
	if !found {
		return false
	}
	rw.f.replace(hcl.RangeBetween(root.SrcRange, idxSeg.SrcRange), addr)

	return true
}

func indexInt(seg hcl.TraverseIndex) (int, bool) {
	if seg.Key.IsNull() || !seg.Key.IsKnown() || seg.Key.Type() != cty.Number {
		return 0, false
	}
	i, acc := seg.Key.AsBigFloat().Int64()
	if acc != big.Exact {
		return 0, false
	}

	return int(i), true
}

func (rw *refRewriter) handleAttrSegment(trav hcl.Traversal, from int, refs Refs, addr string) {
	for i := from; i < len(trav); i++ {
		seg, ok := trav[i].(hcl.TraverseAttr)
		if !ok {
			continue
		}
		if to, found := refs.Rename[seg.Name]; found {
			rw.replaceSegment(seg.SrcRange, to)
		} else if note, found := refs.Warn[seg.Name]; found {
			rw.warnAt(seg.SrcRange, addr, seg.Name+": "+note)
		}
		return
	}
}

func (rw *refRewriter) replaceSegment(rng hcl.Range, name string) {
	text := name
	if rw.f.data[rng.Start.Byte] == '.' {
		text = "." + name
	}
	rw.f.replace(rng, text)
}

func (rw *refRewriter) warnAt(rng hcl.Range, addr, note string) {
	ls := rw.f.lineStart(rng.Start.Byte)
	key := strconv.Itoa(ls) + "|" + note
	if rw.warned[key] {
		return
	}
	rw.warned[key] = true
	indent := rw.f.indentAt(rng.Start.Byte)
	rw.f.insert(ls, indent+todoMarker+note+"\n")
	rw.rep.todo(rw.f.path, rng.Start.Line, addr, "reference "+note)
}
