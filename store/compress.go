package store

import "github.com/golang/snappy"

type CompressError string

func (e CompressError) Error() string { return string(e) }

func uncompress(data []byte) ([]byte, error) {
	b, err := snappy.Decode(nil, data)
	if err == nil {
		return b, nil
	}
	return nil, CompressError(err.Error())
}

func compress(data []byte) []byte {
	b := make([]byte, snappy.MaxEncodedLen(len(data)))
	return snappy.Encode(b, data)
}
