package columnstore

import (
	"sync/atomic"
	"unsafe"
)

type sentinel uint8

const (
	none sentinel = iota
	compacting
	compacted
)

// Node is a Part that is a part of a linked-list
type Node struct {
	next unsafe.Pointer
	part *Part

	sentinel sentinel // sentinel nodes contain no parts, and are to indicate the start of a new sub list
}

type PartList struct {
	next  unsafe.Pointer
	total uint64
}

// Prepend a node onto the front of the list
func (l *PartList) Prepend(part *Part) *Node {
	node := &Node{
		part: part,
	}
	for { // continue until a successful compare and swap occurs
		next := atomic.LoadPointer(&l.next)
		node.next = next
		if (*Node)(next).sentinel == compacted { // This list is apart of a compacted granule, propogate the compacted value so each subsequent Prepend can return the correct value
			node.sentinel = compacted
		}
		if atomic.CompareAndSwapPointer(&l.next, next, (unsafe.Pointer)(node)) {
			atomic.AddUint64(&l.total, 1)
			return node
		}
	}
}

// Iterate accesses every node in the list
func (l *PartList) Iterate(iterate func(*Part) bool) {
	next := atomic.LoadPointer(&l.next)
	for {
		node := (*Node)(next)
		if node == nil {
			return
		}
		if !iterate(node.part) {
			return
		}
		next = atomic.LoadPointer(&node.next)
	}
}
