package converter

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type span struct {
	start int
	end   int
}

type extraction struct {
	parentType string
	parentName string
	path       string
	idx        int
	newType    string
	newName    string
	importTmpl string
	file       string
}

type blockConv struct {
	f       *sourceFile
	rules   *Rules
	kind    string
	block   *hclsyntax.Block
	name    string
	oldAddr string
	newAddr string
	tree    []*node
	rep     *Report
	names   *nameIndex
	blocked []span

	extractions []*extraction
}

type nameIndex struct {
	taken map[string]bool
}

func newNameIndex() *nameIndex {
	return &nameIndex{taken: map[string]bool{}}
}

func (ni *nameIndex) add(typ, name string) {
	ni.taken[typ+"."+name] = true
}

func (ni *nameIndex) claim(typ, base, suffix string) string {
	try := func(name string) bool {
		if ni.taken[typ+"."+name] {
			return false
		}
		ni.taken[typ+"."+name] = true
		return true
	}
	if try(base) {
		return base
	}
	if try(base + "_" + suffix) {
		return base + "_" + suffix
	}
	i := 2
	for {
		if name := base + "_" + suffix + strconv.Itoa(i); try(name) {
			return name
		}
		i++
	}
}

const (
	kindResource = "resource"
	kindData     = "data"
)

func newBlockConv(f *sourceFile, rules *Rules, kind string, block *hclsyntax.Block, rep *Report, names *nameIndex) *blockConv {
	name := block.Labels[1]
	oldAddr := block.Labels[0] + "." + name
	var newType string
	if kind == kindData {
		oldAddr = "data." + oldAddr
		newType = rules.Data.To
	} else {
		newType = rules.Resource.To
	}
	newAddr := newType + "." + name
	if kind == kindData {
		newAddr = "data." + newAddr
	}

	return &blockConv{
		f:       f,
		rules:   rules,
		kind:    kind,
		block:   block,
		name:    name,
		oldAddr: oldAddr,
		newAddr: newAddr,
		tree:    buildTree(block.Body),
		rep:     rep,
		names:   names,
	}
}

func (c *blockConv) convert() {
	for _, op := range c.rules.Ops {
		switch op.Op {
		case OpRename:
			c.applyRename(op)
		case OpDrop:
			c.applyDrop(op)
		case OpTodo:
			c.applyTodo(op)
		case OpRequire:
			c.applyRequire(op)
		case OpSet:
			c.applySet(op)
		case OpSplit:
			c.applySplit(op)
		case OpEnsureOne:
			c.applyEnsureOne(op)
		case OpExtract:
			c.applyExtract(op)
		}
	}
	c.renameTypeLabel()
	c.checkIgnoreChanges()
}

func (c *blockConv) renameTypeLabel() {
	newType := c.rules.Resource.To
	if c.kind == kindData {
		newType = c.rules.Data.To
	}
	rng := c.block.LabelRanges[0]
	text := newType
	if c.f.data[rng.Start.Byte] == '"' {
		text = `"` + newType + `"`
	}
	c.f.replace(rng, text)
}

func (c *blockConv) checkIgnoreChanges() {
	for _, n := range c.tree {
		if !n.isBlock() || n.name != "lifecycle" {
			continue
		}
		if a := n.child("ignore_changes"); a != nil {
			c.rep.warn(c.f.path, a.rng().Start.Line, c.oldAddr, "review lifecycle ignore_changes entries, attribute names changed in V2")
		}
	}
}

func (c *blockConv) applyRename(op Op) {
	for _, n := range matchPath(c.tree, op.pathParts()) {
		if n.isBlock() {
			c.f.replace(n.block.TypeRange, op.To)
		} else {
			c.f.replace(n.attr.NameRange, op.To)
		}
		n.name = op.To
	}
}

func (c *blockConv) applyDrop(op Op) {
	for _, n := range matchPath(c.tree, op.pathParts()) {
		c.f.deleteLines(n.rng())
		n.consumed = true
		c.block2span(n)
		if op.Note != "" {
			c.rep.info(c.f.path, n.rng().Start.Line, c.newAddr, op.Path+" removed: "+op.Note)
		}
	}
}

func (c *blockConv) applyTodo(op Op) {
	for _, n := range matchPath(c.tree, op.pathParts()) {
		if op.WhenEquals != nil {
			if n.isBlock() {
				continue
			}
			v, ok := literalString(n.attr.Expr)
			if !ok || v != *op.WhenEquals {
				continue
			}
		}
		c.todoNode(n, op.Note)
	}
}

