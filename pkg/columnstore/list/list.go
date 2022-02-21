package list

import (
	"sync/atomic"
	"unsafe"
)

// Node is a Part that is a part of a linked-list
type Node struct {
	next unsafe.Pointer
	part int
}

type List Node

// Prepend a node onto the front of the list
func (l *List) Prepend(node *Node) {
	for { // continue until a successful compare and swap occurs
		next := atomic.LoadPointer(&l.next)
		node.next = next
		if atomic.CompareAndSwapPointer(&l.next, next, (unsafe.Pointer)(node)) {
			return
		}
	}
}

// Iterate accesses every node in the list
func (l *List) Iterate(iterate func(int) bool) {
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
