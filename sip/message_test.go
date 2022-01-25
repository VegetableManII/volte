package sip

import (
	"fmt"
	"strings"
	"testing"
)

func TestMessageBase(t *testing.T) {
	tests := []struct {
		item    string
		wantErr bool
	}{
		{`REGISTER sip:192.168.0.2:5060 SIP/2.0
			Via: SIP/2.0/UDP 192.168.0.102:5060;branch=z9hG4bK-172441109;rport
			From: "1010"<sip:1010@192.168.0.2>;tag=943080490
			To: <sip:1010@192.168.0.2>
			Call-ID: 012022699881-214NTIL@192.168.0.102
			CSeq: 1 REGISTER
			Expires: 600
			Allow: INVITE,CANCEL,ACK,BYE,NOTIFY,REFER,OPTIONS,INFO,MESSAGE,UPDATE,PRACK
			Max-Forwards: 70
			User-Agent: NTIL214 SIP-Test UA
			Content-Length:0
			Contact: <sip:1010@192.168.0.102:5060>

		`, false},
		{`INVITE sip:1010@192.168.0.2 SIP/2.0
			Via: SIP/2.0/UDP 192.168.0.100:45508;branch=z9hG4bK1158493fb1e25c83f14f5e7ee368;rport
			From: "1011" <sip:1011@192.168.0.2>;tag=22365c3a331
			To: "1010" <sip:1010@192.168.0.2>
			Call-ID: 93fb1e2-cd303fba92f18-6ebf2a6@192.168.0.100
			CSeq: 1 INVITE
			Contact: <sip:1011@192.168.0.100:45508>
			User-Agent: VaxPhoneSDK(iOS)/v6.8
			Max-Forwards: 70
			Allow: ACK, INFO, BYE, CANCEL, INVITE, NOTIFY, OPTIONS, REFER, REGISTER, SUBSCRIBE, MESSAGE, PRACK
			Content-Type: application/protocol/sdp
			Content-Length:   386

			v=0
			o=1011 244548 244548 IN IP4 192.168.0.100
			s=VaxSoft
			c=IN IP4 192.168.0.100
			t=0 0
			m=audio 63978 RTP/AVP 0 8 18 3 97 101
			a=rtpmap:0 PCMU/8000
			a=rtpmap:8 PCMA/8000
			a=rtpmap:18 G729/8000
			a=rtpmap:3 GSM/8000
			a=rtpmap:97 iLBC/8000
			a=fmtp:18 annexb=no
			a=rtpmap:101 telephone-event/8000
			a=fmtp:101 0-16
			a=sendrecv
			m=video 39198 RTP/AVP 96
			a=rtpmap:96 VP8/90000
			a=sendrecv

		`, false},
		{`SIP/2.0 180 Ringing
			Via: SIP/2.0/UDP 192.168.0.100:45508;branch=z9hG4bK1158493fb1e25c83f14f5e7ee368;rport
			From: "1011"<sip:1011@192.168.0.2>;tag=22365c3a331
			To: <sip:1010@192.168.0.2>;tag=291589446
			Call-ID: 93fb1e2-cd303fba92f18-6ebf2a6@192.168.0.100
			CSeq: 1 INVITE
			Allow: INVITE,CANCEL,ACK,BYE,NOTIFY,REFER,OPTIONS,INFO,MESSAGE,UPDATE,PRACK
			Max-Forwards: 70
			User-Agent: NTIL214 SIP-Test UA
			Content-Length:0
			Contact: <sip:1010@192.168.0.102:5060>

		`, false},
		{`SIP/2.0 200 OK
			Via: SIP/2.0/UDP 192.168.0.100:45508;branch=z9hG4bK1158493fb1e25c83f14f5e7ee368;rport
			From: "1011"<sip:1011@192.168.0.2>;tag=22365c3a331
			To: <sip:1010@192.168.0.2>;tag=291589446
			Call-ID: 93fb1e2-cd303fba92f18-6ebf2a6@192.168.0.100
			CSeq: 1 INVITE
			Allow: INVITE,CANCEL,ACK,BYE,NOTIFY,REFER,OPTIONS,INFO,MESSAGE,UPDATE,PRACK
			Max-Forwards: 70
			User-Agent: NTIL214 SIP-Test UA
			Content-Type: application/protocol/sdp
			Content-Length:248
			Contact: <sip:1010@192.168.0.102:5060>

			v=0
			o=1010 1405748996 8201 IN IP4 192.168.0.102
			s=SIP Implement by NTIL@NTUT
			c=IN IP4 192.168.0.102
			t=0 0
			m=audio 19998 RTP/AVP 0 101
			a=rtpmap:0 PCMU/8000
			a=rtpmap:101 telephone-event/8000
			a=fmtp:101 0-15
			m=video 0 RTP/AVP 96
			a=sendrecv

		`, false},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			item := strings.ReplaceAll(tt.item, "\n", CRLF)
			_, err := NewMessage(strings.NewReader(item))
			if (err != nil) != tt.wantErr {
				t.Errorf("Message error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
