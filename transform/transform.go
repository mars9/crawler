package transform

import "golang.org/x/text/transform"

type normalize struct {
	prev byte
}

func (n *normalize) Transform(dst, src []byte, atEOF bool) (int, int, error) {
	var nDst, nSrc int
	for nDst < len(dst) && nSrc < len(src) {
		c := src[nSrc]
		switch c {
		case ' ', '\t', '\v', '\f', 0x85, 0xA0:
			switch n.prev {
			case ' ', '\t', '\v', '\f', 0x85, 0xA0:
				nSrc++
				n.prev = c
				continue
			}
			dst[nDst] = ' '
		case '\r':
			dst[nDst] = '\n'
		case '\n':
			if n.prev == '\r' {
				nSrc++
				n.prev = c
				continue
			}
			dst[nDst] = '\n'
		default:
			dst[nDst] = c
		}
		n.prev = c
		nDst++
		nSrc++
	}
	if nSrc < len(src) {
		return nDst, nSrc, transform.ErrShortDst
	}
	return nDst, nSrc, nil
}

func (n *normalize) Reset() { n.prev = 0 }

type RemoveFunc func(r rune) bool

func removeFunc(r rune) bool { return false }

// Transform removes from the input all carriage returns and replaces
// multiple whitespaces with one.  Illegal bytes in the input are
// replaced by utf8.RuneError. Not doing so might otherwise turn a
// sequence of invalid UTF-8 into valid UTF-8.  The resulting byte
// sequence may subsequently contain runes for which t(r) is true that
// were passed unnoticed.
//
// If fn is not nil Transform removes from the input data all runes r
// for which fn(r) is true.
func Transform(data []byte, fn RemoveFunc) (result []byte, n int, err error) {
	norm := &normalize{}
	if fn == nil {
		fn = removeFunc
	}

	t := transform.Chain(transform.RemoveFunc(fn), norm)
	result, n, err = transform.Bytes(t, data)
	return result, n, err
}
