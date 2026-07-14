package tfutil

import (
	"reflect"
	"testing"
)

func TestGetOptByName(t *testing.T) {
	t.Parallel()

	opt := map[string]interface{}{"enabled": true, "value": "3600s"}

	cases := []struct {
		name   string
		fields map[string]interface{}
		key    string
		want   map[string]interface{}
		wantOK bool
	}{
		{
			name:   "present",
			fields: map[string]interface{}{"browser_cache_settings": []interface{}{opt}},
			key:    "browser_cache_settings",
			want:   opt,
			wantOK: true,
		},
		{
			name:   "missing key",
			fields: map[string]interface{}{},
			key:    "browser_cache_settings",
			wantOK: false,
		},
		{
			name:   "empty container",
			fields: map[string]interface{}{"browser_cache_settings": []interface{}{}},
			key:    "browser_cache_settings",
			wantOK: false,
		},
		{
			name:   "container is not a list",
			fields: map[string]interface{}{"browser_cache_settings": "nope"},
			key:    "browser_cache_settings",
			wantOK: false,
		},
		{
			name:   "element is not a map",
			fields: map[string]interface{}{"browser_cache_settings": []interface{}{"nope"}},
			key:    "browser_cache_settings",
			wantOK: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := GetOptByName(tt.fields, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("GetOptByName() ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantOK && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetOptByName() = %v, want %v", got, tt.want)
			}
			if !tt.wantOK && got != nil {
				t.Fatalf("GetOptByName() = %v, want nil", got)
			}
		})
	}
}
