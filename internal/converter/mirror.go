package converter

import (
	"math/big"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type node struct {
	name     string
	attr     *hclsyntax.Attribute
	block    *hclsyntax.Block
	children []*node
	consumed bool
}

func (n *node) isBlock() bool { return n.block != nil }

func (n *node) rng() hcl.Range {
	if n.attr != nil {
		return n.attr.SrcRange
	}
	return hcl.RangeBetween(n.block.TypeRange, n.block.CloseBraceRange)
}

func buildTree(body *hclsyntax.Body) []*node {
	out := make([]*node, 0, len(body.Attributes)+len(body.Blocks))
	for _, attr := range body.Attributes {
		out = append(out, &node{name: attr.Name, attr: attr})
	}
	for _, blk := range body.Blocks {
		out = append(out, &node{name: blk.Type, block: blk, children: buildTree(blk.Body)})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].rng().Start.Byte < out[j].rng().Start.Byte
	})

	return out
}

func matchPath(roots []*node, parts []string) []*node {
	cur := roots
	for i, part := range parts {
		var next []*node
		for _, n := range cur {
			if n.consumed || n.name != part {
				continue
			}
			if i == len(parts)-1 {
				next = append(next, n)
				continue
			}
			if n.isBlock() {
				next = append(next, n.children...)
			}
		}
		cur = next
	}

	return cur
}

func (n *node) child(name string) *node {
	for _, c := range n.children {
		if !c.consumed && c.name == name {
			return c
		}
	}
	return nil
}

func literalString(expr hclsyntax.Expression) (string, bool) {
	val, diags := expr.Value(nil)
	if diags.HasErrors() || !val.IsKnown() || val.IsNull() {
		return "", false
	}
	switch val.Type() {
	case cty.String:
		return val.AsString(), true
	case cty.Number:
		bf := val.AsBigFloat()
		return bf.Text('f', -1), true
	case cty.Bool:
		if val.True() {
			return "true", true
		}

		return "false", true
	}

	return "", false
}

func literalRank(expr hclsyntax.Expression) (*big.Float, bool) {
	val, diags := expr.Value(nil)
	if diags.HasErrors() || !val.IsKnown() || val.IsNull() || val.Type() != cty.Number {
		return nil, false
	}
	return val.AsBigFloat(), true
}
