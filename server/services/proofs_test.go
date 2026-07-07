package services

import "testing"

// Bonus #2: uploads are validated by magic bytes, not extension or headers.
func TestSniffImage(t *testing.T) {
	pngMagic := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 13, 'I', 'H', 'D', 'R'}
	jpegMagic := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	gifMagic := []byte("GIF89a\x01\x00\x01\x00")
	exeMagic := []byte{'M', 'Z', 0x90, 0x00, 0x03, 0x00, 0x00, 0x00}
	textFile := []byte("definitely-a-receipt.png but actually plain text")
	pdfMagic := []byte("%PDF-1.4 not an image")

	cases := []struct {
		name string
		data []byte
		ok   bool
	}{
		{"png", pngMagic, true},
		{"jpeg", jpegMagic, true},
		{"gif", gifMagic, true},
		{"exe renamed to png", exeMagic, false},
		{"plain text renamed", textFile, false},
		{"pdf", pdfMagic, false},
		{"empty", nil, false},
	}
	for _, c := range cases {
		ct, ok := sniffImage(c.data)
		if ok != c.ok {
			t.Errorf("%s: sniffImage ok=%v (detected %s), want %v", c.name, ok, ct, c.ok)
		}
		t.Logf("%-22s -> detected %-24s accepted=%v", c.name, ct, ok)
	}
}
