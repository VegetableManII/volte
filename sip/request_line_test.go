package sip

import "testing"

func TestRequestLine(t *testing.T) {
	tests := []struct {
		name    string
		item    string
		wantErr bool
	}{
		{"reg", "REGISTER sip:192.168.0.2:5060 SIP/2.0", false},
		{"ivt", "INVITE sip:1010@192.168.0.2 SIP/2.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := NewRequestLine(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequestLine error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if str := rl.String(); str != tt.item {
					t.Errorf("RequestLine string = %v, wantString %v", str, tt.item)
				}
			}
		})
	}
}
