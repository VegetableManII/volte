package common

import (
	"encoding/binary"
	"testing"
)

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

func TestInit(t *testing.T) {
	data := []byte{1, 0, 0, 37, 51, 52, 53, 54, 55, 56}
	var l uint16
	l = binary.BigEndian.Uint16(data[2:4])
	t.Log(l)
}

func TestIntToUint16(t *testing.T) {
	var a int = 65
	b := uint16(a)
	t.Log(b)
}
