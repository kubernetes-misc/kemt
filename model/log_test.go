package model

import (
	"strconv"
	"testing"
)

func TestNewRotary(t *testing.T) {
	size := 10
	r := NewRotary(size)
	for i := 1; i <= 20; i++ {
		r.Add(strconv.Itoa(i))
	}
	count := 0
	r.ForEach(func(log Log) {
		count++
	})

	if count != size {
		t.Fatal("unexpected number of rows stored - not max of", size, "but rather", count)
	}

}
