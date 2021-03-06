package ring

type BufferUint64 struct {
	Buf    []uint64
	Cursor int
	Full   bool
}

func NewBufferUint64(size int) *BufferUint64 {
	return &BufferUint64{
		Buf:    make([]uint64, size),
		Cursor: -1,
	}
}

// Get returns the value at the given index or nil if nothing
func (b *BufferUint64) Get(index int) (out *uint64) {
	bl := len(b.Buf)
	if index < bl {
		cursor := b.Cursor + index
		if cursor > bl {
			cursor = cursor - bl
		}
		return &b.Buf[cursor]
	}
	return
}

func (b *BufferUint64) Add(value uint64) {
	b.Cursor++
	if b.Cursor == len(b.Buf) {
		b.Cursor = 0
		if !b.Full {
			b.Full = true
		}
	}
	b.Buf[b.Cursor] = value
}

func (b *BufferUint64) ForEach(fn func(v uint64) error) (e error) {
	c := b.Cursor
	i := c + 1
	if i == len(b.Buf) {
		// D.Ln("hit the end")
		i = 0
	}
	if !b.Full {
		// D.Ln("buffer not yet full")
		i = 0
	}
	// D.Ln(b.Buf)
	for ; ; i++ {
		if i == len(b.Buf) {
			// D.Ln("passed the end")
			i = 0
		}
		if i == c {
			// D.Ln("reached cursor again")
			break
		}
		// D.Ln(i, b.Cursor)
		if e = fn(b.Buf[i]); E.Chk(e) {
			break
		}
	}
	return
}
