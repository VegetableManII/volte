package sip

import (
	"fmt"
	"testing"
)

func TestCSeq(t *testing.T) {
	tests := []struct {
		item       string
		wantResult CSeq
		wantErr    bool
	}{
		{"609 REGISTER", CSeq{609, "REGISTER"}, false},
		{"REGISTER", CSeq{}, true},
		{"609", CSeq{}, true},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			m, err := parseCSeq(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("CSeq error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				str := m.String()
				if str != tt.item {
					t.Errorf("CSeq string = %v, wantString %v", str, tt.item)
				}
			}
		})
	}
}
