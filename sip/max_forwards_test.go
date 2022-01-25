package sip

import (
	"fmt"
	"testing"
)

func TestMaxForwards(t *testing.T) {
	tests := []struct {
		mf      string
		wantMf  int
		wantErr bool
	}{
		{"40", 39, false},
		{"1", 0, false},
		{"0", 0, false},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			item, err := parseMaxForwards(tt.mf)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxForward error = %v, wantErr %v", err, tt.wantErr)
			}
			item.Reduce()
			if item.value != tt.wantMf {
				t.Errorf("MaxForward reduce mf = %v, wantMf %v", item.value, tt.wantMf)
			}
			item.Reset()
			if item.value != MaxForwardsCount {
				t.Errorf("MaxForward reset mf = %v, wantMf %v", item.value, MaxForwardsCount)
			}
		})
	}
}