func (c *blockConv) todoNode(n *node, note string) {
	c.f.commentOut(n.rng(), note)
	n.consumed = true
	c.block2span(n)
	c.rep.todo(c.f.path, n.rng().Start.Line, c.newAddr, n.name+": "+note)
}

func (c *blockConv) block2span(n *node) {
	rng := n.rng()
	c.blocked = append(c.blocked, span{start: rng.Start.Byte, end: rng.End.Byte})
}

func (c *blockConv) parentsOf(parts []string) []*node {
	if len(parts) == 1 {
		return []*node{{name: "", block: c.block, children: c.tree}}
	}
	var out []*node
	for _, n := range matchPath(c.tree, parts[:len(parts)-1]) {
		if n.isBlock() {
			out = append(out, n)
		}
	}
	return out
}

func (c *blockConv) insertIntoBlock(blk *hclsyntax.Block, children []*node, text string) {
	pos := c.f.lineStart(blk.CloseBraceRange.Start.Byte)
	indent := c.f.indentAt(blk.CloseBraceRange.Start.Byte) + "  "
	for _, ch := range children {
		if !ch.consumed {
			indent = c.f.indentAt(ch.rng().Start.Byte)
			break
		}
	}
	c.f.insert(pos, indent+text+"\n")
}

func (c *blockConv) applyRequire(op Op) {
	parts := op.pathParts()
	last := parts[len(parts)-1]
	for _, parent := range c.parentsOf(parts) {
		if parent.child(last) != nil {
			continue
		}
		c.insertIntoBlock(parent.block, parent.children, todoMarker+op.Note)
		c.rep.todo(c.f.path, parent.block.TypeRange.Start.Line, c.newAddr, last+" is missing: "+op.Note)
	}
}

func (c *blockConv) applySet(op Op) {
	parts := op.pathParts()
	last := parts[len(parts)-1]
	for _, parent := range c.parentsOf(parts) {
		if parent.child(last) != nil {
			continue
		}
		c.insertIntoBlock(parent.block, parent.children, last+" = "+op.Value)
		c.rep.info(c.f.path, parent.block.TypeRange.Start.Line, c.newAddr, last+" = "+op.Value+" added")
	}
}

func (c *blockConv) applySplit(op Op) {
	for _, n := range matchPath(c.tree, op.pathParts()) {
		if !n.isBlock() {
			continue
		}
		target, ok := c.splitTarget(op, n)
		if !ok {
			c.todoNode(n, op.Note)
			continue
		}
		c.f.replace(n.block.TypeRange, target)
		n.name = target
	}
}

func (c *blockConv) splitTarget(op Op, n *node) (string, bool) {
	by := n.child(op.By)
	if by == nil || by.isBlock() {
		for _, cs := range op.Cases {
			if cs.Absent {
				return cs.To, true
			}
		}
		return op.ElseTo, true
	}
	v, ok := literalString(by.attr.Expr)
	if !ok {
		return "", false
	}
	for _, cs := range op.Cases {
		if cs.Equals != nil && *cs.Equals == v {
			return cs.To, true
		}
	}

	return op.ElseTo, true
}

func (c *blockConv) applyEnsureOne(op Op) {
	var blocks []*node
	for _, n := range matchPath(c.tree, op.pathParts()) {
		if n.isBlock() {
			blocks = append(blocks, n)
		}
	}
	if len(blocks) == 0 {
		return
	}
	for _, n := range blocks {
		if a := n.child(op.Attr); a != nil {
			if v, ok := literalString(a.attr.Expr); ok && v == op.Value {
				return
			}
			if _, ok := literalString(a.attr.Expr); !ok {
				return
			}
		}
	}
	chosen := blocks[0]
	if op.RankBy != "" {
		for _, n := range blocks[1:] {
			cur, curOK := rankOf(chosen, op.RankBy)
			cand, candOK := rankOf(n, op.RankBy)
			if candOK && (!curOK || cand.Cmp(cur) < 0) {
				chosen = n
			}
		}
	}
	if a := chosen.child(op.Attr); a != nil {
		c.f.replace(a.attr.Expr.Range(), op.Value)
	} else {
		c.insertIntoBlock(chosen.block, chosen.children, op.Attr+" = "+op.Value)
	}
	c.rep.info(c.f.path, chosen.block.TypeRange.Start.Line, c.newAddr, op.Attr+" = "+op.Value+" set: "+op.Note)
}

func rankOf(n *node, attr string) (*big.Float, bool) {
	a := n.child(attr)
	if a == nil || a.isBlock() {
		return nil, false
	}
	return literalRank(a.attr.Expr)
}

