package golua

import (
	"io"
	"os"
)

type FILE struct {
	fp   *os.File
	buff []byte
	cur  int
	n    int
	eof  bool
	err  error
}

var STDIN = &FILE{
	fp:   os.Stdin,
	buff: make([]byte, 1024),
	cur:  0,
	n:    0,
	eof:  false,
}

func fopen(name string, flag int) (*FILE, error) {
	f, err := os.OpenFile(name, flag, os.ModePerm)
	if f == nil || err != nil {
		return nil, err
	}
	return &FILE{
		fp:   f,
		buff: make([]byte, 1024),
		cur:  0,
		n:    0,
		eof:  false,
	}, nil
}

func freopen(name string, flag int, old *FILE) (*FILE, error) {
	old.fp.Close()
	return fopen(name, flag)
}

func (f *FILE) fill() {
	if f.cur < f.n {
		return
	}
	n, err := f.fp.Read(f.buff)
	if n == 0 || err == io.EOF {
		f.eof = true
	} else {
		f.cur = 0
		f.n = n
		f.eof = false
	}

	if err != io.EOF {
		f.err = err
	}
}

func (f *FILE) EOF() bool {
	f.fill()
	return f.eof
}

func (f *FILE) getc() (b byte, ok bool) {

	if f.EOF() {
		return 0, false
	}
	f.cur++
	return f.buff[f.cur-1], true
}

func (f *FILE) ungetc(b byte) {
	f.eof = false
	if f.cur == 0 {
		buff := make([]byte, 1024+len(f.buff))
		copy(buff[1024:], f.buff)
		f.buff = buff
		f.cur += 1024
		f.n += 1024
	}
	f.cur--
	f.buff[f.cur] = b
}

func (f *FILE) fread(data []byte) int {
	var cnt = 0
	size := len(data)

	for !f.EOF() && size > 0 {
		n := f.n - f.cur
		if n > size {
			n = size
		}
		copy(data[cnt:], f.buff[f.cur:f.cur+n])
		f.cur += n
		cnt += n
		size -= n
	}
	return cnt
}

func (f *FILE) fclose() {
	f.err = f.fp.Close()
}

func (f *FILE) ferror() error {
	return f.err
}
