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
	equal(val) bool
}

type seq interface {
	pr() string
	equal(val) bool

	empty() bool
	first() val
	rest() seq
}

type empty struct {
}

func (e empty) pr() string {
	return "()"
}

func (e empty) equal(other val) bool {
	_, ok := other.(empty)
	return ok
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

func (c *cons) equal(other val) bool {
	cc, ok := other.(*cons)
	if !ok {
		return false
	}
	return c.car.equal(cc.car) && c.cdr.equal(cc.cdr)
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

func (s symbol) equal(other val) bool {
	ss, ok := other.(symbol)
	if !ok {
		return false
	}
	return s.name == ss.name
}

type number struct {
	i int64
}

func (n number) pr() string {
	return fmt.Sprintf("%v", n.i)
}

func (n number) equal(other val) bool {
	nn, ok := other.(number)
	if !ok {
		return false
	}
	return n.i == nn.i
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

func (b boolean) equal(other val) bool {
	bb, ok := other.(boolean)
	if !ok {
		return false
	}
	return b.b == bb.b
}

func isTrue(v val) bool {
	b, ok := v.(boolean)
	if ok {
		return b.b
	}
	return true
}

type function interface {
	call([]val) val
}

type builtin struct {
	name string
	f    func([]val) val
}

func (b builtin) pr() string {
	return fmt.Sprintf("#<function:%s>", b.name)
}

func (b builtin) equal(other val) bool {
	panic("you should not compare functions!")
}

func (b builtin) call(args []val) val {
	return b.f(args)
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
		ls = ls.advance()
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
		return symbol{name: s}, els, nil
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

func get1(s seq) val {
	v := s.first()

	if !s.rest().empty() {
		panic("Too many items in seq")
	}

	return v
}

func get3(s seq) (val, val, val) {
	v1 := s.first()
	s = s.rest()
	v2 := s.first()
	s = s.rest()
	v3 := s.first()
	s = s.rest()

	if !s.empty() {
		panic(fmt.Sprintf("Too many items in seq: %s", s.pr()))
	}

	return v1, v2, v3
}

type env interface {
	lookup(s symbol) (val, bool)
}

type globalEnv map[string]val

func (ge globalEnv) lookup(s symbol) (val, bool) {
	v, ok := ge[s.name]
	return v, ok
}

func evalApplication(e env, fform val, argForms seq) val {
	vf := eval(e, fform)
	f, ok := vf.(function)
	if !ok {
		panic(fmt.Sprintf("cannot apply non-function %s", vf.pr()))
	}
	args := []val{}
	for !argForms.empty() {
		argForm := argForms.first()
		arg := eval(e, argForm)
		args = append(args, arg)

		argForms = argForms.rest()
	}
	return f.call(args)
}

func eval(e env, v val) val {
	switch v := v.(type) {
	case boolean:
		return v
	case number:
		return v
	case symbol:
		res, ok := e.lookup(v)
		if !ok {
			panic(fmt.Sprintf("unbound %s", v.name))
		}
		return res
	case seq:
		head := v.first()
		switch head := head.(type) {
		case symbol:
			switch head.name {
			case "if":
				cond, cons, alt := get3(v.rest())
				if isTrue(eval(e, cond)) {
					return eval(e, cons)
				} else {
					return eval(e, alt)
				}
			case "quote":
				quotee := get1(v.rest())
				return quotee
			default:
				return evalApplication(e, head, v.rest())
			}
		default:
			return evalApplication(e, head, v.rest())
		}
	default:
		panic(fmt.Sprintf("cannot eval %s", v.pr()))
	}
	//panic("Should not be reached")
}

func builtinPlus(args []val) val {
	sum := int64(0)
	for _, arg := range args {
		n, ok := arg.(number)
		if !ok {
			panic(fmt.Sprintf("cannot add non-number %s", arg.pr()))
		}
		sum += n.i
	}
	return number{sum}
}

func builtinMul(args []val) val {
	prod := int64(1)
	for _, arg := range args {
		n, ok := arg.(number)
		if !ok {
			panic(fmt.Sprintf("cannot multiply non-number %s", arg.pr()))
		}
		prod *= n.i
	}
	return number{prod}
}

func evalTest(input string, expected string) {
	e := map[string]val{
		"one": number{1},
		"+":   builtin{name: "+", f: builtinPlus},
		"*":   builtin{name: "*", f: builtinMul},
	}

	vinput, err := read(input)
	if err != nil {
		panic("could not read")
	}

	vresult := eval(globalEnv(e), vinput)

	if expected != "" {
		vexpected, err := read(expected)
		if err != nil {
			panic("could not read")
		}

		if !vexpected.equal(vresult) {
			panic(fmt.Sprintf("(eval(%s) => %s) != %s", vinput.pr(), vresult.pr(), vexpected.pr()))
		}
	}

	fmt.Printf("eval(%s) => %s\n", vinput.pr(), vresult.pr())
}

func main() {
	readTest("  123  ")
	readTest("1-2")
	readTest("  #t")
	readTest("  #f")
	readTest("  12(  ")
	readTest("  (+ 1 2 () )")
	readTest("(if #f 1 2)")

	evalTest("123", "123")
	evalTest("#t", "#t")
	evalTest("#f", "#f")

	evalTest("(if #f 1 2)", "2")
	evalTest("(if 123 1 2)", "1")
	evalTest("(if 123 (quote true) (quote false))", "true")

	evalTest("one", "1")
	evalTest("+", "")
	evalTest("(+ 1 2 3)", "6")
	evalTest("(* 3 4)", "12")
	evalTest("((if #t + *) 3 4)", "7")
	evalTest("((if #f + *) 3 4)", "12")
}
