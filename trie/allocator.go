package trie

import (
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/metrics"
)

var ArenaAlloc = metrics.NewRegisteredMeter("arenaAlloc", nil)
var ArenaFree = metrics.NewRegisteredMeter("arenaFree", nil)

var ShortPageAlloc = metrics.NewRegisteredMeter("arena/shortpagealloc", nil)
var FullPageAlloc = metrics.NewRegisteredMeter("arena/fullpagealloc", nil)
var BytePageAlloc = metrics.NewRegisteredMeter("arena/bytepagealloc", nil)
var ShortPagePut = metrics.NewRegisteredMeter("arena/shortpageput", nil)
var FullPagePut = metrics.NewRegisteredMeter("arena/fullpageput", nil)
var BytePagePut = metrics.NewRegisteredMeter("arena/bytepageput", nil)

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

const arenaPageSize = 1024

var shortsPagePool = sync.Pool{
	New: func() any {
		ShortPageAlloc.Mark(int64(unsafe.Sizeof(shortNode{}) * 1024))
		return make([]shortNode, arenaPageSize)
	},
}

var fullsPagePool = sync.Pool{
	New: func() any {
		FullPageAlloc.Mark(int64(unsafe.Sizeof(fullNode{}) * 1024))
		return make([]fullNode, arenaPageSize)
	},
}

var bytesPagePool = sync.Pool{
	New: func() any {
		BytePageAlloc.Mark(1024)
		return make([]byte, arenaPageSize)
	},
}

type ArenaNodeAllocator struct {
	usedShorts uint64
	shorts     [][]shortNode

	usedFulls uint64
	fulls     [][]fullNode

	usedBytes uint64
	bytes     [][]byte

	children []*ArenaNodeAllocator
}

func (a *ArenaNodeAllocator) NewFull() *fullNode {
	pageIndex := a.usedFulls / arenaPageSize
	pageOffset := a.usedFulls % arenaPageSize
	if pageOffset == 0 {
		a.fulls = append(a.fulls, fullsPagePool.Get().([]fullNode))
	}
	a.usedFulls++
	return &a.fulls[pageIndex][pageOffset]
}

func (a *ArenaNodeAllocator) NewShort() *shortNode {
	pageIndex := a.usedShorts / arenaPageSize
	pageOffset := a.usedShorts % arenaPageSize
	if pageOffset == 0 {
		a.shorts = append(a.shorts, shortsPagePool.Get().([]shortNode))
	}
	a.usedShorts++
	return &a.shorts[pageIndex][pageOffset]
}

func (a *ArenaNodeAllocator) NewBytes(requestLen int) []byte {
	pageIndex := a.usedBytes / arenaPageSize
	pageOffset := a.usedBytes % arenaPageSize
	overflows := arenaPageSize-pageOffset < uint64(requestLen) // len doesn't fit at the end of the page
	if pageOffset == 0 || overflows {
		if overflows { // move to the start of the new page
			pageIndex++
			pageOffset = 0
			a.usedBytes += arenaPageSize - pageOffset // use up rest of the last page
		}
		a.bytes = append(a.bytes, bytesPagePool.Get().([]byte))
	}

	a.usedBytes += uint64(requestLen)
	return a.bytes[pageIndex][pageOffset : pageOffset+uint64(requestLen)]
}

func (a *ArenaNodeAllocator) Copy() NodeAllocator {
	ArenaAlloc.Mark(1)
	childArena := &ArenaNodeAllocator{}
	a.children = append(a.children, childArena)
	return childArena
}

func (a *ArenaNodeAllocator) Reset() {
	a.usedBytes = 0
	a.usedFulls = 0
	a.usedShorts = 0
}

func (a *ArenaNodeAllocator) Free() {
	ArenaFree.Mark(1)
	for _, page := range a.shorts {
		ShortPagePut.Mark(int64(unsafe.Sizeof(shortNode{}) * 1024))
		shortsPagePool.Put(page)
	}
	a.shorts = nil
	for _, page := range a.fulls {
		FullPagePut.Mark(int64(unsafe.Sizeof(fullNode{}) * 1024))
		fullsPagePool.Put(page)
	}
	a.fulls = nil
	for _, page := range a.bytes {
		BytePagePut.Mark(int64(1024))
		bytesPagePool.Put(page)
	}
	a.bytes = nil
	for _, child := range a.children {
		child.Free()
	}
}
