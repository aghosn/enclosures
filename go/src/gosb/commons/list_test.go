package commons

import (
	"fmt"
	"testing"
	"unsafe"
)

type mint struct {
	ListElem
	val int
}

func (m *mint) toElem() *ListElem {
	return (*ListElem)(unsafe.Pointer(m))
}

func (e *ListElem) toMint() *mint {
	return (*mint)(unsafe.Pointer(e))
}

func toMint(e uintptr) *mint {
	return (*mint)(unsafe.Pointer(e))
}

func convert(o []int) *List {
	l := &List{}
	l.Init()
	for _, v := range o {
		m := &mint{ListElem{}, v}
		l.AddBack(m.toElem())
	}
	return l
}

func TestLists(t *testing.T) {
	// Test if list creation is correct
	original := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	newlist := convert(original)
	counter := 0
	for i, v := 0, newlist.First.toMint(); i < len(original) && v != nil; i, v = i+1, v.Next.toMint() {
		if v.val != original[i] {
			t.Errorf("Mismatched %v -- %v\n", v.val, original[i])
		}
		counter++
	}
	if counter != len(original) {
		t.Errorf("Lengths do not match %v %v\n", counter, len(original))
	}
}

func printList(l *List) {
	for v := l.First.toMint(); v != nil; v = v.Next.toMint() {
		fmt.Printf("%v ->", v.val)
	}
	fmt.Println("nil")
}

func TestInsertAfter(t *testing.T) {
	original := []int{1, 3, 5, 7, 9}
	toInsert := []int{2, 4, 6, 8, 10}
	newlist := convert(original)
	for i := range toInsert {
		m := &mint{ListElem{}, toInsert[i]}
		for v := newlist.First.toMint(); v != nil; v = v.Next.toMint() {
			if v.val == m.val-1 {
				newlist.InsertAfter(m.toElem(), v.toElem())
				break
			}
		}
	}
	counter := 0
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for i, v := 0, newlist.First.toMint(); i < len(expected) && v != nil; i, v = i+1, v.Next.toMint() {
		if v.val != expected[i] {
			t.Errorf("Error expected %d got %d\n", expected[i], v.val)
		}
		counter++
	}
	if counter != len(expected) {
		t.Errorf("Error expected %d got %d len\n", len(expected), counter)
	}
}

func TestInsertBefore(t *testing.T) {
	original := []int{2, 4, 6, 8, 10}
	toInsert := []int{1, 3, 5, 7, 9}
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	list := convert(original)
	for i := range toInsert {
		m := &mint{ListElem{}, toInsert[i]}
		for v := list.First.toMint(); v != nil; v = v.Next.toMint() {
			if v.val-1 == m.val {
				list.InsertBefore(m.toElem(), v.toElem())
			}
		}
	}
	counter := 0
	for v := list.First.toMint(); v != nil; v = v.Next.toMint() {
		if v.val != expected[counter] {
			t.Errorf("Expected %d got %d\n", expected[counter], v.val)
		}
		counter++
	}
	if counter != len(expected) {
		t.Errorf("Expected %d got %d len\n", len(expected), counter)
	}
}

func TestRemove(t *testing.T) {
	original := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	expected := []int{1, 3, 5, 7, 9}
	list := convert(original)
	for v := list.First.toMint(); v != nil; {
		if v.val%2 == 0 {
			n := v.Next.toMint()
			list.Remove(v.toElem())
			v = n
		} else {
			v = v.Next.toMint()
		}
	}
	i := 0
	for v := list.First.toMint(); v != nil; v = v.Next.toMint() {
		if v.val != expected[i] {
			t.Errorf("Expected %d got %d\n", expected[i], v.val)
		}
		i++
	}
	if i != len(expected) {
		t.Errorf("Expected %d got %d\n", len(expected), i)
	}
}
