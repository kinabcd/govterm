package govterm

import "fmt"

var (
	Basic = map[string]struct{}{
		BEL: {},
		BS:  {},
		HT:  {},
		LF:  {},
		VT:  {},
		FF:  {},
		CR:  {},
		SO:  {},
		SI:  {},
	}
	Escape = map[string]struct{}{
		RIS:   {},
		IND:   {},
		NEL:   {},
		RI:    {},
		HTS:   {},
		DECSC: {},
		DECRC: {},
	}
	Sharp = map[string]struct{}{
		DECALN: {},
	}
	Csi = map[string]struct{}{
		ICH:     {},
		CUU:     {},
		CUD:     {},
		CUF:     {},
		HPR:     {},
		CUB:     {},
		CNL:     {},
		CPL:     {},
		CHA:     {},
		HPA:     {},
		CUP:     {},
		HVP:     {},
		ED:      {},
		EL:      {},
		IL:      {},
		DL:      {},
		DCH:     {},
		ECH:     {},
		DA:      {},
		DSR:     {},
		VPA:     {},
		TBC:     {},
		SM:      {},
		RM:      {},
		SGR:     {},
		DECSTBM: {},
	}
)

func (s *Stream) HandleBasic(char string) {
	switch char {
	case BEL:
		s.screen.Bell()
	case BS:
		s.screen.Backspace()
	case HT:
		s.screen.Tab()
	case LF, VT, FF:
		s.screen.LineFeed()
	case CR:
		s.screen.CarriageReturn()
	case SO:
		s.screen.ShiftOut()
	case SI:
		s.screen.ShiftIn()
	}
}

func (s *Stream) HandleEscape(char string) {
	switch char {
	case RIS:
		s.screen.Reset()
	case IND:
		s.screen.Index()
	case NEL:
		s.screen.LineFeed()
	case RI:
		s.screen.ReverseIndex()
	case HTS:
		s.screen.SetTabStop()
	case DECSC:
		s.screen.SaveCursor()
	case DECRC:
		s.screen.RestoreCursor()
	}

}

func (s *Stream) HandleSharp(char string) {
	switch char {
	case DECALN:
		s.screen.AlignmentDisplay()
	}
}

func tidyParams(param []int, num int) []int {
	if len(param) < num {
		var newParams = make([]int, num)
		copy(newParams, param)
		return newParams
	}
	return param
}

func (s *Stream) HandleCSI(char string, params []int, kw map[string]any) {
	param := tidyParams(params, 1)
	switch char {
	case ICH:
		s.screen.InsertCharacters(param[0])
	case CUU:
		s.screen.CursorUp(param[0])
	case CUD:
		s.screen.CursorDown(param[0])
	case CUF, HPR:
		s.screen.CursorForward(param[0])
	case CUB:
		s.screen.CursorBack(param[0])
	case CNL:
		s.screen.CursorDown1(param[0])
	case CPL:
		s.screen.CursorUp1(param[0])
	case CHA, HPA:
		s.screen.CursorToColumn(param[0])
	case CUP, HVP:
		param = tidyParams(params, 2)
		s.screen.CursorPosition(param[0], param[1])
	case ED:
		s.screen.EraseInDisplay(param[0])
	case EL:
		s.screen.EraseInLine(param[0])
	case IL:
		s.screen.InsertLines(param[0])
	case DL:
		s.screen.DeleteLines(param[0])
	case DCH:
		s.screen.DeleteCharacters(param[0])
	case ECH:
		s.screen.EraseCharacters(param[0])
	case DA, DSR:
		req := map[string]bool{}
		for k, v := range kw {
			switch v := v.(type) {
			case bool:
				req[k] = v
			}
		}
		s.screen.ReportDeviceAttributes(param[0], req)
	case VPA:
		s.screen.CursorToLine(param[0])
	case VPR:
		s.screen.CursorDown(param[0])
	case TBC:
		s.screen.ClearTabStop(param[0])
	case SM:
		s.screen.SetMode(params, kw)
	case RM:
		s.screen.ResetMode(params, kw)
	case SGR:
		s.screen.SelectGraphicRendition(params...)
	case DECSTBM:
		s.screen.SetMargins(param[0], param[1])
	default:
		fmt.Println("Unsupport type:", char)
	}
}
