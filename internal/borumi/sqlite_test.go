package borumi

import "testing"

func TestSQLQuoteEscapesSingleQuotes(t *testing.T) {
	t.Parallel()

	got := sqlQuote("I'm testing")
	want := "'I''m testing'"
	if got != want {
		t.Fatalf("unexpected sql quote output: got %q want %q", got, want)
	}
}

