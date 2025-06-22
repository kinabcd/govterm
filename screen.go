package govterm

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode"

	runewidth "github.com/mattn/go-runewidth"
	"golang.org/x/text/unicode/norm"
)

type Screen struct {
	savePoints []Savepoint
	columns    int

	lines int
	//buffer  map[int]*StaticDefaultDict[int, Char]
	buffer  *ScreenBuffer
	mode    map[int]struct{}
	margins *Margins

	title     string
	iconName  string
	charset   int
	g0Charset []rune
	g1Charset []rune
	tabsTops  map[int]bool
	cursor    Cursor
	savedCols int
}

var (
	DefaultMode = map[int]struct{}{
		DECCOLM: {},
		DECTCEM: {},
	}
)

func init() {
	runewidth.DefaultCondition.EastAsianWidth = false
	runewidth.DefaultCondition.StrictEmojiNeutral = false
}

func NewScreen(columns int, lines int) *Screen {
	reverse := false
	if _, ok := DefaultMode[DECSCNM]; ok {
		reverse = ok
	}
	defaultVal := Char{Data: " ", Fg: "default", Bg: "default", Reverse: reverse}
	s := &Screen{
		columns: columns,
		lines:   lines,
		buffer:  NewScreenBuffer(defaultVal),
		mode:    DefaultMode,
		margins: &Margins{Top: 0, Bottom: 0},
	}
	s.Reset()
	return s
}

func (s *Screen) String() string {
	return fmt.Sprintf("%T(%d, %d)", s, s.columns, s.lines)
}

func (s *Screen) DefaultChar() Char {
	_, reverse := s.mode[DECSCNM]
	return Char{
		Data:    " ",
		Fg:      "default",
		Bg:      "default",
		Reverse: reverse,
	}
}

func (s *Screen) EnsureHBounds() {
	s.cursor.X = min(max(0, s.cursor.X), s.columns-1)
}

// EnsureVBounds ensures the cursor is within vertical screen bounds.
func (s *Screen) EnsureVBounds(useMargins bool) {
	top, bottom := 0, s.lines-1
	if s.margins != nil {
		if _, ok := s.mode[DECOM]; ok || useMargins {
			top = s.margins.Top
			bottom = s.margins.Bottom
		}
	}

	s.cursor.Y = min(max(top, s.cursor.Y), bottom)
}

func (s *Screen) Index() {
	top, bottom := 0, s.lines-1
	if s.margins != nil {
		top, bottom = s.margins.Top, s.margins.Bottom
	}

	if s.cursor.Y == bottom {
		for y := top; y < bottom; y++ {
			s.buffer.Set(y, s.buffer.GetValue(y+1))
		}
		s.buffer.Delete(bottom)
	} else {
		s.CursorDown(0)
	}
}

func (s *Screen) Draw(data string) {
	var translated string
	if s.charset > 0 {
		translated = Translate(data, s.g1Charset)
	} else {
		translated = Translate(data, s.g0Charset)
	}

	for _, char := range translated {
		charWidth := runewidth.RuneWidth(char)

		if s.cursor.X == s.columns {
			if _, ok := s.mode[DECAWM]; ok {
				s.CarriageReturn()
				s.LineFeed()
			} else if charWidth > 0 {
				s.cursor.X -= charWidth
			}
		}

		if _, ok := s.mode[IRM]; ok && charWidth > 0 {
			s.InsertCharacters(charWidth)
		}

		line := s.buffer.Get(s.cursor.Y)
		switch {
		case charWidth == 1:
			t := s.cursor.Attrs
			t.Data = string(char)
			line.Set(s.cursor.X, t)
		case charWidth == 2:
			t := s.cursor.Attrs
			t.Data = string(char)
			line.Set(s.cursor.X, t)
			if s.cursor.X+1 < s.columns {
				t1 := s.cursor.Attrs
				t1.Data = ""
				line.Set(s.cursor.X+1, t1)
			}
		case charWidth == 0 && unicode.Is(unicode.Mn, char):
			if s.cursor.X > 0 {
				last := line.Get(s.cursor.X - 1)
				last.Data = norm.NFC.String(last.Data + string(char))
				line.Set(s.cursor.X-1, last)
			} else if s.cursor.Y > 0 {
				l := s.buffer.Get(s.cursor.Y - 1)
				last := l.Get(s.columns - 1)
				l1 := s.buffer.Get(s.cursor.Y - 1)
				last.Data = norm.NFC.String(last.Data + string(char))
				l1.Set(s.columns-1, last)
			}
		default:
			break
		}

		if charWidth > 0 {
			s.cursor.X = min(s.cursor.X+charWidth, s.columns)
		}
	}
}

