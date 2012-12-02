package csexp

import (
  "bytes"
  "fmt"
  "regexp"
  "strconv"
)

type item struct {
  typ itemType
  pos int
  val []byte
}

type itemType int

const (
  itemError itemType = iota
  itemBracketLeft  // (
  itemBracketRight // )
  itemBytes        // []byte
  itemEOF
)

var (
  reBracketLeft  = regexp.MustCompile(`^\(`)
  reBracketRight = regexp.MustCompile(`^\)`)
  reBytesLength  = regexp.MustCompile(`^(\d+):`)
)

const eof = -1

type stateFn func(*lexer) stateFn

type lexer struct {
  input   []byte
  items   chan item
  start   int
  pos     int
  state   stateFn
  matches [][]byte
}

func (l *lexer) emit(t itemType) {
  l.items <- item{t, l.start, l.input[l.start:l.pos]}
}

func (l *lexer) next() item {
  item := <-l.items
  return item
}

// Match and advance pointer.
func (l *lexer) scan(re *regexp.Regexp) bool {
  if l.match(re) {
    l.start = l.pos
    l.pos  += len(l.matches[0])
    return true
  }
  return false
}

// Match without advancing pointer.
func (l *lexer) match(re *regexp.Regexp) bool {
  if l.matches = re.FindSubmatch(l.input[l.pos:]); l.matches != nil {
    return true
  }
  return false
}

func (l *lexer) run() {
  for l.state = lexCsexp; l.state != nil; {
    l.state = l.state(l)
  }
  close(l.items)
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
  l.items <- item{itemError, l.start, []byte(fmt.Sprintf(format, args...))}
  return nil
}

func lex(input []byte) *lexer {
  l := &lexer{
    input: input,
    items: make(chan item),
  }
  go l.run()
  return l
}

func lexCsexp(l *lexer) stateFn {
  if l.pos >= len(l.input) {
    l.emit(itemEOF)
    return nil
  }
  if l.scan(reBracketLeft) {
    l.emit(itemBracketLeft)
    return lexCsexp
  }
  if l.scan(reBracketRight) {
    l.emit(itemBracketRight)
    return lexCsexp
  }
  if l.scan(reBytesLength) {
    bytes, _ := strconv.ParseInt(string(l.matches[1]), 10, 64)
    l.start = l.pos
    l.pos += int(bytes)
    l.emit(itemBytes)

    return lexCsexp
  }
  return l.errorf("Expected expression.")
}

type Atomizer interface {
  Bytes() ([]byte)
}

type Atom struct {
  Value []byte
}

type Expression struct {
  Values []Atomizer
}

func (a *Atom) Bytes() ([]byte) {
  return []byte(fmt.Sprintf("%d:%s", len(a.Value), a.Value))
}

func (s *Expression) Bytes() ([]byte) {
  st := new(bytes.Buffer)
  st.WriteString("(")
  for _, exp := range s.Values {
    st.Write(exp.Bytes())
  }
  st.WriteString(")")
  return st.Bytes()
}

func Unmarshal(data []byte) *Expression {
  l := lex(data)
  s := []*Expression{&Expression{}}

  for item := l.next(); item.typ != itemEOF; item = l.next() {
    switch item.typ {
    case itemBracketLeft:
      s = append(s, &Expression{})
      s[len(s)-2].Values = append(s[len(s)-2].Values, s[len(s)-1])
    case itemBracketRight:
      s = s[:len(s)-1]
    case itemBytes:
      s[len(s)-1].Values = append(s[len(s)-1].Values, &Atom{Value: item.val})
    default:
      panic("unreachable")
    }
  }
  return s[0].Values[0].(*Expression)
}

