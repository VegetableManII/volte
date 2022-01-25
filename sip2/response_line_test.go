package sip

import "testing"

func TestResponseLine(t *testing.T) {
	tests := []struct {
		name    string
		item    string
		wantErr bool
	}{
		{"200", "SIP/2.0 200 OK", false},
		{"481", "SIP/2.0 481 Dialog/Transaction Does Not Exist", false},
		{"NoSip", "200 OK", true},
		{"NoCode", "SIP/2.0 OK", true},
		{"NoReason", "SIP/2.0 200", true},
		{"ErrorCode", "SIP/2.0 CODE OK", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := NewResponseLine(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResponseLine error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if str := rl.String(); str != tt.item {
					t.Errorf("ResponseLine string = %v, wantString %v", str, tt.item)
				}
			}
		})
	}
}
