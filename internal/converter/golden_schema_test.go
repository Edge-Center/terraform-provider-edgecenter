package converter

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/Edge-Center/terraform-provider-edgecenter/edgecenter/provider"
)

var metaArgs = map[string]bool{
	"count":      true,
	"for_each":   true,
	"provider":   true,
	"depends_on": true,
}

var metaBlocks = map[string]bool{
	"lifecycle": true,
	"timeouts":  true,
	"dynamic":   true,
}

func validateBody(t *testing.T, file, addr string, body *hclsyntax.Body, tree *schemaTree, src []byte, topLevel bool) {
	t.Helper()
	hasTodo := strings.Contains(string(src[body.SrcRange.Start.Byte:body.EndRange.End.Byte]), todoMarker)
	for name := range body.Attributes {
		if topLevel && metaArgs[name] {
			continue
		}
		if !tree.hasSettableAttr(name) {
			t.Errorf("%s: %s: attribute %s is not settable in the provider schema", file, addr, name)
		}
	}
	for _, blk := range body.Blocks {
		if metaBlocks[blk.Type] {
			continue
		}
		sub, ok := tree.blocks[blk.Type]
		if !ok {
			t.Errorf("%s: %s: block %s is not in the provider schema", file, addr, blk.Type)
			continue
		}
		validateBody(t, file, addr, blk.Body, sub, src, false)
	}
	if hasTodo {
		return
	}
	for name, s := range tree.attrs {
		if !s.Required {
			continue
		}
		if _, ok := body.Attributes[name]; !ok {
			t.Errorf("%s: %s: required attribute %s is missing", file, addr, name)
		}
	}
}

func TestGoldenOutputsMatchProviderSchemas(t *testing.T) {
	t.Parallel()
	p := provider.Provider()
	for _, name := range []string{"basic", "edge", "nostate"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := runFixture(t, name)
			for file, content := range got {
				if !strings.HasSuffix(file, ".tf") || file == "v2-migrate.tf" {
					continue
				}
				f, diags := hclsyntax.ParseConfig(content, file, hcl.InitialPos)
				if diags.HasErrors() {
					t.Fatalf("%s: %s", file, diags.Error())
				}
				for _, blk := range f.Body.(*hclsyntax.Body).Blocks {
					if len(blk.Labels) != 2 {
						continue
					}
					reg := p.ResourcesMap
					if blk.Type == "data" {
						reg = p.DataSourcesMap
					} else if blk.Type != "resource" {
						continue
					}
					def, ok := reg[blk.Labels[0]]
					if !ok {
						t.Errorf("%s: type %s is not registered in the provider", file, blk.Labels[0])
						continue
					}
					addr := blk.Labels[0] + "." + blk.Labels[1]
					validateBody(t, file, addr, blk.Body, buildSchemaTree(def.Schema), content, true)
				}
			}
		})
	}
}
