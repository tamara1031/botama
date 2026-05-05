package config

import (
	"testing"
)

func TestParseChannels(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want map[string]string
	}{
		{"empty", "", map[string]string{}},
		{"single", "alerts=111", map[string]string{"alerts": "111"}},
		{"multiple", "alerts=111,deploys=222", map[string]string{"alerts": "111", "deploys": "222"}},
		{"spaces", " alerts = 111 , deploys = 222 ", map[string]string{"alerts": "111", "deploys": "222"}},
		{"skip malformed", "alerts=111,bad,=noid,noname=", map[string]string{"alerts": "111"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseChannels(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("len: got %d, want %d (got=%v)", len(got), len(tc.want), got)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}