func (s *Screen) defineCharset(code, mode string) {
	if _, ok := CHARMAPS[code]; ok {
		switch mode {
		case "(":
			s.g0Charset = CHARMAPS[code]
		case ")":
			s.g1Charset = CHARMAPS[code]
		}
	}
}

func (s *Screen) setIconName(param string) {
	s.iconName = param
}

func (s *Screen) setTitle(param string) {
	s.title = param
}

func (s *Screen) Display() []string {
	var renderLine = func(line *StaticDefaultDict[int, Char]) string {
		var lineStr string
		isWideChar := false

		for x := 0; x < s.columns; x++ {
			if isWideChar {
				isWideChar = false
				continue
			}
			char := line.Get(x)
			if len(char.Data) > 0 {
				isWideChar = runewidth.RuneWidth(rune(char.Data[0])) == 2
			}
			lineStr += char.Data
		}
		return lineStr
	}
	var result []string
	for y := 0; y < s.lines; y++ {
		result = append(result, renderLine(s.buffer.Get(y)))
	}
	return result
}

// basic

func (s *Screen) Bell(args ...any) {}
func (s *Screen) Backspace() {
	s.CursorBack(0)
}

func (s *Screen) Tab() {
	var sortTabs []int
	for k := range s.tabsTops {
		sortTabs = append(sortTabs, k)
	}
	sort.Ints(sortTabs)
	column := s.columns - 1
	for stop := range sortTabs {
		if s.cursor.X < stop {
			column = stop
			break
		}
	}
	s.cursor.X = column
}

func (s *Screen) LineFeed() {
	s.Index()
	if _, ok := s.mode[LNM]; ok {
		s.CarriageReturn()
	}
}

func (s *Screen) CarriageReturn() {
	s.cursor.X = 0
}

func (s *Screen) ShiftOut() {
	s.charset = 1
}
func (s *Screen) ShiftIn() {
	s.charset = 0
}

// escape

func (s *Screen) Reset() {
	s.buffer.Clear()

	s.margins = nil
	s.mode = map[int]struct{}{
		DECAWM:  {},
		DECTCEM: {},
	}

	s.title = ""
	s.iconName = ""

	s.charset = 0
	s.g0Charset = LAT1Chars
	s.g1Charset = VT100Chars

	s.tabsTops = make(map[int]bool)
	for i := 8; i < s.columns; i += 8 {
		s.tabsTops[i] = true
	}

	s.cursor = Cursor{X: 0, Y: 0}
	s.CursorPosition(0, 0)
	s.savedCols = -1
}

func (s *Screen) ReverseIndex() {
	top, bottom := 0, s.lines-1
	if s.margins != nil {
		top, bottom = s.margins.Top, s.margins.Bottom
	}
	if s.cursor.Y == top {
		for y := bottom; y > top; y-- {
			s.buffer.Set(y, s.buffer.GetValue(y-1))
		}
		s.buffer.Delete(top)
	} else {
		s.CursorUp(0)
	}

}
func (s *Screen) SetTabStop() {
	s.tabsTops[s.cursor.X] = true
}
func (s *Screen) SaveCursor() {
	sp := Savepoint{
		Cursor:    s.cursor,
		G0Charset: s.g0Charset,
		G1Charset: s.g1Charset,
		Charset:   s.charset,
	}
	if _, ok := s.mode[DECOM]; ok {
		sp.Origin = ok
	}
	if _, ok := s.mode[DECAWM]; ok {
		sp.Wrap = ok
	}
	s.savePoints = append(s.savePoints, sp)
}
func (s *Screen) RestoreCursor() {
	if len(s.savePoints) > 0 {
		sp, _ := Pop(&s.savePoints)
		s.g0Charset = sp.G0Charset
		s.g1Charset = sp.G1Charset
		s.charset = sp.Charset

		if sp.Origin {
			s.SetMode([]int{DECOM}, map[string]any{"private": false})
		}
		if sp.Wrap {
			s.SetMode([]int{DECAWM}, map[string]any{"private": false})
		}
		s.cursor = sp.Cursor
		s.EnsureHBounds()
		s.EnsureVBounds(true)
	} else {
		s.ResetMode([]int{DECOM}, map[string]any{"private": false})
		s.CursorPosition(0, 0)
	}
}

