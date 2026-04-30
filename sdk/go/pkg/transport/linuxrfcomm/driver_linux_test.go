//go:build linux

package linuxrfcomm

import "testing"

func TestParseBDAddr(t *testing.T) {
	got, err := parseBDAddr("AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := [6]byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}
	if got != want {
		t.Fatalf("got %x want %x", got, want)
	}
}

func TestParseBDAddrErrors(t *testing.T) {
	for _, s := range []string{"", "AA:BB:CC:DD:EE", "AA:BB:CC:DD:EE:FFFF", "ZZ:BB:CC:DD:EE:FF", "not-an-address"} {
		if _, err := parseBDAddr(s); err == nil {
			t.Errorf("parseBDAddr(%q) accepted invalid input", s)
		}
	}
}

func TestParseBluetoothctlDevice(t *testing.T) {
	cases := []struct {
		in       string
		ok       bool
		address  string
		name     string
	}{
		{"Device AA:BB:CC:DD:EE:01 stub-pi", true, "AA:BB:CC:DD:EE:01", "stub-pi"},
		{"Device 11:22:33:44:55:66 Pixel 7  ", true, "11:22:33:44:55:66", "Pixel 7"},
		{"Controller AA:BB:CC:DD:EE:00 [default]", false, "", ""},
		{"", false, "", ""},
		{"Device not-a-mac BadName", false, "", ""},
	}
	for _, c := range cases {
		dev, ok := parseBluetoothctlDevice(c.in)
		if ok != c.ok {
			t.Errorf("parseBluetoothctlDevice(%q): ok=%v want=%v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if dev.Address != c.address || dev.Name != c.name || dev.Transport != "rfcomm" {
			t.Errorf("parseBluetoothctlDevice(%q) = %+v", c.in, dev)
		}
	}
}
