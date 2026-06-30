package services

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// endsOnWholeWord encodes the spec's accept/reject rule: the text before the
// ellipsis must end on a COMPLETE word from the original — i.e. either it is the
// whole original, or the original continues with a space right after it.
// accept: "dinner at the" (orig has "dinner at the hardware...") -> true
// reject: "dinner at the har" (cuts the word "hardware")          -> false
func endsOnWholeWord(orig, head string) bool {
	head = strings.TrimRight(head, " ")
	if head == "" || head == orig {
		return true
	}
	return strings.HasPrefix(orig, head+" ")
}

func TestTruncateDescriptionShortIsUnchanged(t *testing.T) {
	cases := []string{
		"",
		"dinner",
		"dinner at the hardware store",
		strings.Repeat("a", 80), // exactly 80 -> untouched
	}
	for _, in := range cases {
		got := truncateDescription(in)
		if got != in {
			t.Errorf("len %d should pass through unchanged\n in: %q\nout: %q", utf8.RuneCountInString(in), in, got)
		}
		t.Logf("PASS through (%d chars): %q", utf8.RuneCountInString(in), got)
	}
}

// Precise, hand-verifiable construction: six complete 10-char words (65 chars)
// then a space and a 20-char word that straddles the 80-char window. The cut
// must DROP the straddling word entirely and keep "<six words> ..." — never a
// partial slice of the long word.
func TestTruncateDescriptionKeepsLastWordComplete(t *testing.T) {
	word := strings.Repeat("a", 10)
	sixWords := strings.Join([]string{word, word, word, word, word, word}, " ") // 65 chars
	input := sixWords + " " + strings.Repeat("b", 20)                           // 86 chars

	if utf8.RuneCountInString(input) <= 80 {
		t.Fatalf("test input must exceed 80 chars, got %d", utf8.RuneCountInString(input))
	}

	got := truncateDescription(input)
	t.Logf("input  (%d): %q", utf8.RuneCountInString(input), input)
	t.Logf("output (%d): %q", utf8.RuneCountInString(got), got)

	accept := sixWords + " ..."  // complete-word truncation (correct)
	reject := input[:77] + "..." // partial-word truncation (forbidden) -> "...b..."
	if got != accept {
		t.Errorf("want complete-word truncation\nwant: %q\n got: %q", accept, got)
	}
	if got == reject {
		t.Errorf("must NOT cut a word mid-way: %q", got)
	}
	if strings.Contains(got, "b") {
		t.Errorf("the straddling word should be dropped whole, not partially kept: %q", got)
	}
}

// The spec's own example, read as the rule rather than the literal short string:
// a real >80 description must truncate to "...the ..." (accept), never "...har..." (reject).
func TestTruncateDescriptionSpecExample(t *testing.T) {
	input := "dinner at the hardware store downtown with the whole crew before the long road trip back home"
	if utf8.RuneCountInString(input) <= 80 {
		t.Fatalf("test input must exceed 80 chars, got %d", utf8.RuneCountInString(input))
	}

	got := truncateDescription(input)
	t.Logf("input  (%d): %q", utf8.RuneCountInString(input), input)
	t.Logf("output (%d): %q", utf8.RuneCountInString(got), got)

	if utf8.RuneCountInString(got) > 80 {
		t.Errorf("truncated description must be <= 80 chars, got %d: %q", utf8.RuneCountInString(got), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("truncated description must end with ellipsis: %q", got)
	}
	head := strings.TrimSuffix(got, "...")
	if !endsOnWholeWord(input, head) {
		t.Errorf("truncation cut a word in half (reject case): %q is not a whole-word prefix of %q", head, input)
	}
}

func TestTruncateDescriptionSingleHugeWord(t *testing.T) {
	input := strings.Repeat("z", 100) // no spaces -> nothing to break on
	got := truncateDescription(input)
	t.Logf("output (%d): %q", utf8.RuneCountInString(got), got)
	if utf8.RuneCountInString(got) > 80 {
		t.Errorf("must stay <= 80 chars even with no word boundary, got %d", utf8.RuneCountInString(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("must end with ellipsis: %q", got)
	}
}