// sharp

func (s *Screen) AlignmentDisplay() {
	for y := 0; y < s.lines; y++ {
		for x := 0; x < s.columns; x++ {
			t := s.buffer.Get(y).Get(x)
			t.Data = "E"
			s.buffer.Get(y).Set(x, t)
		}
	}
}

// csi

func (s *Screen) InsertCharacters(count int) {
	count = max(count, 1)
	line := s.buffer.Get(s.cursor.Y)
	for x := s.columns; x >= s.cursor.X; x-- {
		currentValue := line.Get(x)
		if x+count <= s.columns {
			line.Set(x+count, currentValue)
		}
		line.Del(x)
	}
}

func (s *Screen) CursorUp(count int) {
	count = max(1, count)
	// Move cursor up the indicated # of lines in the same column
	// Cursor stops at top margin.
	if s.margins != nil {
		s.cursor.Y = max(s.cursor.Y-count, s.margins.Top)
	} else {
		s.cursor.Y = max(s.cursor.Y-count, 0)
	}
}

func (s *Screen) CursorDown(count int) {
	count = max(1, count)
	bottom := s.lines - 1
	if s.margins != nil {
		bottom = s.margins.Bottom
	}

	newPosition := s.cursor.Y + count
	if newPosition > bottom {
		newPosition = bottom
	}
	s.cursor.Y = newPosition
}

func (s *Screen) CursorForward(count int) {
	count = max(1, count)
	s.cursor.X += count
	s.EnsureHBounds()
}
func (s *Screen) CursorBack(count int) {
	count = max(1, count)
	if s.cursor.X == s.columns {
		s.cursor.X -= 1
	}
	s.cursor.X -= count
	s.EnsureHBounds()
}
func (s *Screen) CursorDown1(count int) {
	s.CursorDown(count)
	s.CarriageReturn()
}
func (s *Screen) CursorUp1(count int) {
	s.CursorUp(count)
	s.CarriageReturn()
}

