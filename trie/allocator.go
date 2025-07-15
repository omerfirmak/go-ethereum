package trie

type NodeAllocator interface {
	//
	NewFull() *fullNode

	//
	NewShort() *shortNode

	//
	NewBytes(int) []byte

	//
	Copy() NodeAllocator
}

func MakeFullNode(allocator NodeAllocator, v *fullNode) *fullNode {
	f := allocator.NewFull()
	*f = *v
	return f
}

func MakeShortNode(allocator NodeAllocator, v *shortNode) *shortNode {
	s := allocator.NewShort()
	*s = *v
	return s
}

func MakeValueNode(allocator NodeAllocator, v []byte) valueNode {
	b := allocator.NewBytes(len(v))
	copy(b, v)
	return b
}

type GcNodeAllocator struct{}

func (GcNodeAllocator) NewFull() *fullNode      { return new(fullNode) }
func (GcNodeAllocator) NewShort() *shortNode    { return new(shortNode) }
func (GcNodeAllocator) NewBytes(len int) []byte { return make([]byte, len) }
func (GcNodeAllocator) Copy() NodeAllocator     { return GcNodeAllocator{} }
