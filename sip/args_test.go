package sip

import (
	"fmt"
	"testing"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		item   string
		result Args
	}{
		{"", Args{}},
		{";tag=22265bf4970", NewArgs(map[string]string{"tag": "22265bf4970"})},
		{";rport", NewArgs(map[string]string{"rport": ""})},
		{";branch=22265bf4970;rport", NewArgs(map[string]string{"branch": "22265bf4970", "rport": ""})},
	}
	for i, _ := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			// h := parseArgs(tt.item)
			// for k, v := range h {
			// 	if rv, ok := tt.result[k]; !ok || v != rv {
			// 		t.Errorf("Args key = %v, value = %v, want %v", k, v, rv)
			// 	}
			// }
			// for k, v := range tt.result {
			// 	if rv, ok := h[k]; !ok || v != rv {
			// 		t.Errorf("Args key = %v, value = %v, want %v", k, rv, v)
			// 	}
			// }
			// if str := h.String(); str != tt.item {
			// 	t.Errorf("Args string = %v, wantString %v", str, tt.item)
			// }
		})
	}
}