func (s *Screen) EraseInDisplay(how int) {
	var startY, endY int
	switch how {
	case 0:
		startY = s.cursor.Y + 1
		endY = s.lines
	case 1:
		startY = 0
		endY = s.cursor.Y
	case 2, 3:
		startY = 0
		endY = s.lines
	}
	for y := startY; y < endY; y++ {
		s.buffer.Delete(y)
	}
	if how == 0 || how == 1 {
		s.EraseInLine(how)
	}

}
func (s *Screen) EraseInLine(how int) {
	var startX, endX int
	switch how {
	case 0: // 清除從當前位置到本行結尾的字符
		startX = s.cursor.X
		endX = s.columns
	case 1: // 清除從本行開始到當前位置的字符
		startX = 0
		endX = s.cursor.X + 1
	case 2: // 清除本行所有字符 (光標位置不變)
		startX = 0
		endX = s.columns
	}
	line := s.buffer.Get(s.cursor.Y)
	for x := startX; x < endX; x++ {
		line.Set(x, s.cursor.Attrs)
	}
}
func (s *Screen) InsertLines(count int) {
	count = max(1, count)
	top, bottom := 0, s.lines-1
	if s.margins != nil {
		top, bottom = s.margins.Top, s.margins.Bottom
	}

	if top <= s.cursor.Y && s.cursor.Y <= bottom {
		for y := s.cursor.Y - 1; y > bottom; y-- {
			if y+count <= bottom {
				if s.buffer.HasKey(y) {
					s.buffer.Set(y+count, s.buffer.GetValue(y))
				}
			}
			s.buffer.Delete(y)
		}
		s.CarriageReturn()
	}
}
func (s *Screen) DeleteLines(count int) {
	if count == 0 {
		count = 1
	}
	top, bottom := 0, s.lines-1
	if s.margins != nil {
		top, bottom = s.margins.Top, s.margins.Bottom
	}
	if top <= s.cursor.Y && s.cursor.Y <= bottom {
		for y := s.cursor.Y; y < bottom+1; y++ {
			if y+count <= bottom {
				if s.buffer.HasKey(y + count) {
					s.buffer.Set(y, s.buffer.GetValue(y+count))
					s.buffer.Delete(y + count)
				}
			} else {
				s.buffer.Delete(y)
			}
		}
		s.CarriageReturn()
	}
}
func (s *Screen) DeleteCharacters(count int) {
	count = max(1, count)

	line := s.buffer.Get(s.cursor.Y)
	for x := s.cursor.X; x < s.columns; x++ {
		if x+count <= s.columns {
			line.Set(x, line.Get(x+count))
		} else {
			line.Del(x)
		}
	}
}
func (s *Screen) EraseCharacters(count int) {
	count = max(1, count)

	line := s.buffer.Get(s.cursor.Y)
	endX := min(s.cursor.X+count, s.columns)
	for x := s.cursor.X; x < endX; x++ {
		line.Set(x, s.cursor.Attrs)
	}
}
func (s *Screen) ReportDeviceAttributes(mode int, kw map[string]bool) {
	if mode == 0 {
		if v, ok := kw["private"]; ok && v {
			s.WriteProcessInput(CSI + "?6c")
		}
	}
}

func (s *Screen) CursorToLine(line int) {
	y := max(0, line-1)
	s.cursor.Y = y

	if _, ok := s.mode[DECOM]; ok {
		if s.margins == nil {
			fmt.Println("error: margin is none")
			return
		}
		s.cursor.Y += s.margins.Top
	}
	s.EnsureVBounds(false)
}
func (s *Screen) CursorPosition(line, column int) {
	y := max(0, line-1)
	x := max(0, column-1)

	if s.margins != nil {
		if _, ok := s.mode[DECOM]; ok {
			y += s.margins.Top
		}
		if !(s.margins.Top <= y && y <= s.margins.Bottom) {
			return
		}
	}

	s.cursor.X = x
	s.cursor.Y = y
	s.EnsureHBounds()
	s.EnsureVBounds(false)
}
func (s *Screen) ClearTabStop(how int) {
	switch how {
	case 0:
		delete(s.tabsTops, s.cursor.X)
	case 3:
		s.tabsTops = make(map[int]bool)
	}
}
func (s *Screen) SetMode(modes []int, kw map[string]any) {
	//var modeList []int
	var modeList = make([]int, len(modes))
	copy(modeList, modes)
	if val, ok := kw["private"]; ok && val.(bool) {
		for i, v := range modes {
			modeList[i] = v << 5
		}
	}
	for _, v := range modeList {
		s.mode[v] = struct{}{}
	}

	if slices.Contains(modeList, DECCOLM) {
		s.savedCols = s.columns
		s.Resize(0, 132)
		s.EraseInDisplay(2)
		s.CursorPosition(0, 0)
	}

	if slices.Contains(modeList, DECOM) {
		s.CursorPosition(0, 0)
	}

	if slices.Contains(modeList, DECSCNM) {
		for _, v := range s.buffer.Map {
			v.DefaultVal = s.DefaultChar()
			for x, v1 := range v.Data {
				v1.Reverse = true
				v.Set(x, v1)
			}
		}
		s.SelectGraphicRendition(7)
	}

	if slices.Contains(modeList, DECTCEM) {
		s.cursor.Hidden = false
	}
}

