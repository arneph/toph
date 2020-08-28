package db

import (
	"testing"
)

func TestDBConcurrentReadWrite(t *testing.T) {
	db := NewDB()
	db.Write("abc", "A")
	for i := 0; i < 3; i++ {
		go func() {
			value := db.Read("abc")
			if value != "A" && value != "X" {
				t.Errorf("db value for 'abc' does not match, expected 'A' or 'X', got: %q", value)
			}
		}()
	}
	go db.Write("abc", "X")
	go db.WriteIfAbsent("abc", "Y")
}
