package gotfs

import "context"

type FileReader struct {
	ctx context.Context
	s   Store
	op  *Operator
	x   Root
	p   string

	offset uint64
}

func NewReader(ctx context.Context, s Store, op *Operator, x Root, p string) *FileReader {
	return &FileReader{
		ctx: ctx,
		s:   s,
		op:  op,
		x:   x,
		p:   p,
	}
}

func (fr *FileReader) Read(p []byte) (int, error) {
	n, err := fr.op.ReadFileAt(fr.ctx, fr.s, fr.x, fr.p, fr.offset, p)
	fr.offset += uint64(n)
	return n, err
}
