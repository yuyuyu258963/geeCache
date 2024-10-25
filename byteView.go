package main

// A byteView holds an  immutable view of bytes
type ByteView struct {
	b []byte
}

// Len returns the view's length
func (v ByteView) Len() int { return len(v.b) }

// ByteSlice returns a copy of the data as a byte slice
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String returns the data as a string, making a copy if necessary
func (v ByteView) String() string {
	return string(v.b)
}

// 将数据拷贝一份返回是为了避免返回的是同一个slice，用户修改的时候，直接修改了缓存中的数据
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
