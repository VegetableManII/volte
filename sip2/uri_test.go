package sip

import (
	"testing"
)

func TestURI(t *testing.T) {
	tests := []struct {
		name    string
		item    string
		wantErr bool
	}{
		// DD(DomainDNS)，DH(DomainHostNoPort)，DP(DomainWithPort)
		// UN(UserNormal)，UserP(UserPhone)，UP(UserPassword)，S(SchemeWithS)，A(Arguments)
		{"DP", "sip:192.168.0.2:5060", false},
		{"UN+DH", "sip:alice@192.0.2.4", false},
		{"UN+DD", "sip:alice@atlanta.com", false},
		{"UP+DD+A", "sip:+1-212-555-1212:1234@gateway.com;user=phone", false},
		{"S+UN+DD", "sips:1212@gateway.com", false},
		{"UN+A", "sip:alice;day=Tuesday@atlanta.com", false},
		{"UP+DD+A", "sip:alice:secretword@atlanta.com;transport=tcp", false},
		// TODO {"sip:alice@atlanta.com?subject=project%20x&priority=urgent", false},
		// TODO {"sip:atlanta.com;method=REGISTER?to=alice%40atlanta.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := NewURI(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("URI error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if str := u.String(); str != tt.item {
					t.Errorf("URI string = %v, wantString %v", str, tt.item)
				}
			}
		})
	}
}
