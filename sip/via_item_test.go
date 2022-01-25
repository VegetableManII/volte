package sip

import (
	"fmt"
	"testing"
)

func TestVia(t *testing.T) {
	tests := []struct {
		item       string
		wantResult Via
		wantErr    bool
	}{
		{
			"SIP/2.0/UDP 192.168.0.100:43188;branch=z9hG4bK111643fe9a9f389667c5e7d8873;rport",
			Via{"SIP/2.0", "UDP", "192.168.0.100:43188",
				NewArgs(map[string]string{"branch": "z9hG4bK111643fe9a9f389667c5e7d8873", "rport": ""}),
			}, false,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v, err := parseVia(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("Via error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				str := v.String()
				if str != tt.item {
					t.Errorf("Via string = %v, wantString %v", str, tt.item)
				}
			}
		})
	}
}
