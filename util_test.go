package zippath

import (
	"archive/zip"
	"reflect"
	"sort"
	"testing"
)

func TestCompile(t *testing.T) {
	tests := []struct {
		pattern string
		prog    []inst
	}{
		{"a", []inst{{opChar, 'a'}, {op: opMatch}}},
		{"\\b*", []inst{{opChar, '\\'}, {opChar, 'b'}, {op: opSplit}, {op: opMatch}}},
		{"a*2", []inst{{opChar, 'a'}, {op: opSplit}, {opChar, '2'}, {op: opMatch}}},
		{"**", []inst{{op: opSplit}, {op: opSplit}, {op: opMatch}}},
		{"*a", []inst{{op: opSplit}, {opChar, 'a'}, {op: opMatch}}},
	}

	for i, test := range tests {
		prog := compile(test.pattern)
		if !reflect.DeepEqual(prog, test.prog) {
			t.Errorf("case %d: want %v got %v", i, test.prog, prog)
		}
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern string
		s       string
		match   bool
	}{
		{"", "asdfasdf", false},
		{"*", "", true},
		{"*", "o827364zγ", true},
		{"a*", "asdf", true},
		{"a*", "", false},
		{"*a", "a", true},
		{"*a", "", false},
		{"*a", "ab", false},
		{"/path/to*", "/path/to/z", true},
		{"/path/to*", "/path/toloay", true},
	}

	threads := make([]thread, 0, 64)
	for i, test := range tests {
		prog := compile(test.pattern)
		if match(prog, test.s, threads) != test.match {
			t.Errorf("case %d (%s): want %t got %t", i, test.pattern, test.match, !test.match)
		}
	}
}

func TestGlob(t *testing.T) {
	rc, err := zip.OpenReader("testdata/files.zip")
	if err != nil {
		t.Fatalf("couldn't open test data")
	}
	defer rc.Close()
	tests := []struct {
		pattern string
		matches []string
	}{
		{"a*", []string{"a/", "a/b/", "a/b/c/", "a/b/c/f", "a/b/c/f1", "a/b/c/f2", "a/b/d/"}},
		{"a/b/c*", []string{"a/b/c/", "a/b/c/f", "a/b/c/f1", "a/b/c/f2"}},
		{"a/b*f", []string{"a/b/c/f"}},
		{"a/b/c", nil},
	}
	for i, test := range tests {
		matches := Glob(&rc.Reader, test.pattern)
		sort.Strings(test.matches)
		sort.Strings(matches)
		if !reflect.DeepEqual(test.matches, matches) {
			t.Errorf("case %d: want %v got %v", i, test.matches, matches)
		}
	}
}

func TestFilterOutDirs(t *testing.T) {
	tests := []struct {
		orig   []string
		newlen int
	}{
		{[]string{"/a", "b/", "c/d/"}, 1},
		{[]string{"a/", "b/"}, 0},
		{[]string{"."}, 1},
	}
	for i, test := range tests {
		f := FilterOutDirs(test.orig)
		if len(f) != test.newlen {
			t.Errorf("case %d: want len %d got list %v", i, test.newlen, f)
		}
	}
}
