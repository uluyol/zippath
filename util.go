package zippath

import (
	"archive/zip"
	"io"
	"os"

	"github.com/pkg/errors"
)

// Open opens the file in the reader if it exists. The filename
// must be an exact match.
func Open(r *zip.Reader, path string) (io.ReadCloser, error) {
	for _, f := range r.File {
		if f.Name == path {
			return f.Open()
		}
	}
	return nil, errors.Wrapf(os.ErrNotExist, "error finding %s", path)
}

// Glob returns the names of all files matching the given pattern in
// the zip archive. The only special character in a pattern is '*',
// it indicates a match of zero or more of any character. It may be
// escaped with a '\' to match the '*' character. Be careful to
// escape the '\' if you are not using a raw string literal.
func Glob(r *zip.Reader, pattern string) []string {
	var matches []string

	threads := make([]thread, 1, 64)
	prog := compile(pattern)

	for _, f := range r.File {
		if match(prog, f.Name, threads) {
			matches = append(matches, f.Name)
		}
	}

	return matches
}

// FilterOutDirs filters out directories in a list of paths. Note that
// the paths slice will be modified by the function.
func FilterOutDirs(paths []string) []string {
	for i := 0; i < len(paths); {
		if paths[i][len(paths[i])-1] == '/' {
			paths = append(paths[:i], paths[i+1:]...)
			continue
		}
		i++
	}
	return paths
}

type opcode uint8

const (
	opChar opcode = iota
	opMatch
	opSplit
)

func (op opcode) String() string {
	switch op {
	case opChar:
		return "char"
	case opMatch:
		return "match"
	case opSplit:
		return "split"
	default:
		return "(unknown)"
	}
}

type inst struct {
	op opcode
	c  rune
}

type thread struct {
	pc   int
	sp   int
	dead bool
}

func compile(pattern string) []inst {
	insts := make([]inst, 0, 64)
	escape := false
	for _, c := range pattern {
		if escape {
			escape = false
			if c == '*' {
				insts = append(insts, inst{op: opChar, c: c})
				continue
			}
			insts = append(insts, inst{op: opChar, c: '\\'})
		}
		switch c {
		case '*':
			insts = append(insts, inst{op: opSplit})
		case '\\':
			escape = true
		default:
			insts = append(insts, inst{op: opChar, c: c})
		}
	}
	insts = append(insts, inst{op: opMatch})
	return insts
}

// match will check if the program terminates successfully
// for the provided string. threads should be a pre-allocated
// slice of threads of at minimum capacity 1 (it is passed in
// to reduce allocations).
func match(prog []inst, s string, threads []thread) bool {
	threads = threads[:1]
	threads[0].pc = 0
	threads[0].sp = 0
	threads[0].dead = false
	numDead := 0
	name := []rune(s)
	threadsNew := make([]thread, 10)

	// This part isn't terribly efficient, we keep dead threads
	// around and keep new threads separate. Both can be solved
	// with smarted indexing in the for loops.
	for len(threads) > numDead {
		threadsNew = threadsNew[:0]
		for i := range threads {
			t := &threads[i]
			if t.dead {
				continue
			}
			inst := prog[t.pc]
			switch inst.op {
			case opChar:
				if t.sp >= len(name) || name[t.sp] != inst.c {
					t.dead = true
					numDead++
					continue
				}
				t.pc++
				t.sp++
			case opMatch:
				if t.sp == len(name) {
					return true
				}
				t.dead = true
				numDead++
			case opSplit:
				if t.sp < len(name) {
					threadsNew = append(threadsNew, thread{pc: t.pc, sp: t.sp + 1})
				}
				t.pc++
			}
		}
		threads = append(threads, threadsNew...)
	}
	return false
}
