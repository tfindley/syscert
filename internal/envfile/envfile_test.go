package envfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBasicPairs(t *testing.T) {
	kvs, err := Parse(strings.NewReader("A=1\nB=two\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []KV{{"A", "1"}, {"B", "two"}}
	assertKVs(t, kvs, want)
}

func TestParseSkipsCommentsAndBlankLines(t *testing.T) {
	in := "# a comment\n\n;semicolon comment\nA=1\n   # indented comment\nB=2\n"
	kvs, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	assertKVs(t, kvs, []KV{{"A", "1"}, {"B", "2"}})
}

func TestParseValueMayContainEquals(t *testing.T) {
	kvs, err := Parse(strings.NewReader("URL=a=b=c\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	assertKVs(t, kvs, []KV{{"URL", "a=b=c"}})
}

func TestParseDoubleQuotesPreserveInnerWhitespace(t *testing.T) {
	kvs, err := Parse(strings.NewReader("A=\"  x y  \"\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	assertKVs(t, kvs, []KV{{"A", "  x y  "}})
}

func TestParseSingleQuotesStripped(t *testing.T) {
	kvs, err := Parse(strings.NewReader("A='x y'\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	assertKVs(t, kvs, []KV{{"A", "x y"}})
}

func TestParseTrimsKeyAndUnquotedValue(t *testing.T) {
	kvs, err := Parse(strings.NewReader("  A =  val  \n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	assertKVs(t, kvs, []KV{{"A", "val"}})
}

func TestParseEmptyValueAllowed(t *testing.T) {
	kvs, err := Parse(strings.NewReader("A=\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	assertKVs(t, kvs, []KV{{"A", ""}})
}

func TestParseErrorOnMissingEquals(t *testing.T) {
	_, err := Parse(strings.NewReader("A=1\nNOEQUALS\n"))
	if err == nil {
		t.Fatal("expected error for line without '='")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should cite line 2, got: %v", err)
	}
}

func TestParseErrorOnEmptyKey(t *testing.T) {
	_, err := Parse(strings.NewReader("=val\n"))
	if err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("expected line-1 error for empty key, got: %v", err)
	}
}

func TestParseErrorOnInvalidKey(t *testing.T) {
	_, err := Parse(strings.NewReader("1BAD=x\n"))
	if err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("expected line-1 error for invalid key, got: %v", err)
	}
}

func TestParseDoesNotLeakValuesInErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("BADKEY WITH SPACES=supersecretvalue\n"))
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "supersecretvalue") {
		t.Errorf("error must not contain the secret value, got: %v", err)
	}
}

func TestMergeRealEnvWins(t *testing.T) {
	preset := map[string]bool{"A": true}
	got := merge(preset, [][]KV{{{"A", "fromfile"}, {"B", "2"}}})
	assertKVs(t, got, []KV{{"B", "2"}})
}

func TestMergeLastFileWins(t *testing.T) {
	got := merge(nil, [][]KV{{{"A", "1"}}, {{"A", "2"}}})
	assertKVs(t, got, []KV{{"A", "2"}})
}

func TestMergeDuplicateWithinFileLastWins(t *testing.T) {
	got := merge(nil, [][]KV{{{"A", "1"}, {"A", "2"}}})
	assertKVs(t, got, []KV{{"A", "2"}})
}

func TestMergePreservesFirstSeenOrder(t *testing.T) {
	got := merge(nil, [][]KV{{{"A", "1"}, {"B", "2"}}, {{"A", "3"}}})
	assertKVs(t, got, []KV{{"A", "3"}, {"B", "2"}})
}

func TestLoadSetsOnlyMissingVars(t *testing.T) {
	t.Setenv("SYSCERT_TEST_PRESET", "orig")
	dir := t.TempDir()
	p := filepath.Join(dir, "secrets")
	if err := os.WriteFile(p, []byte("SYSCERT_TEST_PRESET=new\nSYSCERT_TEST_NEW=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Unsetenv("SYSCERT_TEST_NEW") })

	set, err := Load([]string{p})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	assertStrings(t, set, []string{"SYSCERT_TEST_NEW"})
	if got := os.Getenv("SYSCERT_TEST_PRESET"); got != "orig" {
		t.Errorf("preset var overwritten: got %q, want \"orig\"", got)
	}
	if got := os.Getenv("SYSCERT_TEST_NEW"); got != "x" {
		t.Errorf("new var not set: got %q, want \"x\"", got)
	}
}

func TestLoadErrorsOnMissingFile(t *testing.T) {
	_, err := Load([]string{filepath.Join(t.TempDir(), "nope")})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func assertKVs(t *testing.T, got, want []KV) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d pairs %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pair %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
