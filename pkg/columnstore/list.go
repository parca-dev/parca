package columnstore

import (
	"sync/atomic"
	"unsafe"
)

// Node is a Part that is a part of a linked-list
type Node struct {
	next unsafe.Pointer
	part *Part
}

type List struct {
	next  unsafe.Pointer
	total uint64
}

// Prepend a node onto the front of the list
func (l *List) Prepend(part *Part) {
	node := &Node{
		part: part,
	}
	for { // continue until a successful compare and swap occurs
		next := atomic.LoadPointer(&l.next)
		node.next = next
		if atomic.CompareAndSwapPointer(&l.next, next, (unsafe.Pointer)(node)) {
			atomic.AddUint64(&l.total, 1)
			return
		}
	}
}

// Iterate accesses every node in the list
func (l *List) Iterate(iterate func(*Part) bool) {
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