func (s *Screen) ResetMode(modes []int, kw map[string]any) {
	var modeList = make([]int, len(modes))
	copy(modeList, modes)
	if private, ok := kw["private"]; ok && private.(bool) {
		for i, v := range modes {
			modeList[i] = v << 5
		}
	}
	for _, v := range modeList {
		delete(s.mode, v)
	}

	if slices.Contains(modeList, DECCOLM) {
		if s.columns == 132 && s.savedCols != -1 {
			s.Resize(0, s.savedCols)
			s.savedCols = -1
		}
		s.EraseInDisplay(2)
		s.CursorPosition(0, 0)
	}

	if slices.Contains(modeList, DECOM) {
		s.CursorPosition(0, 0)
	}

	if slices.Contains(modeList, DECSCNM) {
		for _, v := range s.buffer.Map {
			v.DefaultVal = s.DefaultChar()
			for x, v1 := range v.Data {
				v1.Reverse = false
				v.Set(x, v1)
			}
		}
		s.SelectGraphicRendition(27)
	}

	if slices.Contains(modeList, DECTCEM) {
		s.cursor.Hidden = true
	}
}
func (s *Screen) SelectGraphicRendition(attrs ...int) {
	if attrs == nil || (len(attrs) == 1 && attrs[0] == 0) {
		s.cursor.Attrs = s.DefaultChar()
		return
	}
	replace := map[string]any{}

	attrList := slices.Clone(attrs)
	slices.Reverse(attrList)
	var attr int
	for len(attrList) > 0 {
		attr, _ = Pop(&attrList)
		if attr == 0 {
			defaultV := s.DefaultChar()
			for k, v := range defaultV.Map() {
				replace[k] = v
			}
		} else if v, ok := FG_ANSI[attr]; ok {
			replace["fg"] = v
		} else if v, ok := BG[attr]; ok {
			replace["bg"] = v
		} else if v, ok := TEXT[attr]; ok {
			replace[v[1:]] = strings.HasPrefix(v, "+")
		} else if v, ok := FG_AIXTERM[attr]; ok {
			replace["fg"] = v
		} else if v, ok := BG_AIXTERM[attr]; ok {
			replace["bg"] = v
		} else if attr == FG_256 || attr == BG_256 {
			key := "bg"
			if attr == FG_256 {
				key = "fg"
			}
			if n, ok := Pop(&attrList); ok {
				if n == 5 {
					if m, ok := Pop(&attrList); ok {
						replace[key] = FG_BG_256[m]
					}
				} else if n == 2 {
					v1, ok1 := Pop(&attrList)
					v2, ok2 := Pop(&attrList)
					v3, ok3 := Pop(&attrList)
					if ok1 && ok2 && ok3 {
						replace[key] = fmt.Sprintf("%02x%02x%02x", v1, v2, v3)
					}
				}
			}
		}
	}
}
func (s *Screen) ReportDeviceStatus(mode int, kw map[string]bool) {
	if mode == 0 {
		if v, ok := kw["private"]; !(ok || v) {
			s.WriteProcessInput(CSI + "?6c")
		}
	}
}
func (s *Screen) SetMargins(top, bottom int) {
	if (top == -1 || top == 0) && bottom == -1 {
		s.margins = nil
	}
	margins := Margins{Top: 0, Bottom: s.lines - 1}

	if s.margins == nil {
		top = margins.Top
	} else {
		top = max(0, min(top-1, s.lines-1))
	}
	if bottom == -1 {
		bottom = margins.Bottom
	} else {
		bottom = max(0, min(bottom-1, s.lines-1))
	}

	if bottom-top >= 1 {
		s.margins = &Margins{top, bottom}
		s.CursorPosition(0, 0)
	}
}
func (s *Screen) CursorToColumn(column int) {
	x := max(0, column-1)
	s.cursor.X = x
	s.EnsureHBounds()
}

func (s *Screen) Resize(lines, columns int) {
	if lines == s.lines && columns == s.columns {
		return
	}
	// 其他调整大小的逻辑
}

// WriteProcessInput default is a noop
// data text to write to the process stdin
func (s *Screen) WriteProcessInput(data string) {

}
