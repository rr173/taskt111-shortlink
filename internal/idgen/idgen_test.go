package idgen

import "testing"

func TestNewCodeLength(t *testing.T) {
	for _, n := range []int{4, 6, 8} {
		c, err := NewCode(n)
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != n {
			t.Fatalf("NewCode(%d) len = %d", n, len(c))
		}
	}
}

func TestUniqueCodeAvoidsCollision(t *testing.T) {
	used := map[string]bool{"abc": true}
	c, err := UniqueCode(3, 20, func(s string) bool { return used[s] })
	if err != nil {
		t.Fatal(err)
	}
	if used[c] {
		t.Fatalf("returned colliding code %q", c)
	}
}

func TestUniqueCodeRetriesExhausted(t *testing.T) {
	_, err := UniqueCode(3, 5, func(string) bool { return true })
	if err == nil {
		t.Fatal("expected error when all codes collide")
	}
}
