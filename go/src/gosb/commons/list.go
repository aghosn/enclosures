package commons

import (
	"log"
	//	"unsafe"
)

type ListElem struct {
	Prev *ListElem
	Next *ListElem
	List *List
}

type List struct {
	First *ListElem
	Last  *ListElem
	Len   uint64
}

//go:nosplit
func (l *List) Init() {
	l.First = nil
	l.Last = nil
}

//go:nosplit
func (l *List) IsEmpty() bool {
	return l.First == nil
}

//go:nosplit
func (l *List) Foreach(f func(e *ListElem)) {
	for v := l.First; v != nil; v = v.Next {
		f(v)
	}
}

//go:nosplit
func (l *List) AddBack(e *ListElem) {
	if e.Prev != nil || e.Next != nil || e.List != nil {
		log.Fatalf("element already in a List! %v\n", e)
	}
	if l.Last != nil {
		l.Last.Next = e
		e.Prev = l.Last
	} else {
		l.First = e
	}
	l.Last = e
	e.List = l
	l.Len++
}

//go:nosplit
func (l *List) InsertBefore(toins, elem *ListElem) {
	if elem.List != l {
		log.Fatalf("The element is not in the given List %v %v\n", elem.List, l)
	}
	if toins.Next != nil || toins.Prev != nil || toins.List != nil {
		log.Fatalf("The provided element is already in a List!\n")
	}
	oPrev := elem.Prev
	elem.Prev = toins
	toins.Next = elem
	if oPrev != nil {
		oPrev.Next = toins
		toins.Prev = oPrev
	} else {
		if l.First != elem {
			log.Fatalf("Malformed List, this should have been equal to the elem\n")
		}
		l.First = toins
	}
	toins.List = l
	l.Len++
}

//go:nosplit
func (l *List) InsertAfter(toins, elem *ListElem) {
	if elem.List != l {
		log.Fatalf("The element is not in the given List %v %v\n", elem.List, l)
	}
	if toins.Next != nil || toins.Prev != nil || toins.List != nil {
		log.Fatalf("The provided element is already in a List!\n")
	}
	oNext := elem.Next
	elem.Next = toins
	toins.Prev = elem
	if oNext != nil {
		oNext.Prev = toins
		toins.Next = oNext
	} else {
		if l.Last != elem {
			log.Fatalf("Malformed List, this should have been equal to the elem\n")
		}
		l.Last = toins
	}
	toins.List = l
	l.Len++
}

//go:nosplit
func (l *List) Remove(e *ListElem) {
	if e.List != l {
		log.Fatalf("Removing element not in the correct List %v %v\n", e, l)
	}
	if l.First == e {
		l.First = e.Next
	} else {
		e.Prev.Next = e.Next
	}
	if l.Last == e {
		l.Last = e.Prev
	} else {
		e.Next.Prev = e.Prev
	}
	e.Next = nil
	e.Prev = nil
	e.List = nil
	l.Len--
}
