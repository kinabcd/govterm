package govterm

import (
	"bytes"
	"unicode/utf8"
)

type ByteStream struct {
	*Stream
	utf8Decoder func(data []byte) (string, error)
}

func (b *ByteStream) Write(data []byte) (n int, err error) {
	var dataStr string
	if b.UseUTF8 {
		dataStr, err = b.utf8Decoder(data)
		if err != nil {
			return 0, err
		}
	} else {
		dataStr = BytesToString(data)
	}
	b.Stream.WriteString(dataStr)
	return len(data), nil
}

func NewByteStream(stream *Stream) *ByteStream {
	bs := &ByteStream{
		Stream:      stream,
		utf8Decoder: DecodeUTF8WithReplacement,
	}
	return bs
}

func BytesToString(data []byte) string {
	var result string
	for _, b := range data {
		result += string(b)
	}
	return result
}
func DecodeUTF8WithReplacement(data []byte) (string, error) {
	var output bytes.Buffer
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			output.WriteRune('\uFFFD')
			data = data[size:]
		} else {
			output.WriteRune(r)
			data = data[size:]
		}
	}
	return output.String(), nil
}
