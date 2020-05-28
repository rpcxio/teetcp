package main

import "io"

func TeeReader(r io.ReadCloser, w io.WriteCloser) *teeReader {
	return &teeReader{r, w}
}

type teeReader struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (t *teeReader) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 && t.w != nil {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}

func (t *teeReader) Close() error {
	w := t.w
	if w != nil {
		w.Close()
	}
	return t.r.Close()
}
