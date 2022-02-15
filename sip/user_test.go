package sip

import (
	"fmt"
	"testing"
)

func TestUser(t *testing.T) {
	tests := []struct {
		item    string
		wantErr bool
	}{
		{"\"1011\"<sip:1011@192.168.0.2>;tag=22365c3a331", false},
		{"\"1011\"<sip:1011@192.168.0.2>", false},
		{"<sip:1010@192.168.0.2>;tag=291589446", false},
		{"<sip:1010@192.168.0.2>", false},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			u, err := parseUser(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("User error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if str := u.String(); str != tt.item {
					t.Errorf("User string = %v, wantString %v", str, tt.item)
				}
			}
		})
	}
}
