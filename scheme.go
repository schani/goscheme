package main

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"
)

// seq
// integer
// bool
// symbol

type val interface {
	pr() string
}

type seq interface {
	pr() string

	empty() bool
	first() val
	rest() seq
}

type empty struct {
}

func (e empty) pr() string {
	return "()"
}

func (e empty) empty() bool {
	return true
}

func (e empty) first() val {
	panic("first called on empty")
}

func (e empty) rest() seq {
	panic("rest called on empty")
}

type cons struct {
	car val
	cdr seq
}

func (c *cons) pr() string {
	s := fmt.Sprintf("(%s", c.car.pr())
	seq := c.rest()
	for !seq.empty() {
		s = fmt.Sprintf("%s %s", s, seq.first().pr())
		seq = seq.rest()
	}
	return fmt.Sprintf("%s)", s)
}

func (c *cons) empty() bool {
	return false
}

func (c *cons) first() val {
	return c.car
}

func (c *cons) rest() seq {
	return c.cdr
}

type symbol struct {
	name string
}

func (s symbol) pr() string {
	return s.name
}

type number struct {
	i int64
}

func (n number) pr() string {
	return fmt.Sprintf("%v", n.i)
}

type boolean struct {
	b bool
}

func (b boolean) pr() string {
	if b.b {
		return "#t"
	} else {
		return "#f"
	}
}

type lexState struct {
	s   string
	pos int
}

func (ls lexState) isEOS() bool {
	return ls.pos >= len(ls.s)
}

func (ls lexState) advance() lexState {
	if ls.isEOS() {
		panic("Advancing beyond EOS")
	}
	return lexState{s: ls.s, pos: ls.pos + 1}
}

func (ls lexState) current() rune {
	if ls.isEOS() {
		panic("Advancing beyond EOS")
	}
	return rune(ls.s[ls.pos])
}

func (ls lexState) skipWhile(pred func(rune) bool) lexState {
	for !ls.isEOS() && pred(ls.current()) {
		ls = ls.advance()
	}
	return ls
}

func (ls lexState) skipWS() lexState {
	return ls.skipWhile(func(c rune) bool { return unicode.IsSpace(c) })
}

func getToken(start lexState, end lexState) string {
	if start.s != end.s {
		panic("Can't get token from two different strings")
	}
	return start.s[start.pos:end.pos]
}

func (ls lexState) readSeq() (seq, lexState, error) {
	ls = ls.skipWS()
	c := ls.current()
	if c == ')' {
		ls = ls.advance()
		return empty{}, ls, nil
	}
	car, ls, err := ls.read()
	if err != nil {
		return nil, ls, err
	}
	cdr, ls, err := ls.readSeq()
	if err != nil {
		return nil, ls, err
	}
	return &cons{car: car, cdr: cdr}, ls, nil
}

func (ls lexState) read() (val, lexState, error) {
	ls = ls.skipWS()
	if ls.isEOS() {
		return nil, ls, errors.New("EOS")
	}
	c := ls.current()
	if c == '#' {
		ls = ls.advance()
		if ls.isEOS() {
			return nil, ls, errors.New("EOS")
		}
		c = ls.current()
		if c == 't' {
			return boolean{true}, ls, nil
		}
		if c == 'f' {
			return boolean{false}, ls, nil
		}
		return nil, ls, errors.New("No boolean")
	}
	if c == '(' {
		ls = ls.advance()
		return ls.readSeq()
	}
	if c == ')' {
		return nil, ls, errors.New("unexpected `)`")
	}
	els := ls.skipWhile(func(c rune) bool {
		return !unicode.IsSpace(c) && c != '(' && c != ')'
	})
	// FIXME: Actually check whether the string contains any
	// nondigits.
	s := getToken(ls, els)
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return &symbol{name: s}, els, nil
	}
	return number{num}, els, nil
}

func read(s string) (val, error) {
	ls := lexState{s: s, pos: 0}
	v, _, err := ls.read()
	return v, err
}

func readTest(s string) val {
	v, err := read(s)
	if err != nil {
		panic("could not read")
	}
	fmt.Printf("`%s` => %s\n", s, v.pr())
	return v
}

func main() {
	readTest("  123  ")
	readTest("1-2")
	readTest("  #t")
	readTest("  #f")
	readTest("  12(  ")
	readTest("  (+ 1 2 () )")
}