func (c *blockConv) applyExtract(op Op) {
	if c.kind != kindResource {
		for _, n := range matchPath(c.tree, op.pathParts()) {
			c.todoNode(n, op.Note)
		}

		return
	}
	counted := false
	for _, n := range c.tree {
		if !n.isBlock() && (n.name == "count" || n.name == "for_each") {
			counted = true
		}
	}
	blocks := matchPath(c.tree, op.pathParts())
	for idx, n := range blocks {
		if !n.isBlock() {
			continue
		}
		if counted {
			note := op.Note + ", cannot extract automatically when the parent uses count or for_each, create the " + op.NewType + " resource manually"
			c.todoNode(n, note)

			continue
		}
		c.extractBlock(op, n, idx)
	}
}

func (c *blockConv) extractBlock(op Op, n *node, idx int) {
	newName := c.names.claim(op.NewType, c.name, op.Path)
	var b strings.Builder
	fmt.Fprintf(&b, "\nresource %q %q {\n", op.NewType, newName)
	for _, attrName := range op.CopyParentAttrs {
		for _, pn := range c.tree {
			if !pn.consumed && !pn.isBlock() && pn.name == attrName {
				fmt.Fprintf(&b, "  %s = %s\n", attrName, c.f.src(pn.attr.Expr.Range()))
			}
		}
	}
	fmt.Fprintf(&b, "  %s = %s.%s.id\n", op.ParentRefAttr, c.rules.Resource.To, c.name)

	todoSpans := map[*hclsyntax.Attribute]string{}
	for _, ch := range n.children {
		if !ch.isBlock() {
			if note, ok := op.AttrTodos[ch.name]; ok {
				todoSpans[ch.attr] = note
			}
		}
	}
	bodyStart := c.f.lineEnd(n.block.OpenBraceRange.End.Byte - 1)
	bodyEnd := c.f.lineStart(n.block.CloseBraceRange.Start.Byte)
	if bodyStart >= bodyEnd {
		for _, ch := range n.children {
			if ch.isBlock() {
				continue
			}
			if note, ok := todoSpans[ch.attr]; ok {
				b.WriteString("  " + todoMarker + note + "\n")
				fmt.Fprintf(&b, "  # %s = %s\n", ch.name, c.f.src(ch.attr.Expr.Range()))
				continue
			}
			fmt.Fprintf(&b, "  %s = %s\n", ch.name, c.f.src(ch.attr.Expr.Range()))
		}
	} else {
		c.writeExtractBody(&b, bodyStart, bodyEnd, todoSpans)
	}
	b.WriteString("}\n")

	insertAt := c.f.lineEnd(c.block.CloseBraceRange.End.Byte - 1)
	c.f.insert(insertAt, b.String())

	c.f.deleteLines(n.rng())
	n.consumed = true
	c.block2span(n)

	tmpl := strings.ReplaceAll(op.ImportID, "{$.", "{"+op.Path+"."+strconv.Itoa(idx)+".")
	c.extractions = append(c.extractions, &extraction{
		parentType: c.rules.Resource.From,
		parentName: c.name,
		path:       op.Path,
		idx:        idx,
		newType:    op.NewType,
		newName:    newName,
		importTmpl: tmpl,
		file:       c.f.path,
	})
	c.rep.Extracted = append(c.rep.Extracted, ExtractedItem{
		File:       c.f.path,
		ParentAddr: c.oldAddr,
		NewAddr:    op.NewType + "." + newName,
	})
	for attr, note := range todoSpans {
		c.rep.todo(c.f.path, attr.SrcRange.Start.Line, op.NewType+"."+newName, attr.Name+": "+note)
	}
}

func (c *blockConv) writeExtractBody(b *strings.Builder, start, end int, todos map[*hclsyntax.Attribute]string) {
	type todoSpan struct {
		span
		note string
	}
	spans := make([]todoSpan, 0, len(todos))
	for attr, note := range todos {
		spans = append(spans, todoSpan{span{attr.SrcRange.Start.Byte, attr.SrcRange.End.Byte}, note})
	}
	pos := start
	for pos < end {
		le := c.f.lineEnd(pos)
		line := string(c.f.data[pos:le])
		trimmed := strings.TrimLeft(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			b.WriteString("\n")
			pos = le
			continue
		}
		contentOff := pos + len(line) - len(trimmed)
		inTodo := false
		for _, ts := range spans {
			if contentOff >= ts.start && contentOff < ts.end {
				if contentOff == ts.start {
					b.WriteString("  " + todoMarker + ts.note + "\n")
				}
				inTodo = true
			}
		}
		if !strings.HasSuffix(trimmed, "\n") {
			trimmed += "\n"
		}
		if inTodo {
			b.WriteString("  # " + trimmed)
		} else {
			b.WriteString("  " + trimmed)
		}
		pos = le
	}
}
