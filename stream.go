package govterm

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	Bell             = "bell"
	Backspace        = "backspace"
	Tab              = "tab"
	Linefeed         = "linefeed"
	CarriageReturn   = "carriage_return"
	ShiftOut         = "shift_out"
	ShiftIn          = "shift_in"
	Reset            = "reset"
	Index            = "index"
	ReverseIndex     = "reverse_index"
	SetTabStop       = "set_tab_stop"
	SaveCursor       = "save_cursor"
	RestoreCursor    = "restore_cursor"
	AlignmentDisplay = "alignment_display"
)

type Stream struct {
	screen          *Screen
	UseUTF8         bool
	TakingPlainText bool
	Basic           map[string]struct{}
	Escape          map[string]struct{}
	Sharp           map[string]struct{}
	Csi             map[string]struct{}
	TextPattern     *regexp.Regexp

	buf *bytes.Buffer
}

func generateTextPattern() (*regexp.Regexp, error) {
	special := map[string]struct{}{
		RegBEL:   {},
		RegBS:    {},
		RegHT:    {},
		RegLF:    {},
		RegVT:    {},
		RegFF:    {},
		RegCR:    {},
		RegSO:    {},
		RegSI:    {},
		RegESC:   {},
		RegCSIC1: {},
		RegNUL:   {},
		RegDEL:   {},
		RegOSCC1: {},
	}
	//var specialChars []string
	//for k := range special {
	//	specialChars = append(specialChars, k)
	//	//specialChars = append(specialChars, regexp.QuoteMeta(k))
	//}

	var escaped string
	for s := range special {
		//escaped += regexp.QuoteMeta(s)
		escaped += s
	}

	// 创建正则表达式
	pattern := "[^" + escaped + "]+"

	//pattern := "[^" + strings.Join(specialChars, "") + "]+"
	return regexp.Compile(pattern)
}

func NewStream(screen *Screen) *Stream {
	textPattern, err := generateTextPattern()
	if err != nil {
		fmt.Println(err)
	}
	s := &Stream{
		screen:          screen,
		UseUTF8:         true,
		TakingPlainText: false,
		TextPattern:     textPattern,
		Basic:           Basic,
		Escape:          Escape,
		Sharp:           Sharp,
		Csi:             Csi,

		buf: &bytes.Buffer{},
	}

	return s
}

func (s *Stream) WriteString(data string) (n int, err error) {
	n, err = s.buf.WriteString(data)
	if s.screen == nil {
		panic("Listener is nil")
	}
	for {
		if s.buf.Len() == 0 {
			return
		} else if sb := s.buf.String(); s.isFsm(sb) {
			l, errFsm := s.processFsm(sb)
			if errFsm != nil {
				return
			}
			s.buf.Next(l)
		} else {
			if s.UseUTF8 {
				r, _, _ := s.buf.ReadRune()
				if r == utf8.RuneError {
					if s.buf.Len() >= 4 {
						s.buf.UnreadRune()
					}
					return
				}
				s.screen.Draw(string(r))
			} else {
				b, _ := s.buf.ReadByte()
				s.screen.Draw(string([]byte{b}))
			}
		}
	}
}

// return true if Fsm
func (s *Stream) isFsm(sBuf string) bool {
	for _, sp := range spByte {
		if strings.HasPrefix(sBuf, sp) {
			return true
		}
	}
	return false
}

func (s *Stream) processFsm(buf string) (l int, err error) {
	if char := buf[0:1]; char == ESC {
		if len(buf) < 2 {
			return 0, io.EOF
		} else if char2 := buf[1:2]; char2 == "[" {
			l, err = s.processCSI(buf[2:])
			return l + 2, err
		} else if char2 == "]" {
			l, err = s.processOSC(buf[2:])
			return l + 2, err
		} else {
			if len(buf) < 3 {
				return 0, io.EOF
			}
			char3 := buf[2:3]
			if char2 == "#" {
				s.HandleSharp(char3)
			} else if char2 == "%" {
				s.selectOtherCharset(char3)
			} else if char2 == "(" || char2 == ")" {
				if !s.UseUTF8 {
					s.screen.defineCharset(char3, char2)
				}
			} else {
				return 1, nil
			}
			return 3, nil
		}
	} else if _, ok := s.Basic[char]; ok {
		if (char == SI || char == SO) && s.UseUTF8 {
		} else {
			s.HandleBasic(char)
		}
		return 1, nil
	} else if char == CSIC1 {
		l, err = s.processCSI(buf[1:])
		return l + 1, err
	} else if char == OSCC1 {
		l, err = s.processOSC(buf[1:])
		return l + 1, err
	} else if char == NUL || char == DEL {
		s.screen.Draw(char)
		return 1, nil
	} else {
		return 1, nil
	}
}

func (s *Stream) processCSI(buf string) (l int, err error) {
	var params []int
	current := ""
	private := false
	AllowedInCsi := BEL + BS + HT + LF + VT + FF + CR
	SpOrGt := SP + ">"
	CanOrSub := CAN + SUB
	var basicQueue = []string{}
	doBasic := func() {
		for _, char := range basicQueue {
			s.HandleBasic(char)
		}
	}

	for {
		if len(buf) == 0 {
			return 0, io.EOF
		}
		char := buf[0:1]
		buf = buf[1:]
		l += 1
		if char == "?" {
			private = true
		} else if strings.Contains(AllowedInCsi, char) {
			basicQueue = append(basicQueue, char)
		} else if strings.Contains(SpOrGt, char) {
		} else if strings.Contains(CanOrSub, char) {
			doBasic()
			s.screen.Draw(char)
			return l, nil
		} else if unicode.IsDigit(rune(char[0])) {
			current += char
		} else if char == "$" {
			doBasic()
			return l, nil
		} else {
			num, _ := strconv.Atoi(current)
			params = append(params, min(num, 9999))
			if char == ";" {
				current = ""
			} else {
				doBasic()
				if private {
					s.HandleCSI(char, params, map[string]any{"private": true})
				} else {
					s.HandleCSI(char, params, nil)
				}
				return l, nil
			}
		}
	}

}
func (s *Stream) processOSC(buf string) (l int, err error) {
	OscTermINATORS := map[string]struct{}{
		STC0: {},
		STC1: {},
		BEL:  {},
	}
	if len(buf) == 0 {
		return 0, io.EOF
	}
	code := buf[0:1]
	switch code {
	case "R", "P":
		return 1, nil
	}
	buf = buf[1:]
	l += 1

	param := ""
	for {
		if len(buf) == 0 {
			return 0, io.EOF
		}
		char := buf[0:1]
		buf = buf[1:]
		l += 1

		if char == ESC {
			char += buf[0:1]
			buf = buf[1:]
			l += 1
		}
		if _, ok := OscTermINATORS[char]; ok {
			break
		} else {
			param += char
		}
	}
	param = param[:1]
	if strings.Contains("01", code) {
		s.screen.setIconName(param)
	}
	if strings.Contains("02", code) {
		s.screen.setTitle(param)
	}
	return
}

func (s *Stream) selectOtherCharset(code string) {
	if code == "@" {
		s.UseUTF8 = false
	} else if strings.Contains("G8", code) {
		s.UseUTF8 = true
	}
}
