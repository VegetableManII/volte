package common

import "testing"

func TestStrLineUnmarshal(t *testing.T) {
	data := map[string]string{
		"imsi": "2123113243",
		"XRES": "asf1a35sdg14f32afs",
		"RAND": "32415341531",
	}
	s := StrLineMarshal(data)
	t.Log(s)
	m := StrLineUnmarshal([]byte(s))
	t.Log(m)
}

func TestStrLineUnMarshalRaw(t *testing.T) {
	s := "a=b\r\nc=d\r\ne=f"
	m := StrLineUnmarshal([]byte(s))
	t.Log(m)
}
