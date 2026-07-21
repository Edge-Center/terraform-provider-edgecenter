package converter

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestExpandImportID(t *testing.T) {
	t.Parallel()
	attrs := map[string]any{
		"id":         "uuid-1",
		"project_id": json.Number("1"),
		"region_id":  json.Number("2"),
		"listener": []any{
			map[string]any{"id": "lsn-1"},
		},
		"empty": "",
	}
	cases := []struct {
		tmpl    string
		want    string
		missing []string
	}{
		{"{project_id}:{region_id}:{id}", "1:2:uuid-1", nil},
		{"{project_id}:{region_id}:{listener.0.id}:{id}", "1:2:lsn-1:uuid-1", nil},
		{"reseller:{reseller_id}", "reseller:<reseller_id>", []string{"reseller_id"}},
		{"{empty}:{id}", "<empty>:uuid-1", []string{"empty"}},
		{"{listener.5.id}", "<listener.5.id>", []string{"listener.5.id"}},
	}
	for _, c := range cases {
		got, missing := expandImportID(c.tmpl, attrs)
		if got != c.want || !reflect.DeepEqual(missing, c.missing) {
			t.Errorf("expandImportID(%q) = %q %v, want %q %v", c.tmpl, got, missing, c.want, c.missing)
		}
	}
	if got, missing := expandImportID("{a}:{b}", nil); got != "<a>:<b>" || len(missing) != 2 {
		t.Errorf("nil attrs: got %q %v", got, missing)
	}
}

func TestRenderIndexKey(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{json.Number("0"), "[0]"},
		{json.Number("12"), "[12]"},
		{"a", `["a"]`},
	}
	for _, c := range cases {
		if got := renderIndexKey(c.in); got != c.want {
			t.Errorf("renderIndexKey(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNameIndexClaim(t *testing.T) {
	t.Parallel()
	ni := newNameIndex()
	ni.add("edgecenter_lblistener", "lb")
	if got := ni.claim("edgecenter_lblistener", "lb", "listener"); got != "lb_listener" {
		t.Errorf("claim = %q, want lb_listener", got)
	}
	if got := ni.claim("edgecenter_lblistener", "lb", "listener"); got != "lb_listener2" {
		t.Errorf("second claim = %q, want lb_listener2", got)
	}
	if got := ni.claim("edgecenter_lblistener", "other", "listener"); got != "other" {
		t.Errorf("claim = %q, want other", got)
	}
}

func TestCommentOutPreservesIndent(t *testing.T) {
	t.Parallel()
	src := []byte("resource \"edgecenter_instance\" \"x\" {\n  volume {\n    boot_index = 0\n  }\n}\n")
	f, err := parseSourceFile("t.tf", src)
	if err != nil {
		t.Fatal(err)
	}
	f.commentOut(f.body.Blocks[0].Body.Blocks[0].Range(), "note")
	out, err := f.apply()
	if err != nil {
		t.Fatal(err)
	}
	want := "resource \"edgecenter_instance\" \"x\" {\n  # TODO(v2migrate): note\n  # volume {\n    # boot_index = 0\n  # }\n}\n"
	if string(out) != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestLoadStateFiltersModules(t *testing.T) {
	t.Parallel()
	idx, err := loadState("testdata/edge/in/terraform.tfstate", map[string]bool{"edgecenter_instance": true})
	if err != nil {
		t.Fatal(err)
	}
	if idx.lookup("edgecenter_instance", "workers") == nil {
		t.Error("workers not found in state index")
	}
	if idx.lookup("edgecenter_instance", "inmodule") != nil {
		t.Error("module resource must not be indexed at root")
	}
	if len(idx.modular) != 1 || idx.modular[0] != "module.net.edgecenter_instance.inmodule" {
		t.Errorf("modular = %v", idx.modular)
	}
}

func TestLoadRulesEmbedded(t *testing.T) {
	t.Parallel()
	rules, err := LoadRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rule files, got %d", len(rules))
	}
	byFrom := map[string]bool{}
	for _, r := range rules {
		byFrom[r.Resource.From] = true
	}
	for _, typ := range []string{"edgecenter_instance", "edgecenter_loadbalancer", "edgecenter_reseller_images"} {
		if !byFrom[typ] {
			t.Errorf("missing rules for %s", typ)
		}
	}
}
