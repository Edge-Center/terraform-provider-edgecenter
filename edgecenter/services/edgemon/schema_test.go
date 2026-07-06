package edgemon

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Edge-Center/edgecenteredgemon-go/statuspage"
)

func TestBoolToInt(t *testing.T) {
	t.Parallel()
	if got := boolToInt(true); got != 1 {
		t.Fatalf("boolToInt(true) = %d, want 1", got)
	}
	if got := boolToInt(false); got != 0 {
		t.Fatalf("boolToInt(false) = %d, want 0", got)
	}
}

func TestIntToBool(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   float64
		want bool
	}{
		{"one is true", 1, true},
		{"zero is false", 0, false},
		{"two is false", 2, false},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := intToBool(tt.in); got != tt.want {
				t.Fatalf("intToBool(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandIntList(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []interface{}
		want []int
	}{
		{"empty", []interface{}{}, []int{}},
		{"values", []interface{}{1, 2, 3}, []int{1, 2, 3}},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := expandIntList(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expandIntList(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandStatusPageChecks(t *testing.T) {
	t.Parallel()
	in := []interface{}{
		map[string]interface{}{"check_id": 7},
		map[string]interface{}{"check_id": 9},
	}
	if got := expandStatusPageChecks(in); !reflect.DeepEqual(got, []int{7, 9}) {
		t.Fatalf("expandStatusPageChecks() = %v, want [7 9]", got)
	}
}

func TestFlattenStatusPageChecks(t *testing.T) {
	t.Parallel()
	in := []statuspage.Checks{{CheckID: 7}, {CheckID: 9}}
	want := []map[string]interface{}{
		{"check_id": 7},
		{"check_id": 9},
	}
	if got := flattenStatusPageChecks(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("flattenStatusPageChecks() = %v, want %v", got, want)
	}
}

func TestStatusPageChecksRoundTrip(t *testing.T) {
	t.Parallel()
	flat := flattenStatusPageChecks([]statuspage.Checks{{CheckID: 1}, {CheckID: 2}})
	raw := make([]interface{}, len(flat))
	for i, m := range flat {
		raw[i] = m
	}
	if got := expandStatusPageChecks(raw); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("round trip = %v, want [1 2]", got)
	}
}

func TestIsNotFoundErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"http 404", errors.New("api returned 404"), true},
		{"not found text", errors.New("resource Not Found"), true},
		{"other", errors.New("internal server error"), false},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isNotFoundErr(tt.err); got != tt.want {
				t.Fatalf("isNotFoundErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
