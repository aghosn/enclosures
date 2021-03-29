package runtime

//go:notinheap
type sbSpanList struct {
	first *mspan
	last  *mspan
}

//go:notinheap
type spanExtras struct {
	id    int // package id
	oldid int
	dirty bool
	move  bool
	inext *mspan      // next entry in list
	iprev *mspan      // previous entry in list
	ilist *sbSpanList // link back to the list

	// Allocator cache for tiny objects w/o pointers.
	// See "Tiny allocator" comment in malloc.go.

	// tiny points to the beginning of the current tiny block, or
	// nil if there is no current tiny block.
	//
	// tiny is a heap pointer. Since mcache is in non-GC'd memory,
	// we handle it by clearing it in releaseAll during mark
	// termination.
	tiny       uintptr
	tinyoffset uintptr
}

// setId takes care of calling our backend if we want to switch a section.
// The move boolean is true if we are moving the span from one section to another.
// It is false for newly mmapped sections.
//go:nosplit
func (e *mspan) setId(id int, move bool) {
	mp := acquirem()
	if gp := getg(); gp.sbid != _OUT_MODE && id == -1 {
		throw("Setid -1 inside an enclosure")
	}
	if e.id == id || !bloatInitDone {
		e.id = id
		releasem(mp)
		return
	}
	if mp.toclean != nil {
		throw("Already have stuff to clean")
	}
	if e.dirty && e.id != id {
		throw("Nested dirty span")
	}
	e.dirty = true
	e.oldid = e.id
	e.id = id
	e.move = move
	mp.toclean = e
	mp.nester = mp.locks - 1
	releasem(mp)
}

func (e *mspan) DoRegistration() {
	if !e.dirty {
		panic("we fucked up")
	}
	if e.move && bloatInitDone {
		transferSection(e.oldid, e.id, e.startAddr, e.npages<<_PageShift)
	} else if !e.move && registerSection != nil {
		registerSection(e.id, e.startAddr, e.limit-e.startAddr)
	}
	e.dirty = false
}

// Initialize an empty doubly-linked list.
func (list *sbSpanList) init() {
	list.first = nil
	list.last = nil
}

func (list *sbSpanList) pop() *mspan {
	s := list.first
	if s != nil {
		s.tiny = 0
		s.tinyoffset = 0
		list.remove(s)
	}
	return s
}

func (list *sbSpanList) popOrEmpty() *mspan {
	s := list.pop()
	if s == nil {
		return &emptymspan
	}
	return s
}

func (list *sbSpanList) getId(id int) *mspan {
	for s := list.first; s != nil; s = s.inext {
		if s.id == id || s.allocCount == 0 {
			if s.id != id {
				s.setId(id, true)
			}
			return s
		}
	}
	return nil
}

func (list *sbSpanList) getIdOrEmpty(id int) *mspan {
	s := list.getId(id)
	if s != nil {
		return s
	}
	return &emptymspan
}

func (list *sbSpanList) remove(span *mspan) {
	if span == nil || span == &emptymspan {
		panic("Nil remove")
	}
	if span.ilist != list {
		print("runtime: failed sbSpanList.remove span.npages=", span.npages,
			" span=", span, " prev=", span.iprev, " span.ilist=", span.ilist, " list=", list, "\n")
		throw("sbSpanList.remove")
	}
	if span.tiny != 0 || span.tinyoffset != 0 {
		throw("Freeing span without resetting tiny and offset")
	}
	if list.first == span {
		list.first = span.inext
	} else {
		span.iprev.inext = span.inext
	}
	if list.last == span {
		list.last = span.iprev
	} else {
		span.inext.iprev = span.iprev
	}
	span.inext = nil
	span.iprev = nil
	span.ilist = nil
}

func (list *sbSpanList) isEmpty() bool {
	if (list.first == nil || list.last == nil) && (list.first != list.last) {
		throw("List is malformed")
	}
	return list.first == nil
}

//TODO(aghosn) do it so that it stays sorted?
func (list *sbSpanList) insert(span *mspan) {
	if span == nil || span == &emptymspan {
		panic("Insert nil")
	}
	if span.inext != nil || span.iprev != nil || span.ilist != nil {
		println("runtime: failed sbSpanList.insert", span, span.inext, span.iprev, span.ilist)
		throw("sbSpanList.insert")
	}
	span.inext = list.first
	if list.first != nil {
		// The list contains at least one span; link it in.
		// The last span in the list doesn't change.
		list.first.iprev = span
	} else {
		// The list contains no spans, so this is also the last span.
		list.last = span
	}
	list.first = span
	span.ilist = list
}

/* For mcache */
func (c *mcache) allocWithId(id int, spc spanClass) *mspan {
	return c.alloc[spc].getIdOrEmpty(id)
}
