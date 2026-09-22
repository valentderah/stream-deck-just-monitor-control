package display

import "testing"

func TestIDLocksSequential(t *testing.T) {
	l := NewIDLocks()
	u1 := l.Lock("a")
	u1()
	u2 := l.Lock("a")
	u2()
}
