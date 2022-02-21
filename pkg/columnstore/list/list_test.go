package list

import (
	"fmt"
	"testing"
)

func Test_SingleList(t *testing.T) {
	l := List{}

	for i := 0; i < 10; i++ {
		l.Prepend(&Node{
			part: i,
		})
	}

	l.Iterate(func(i int) bool {
		fmt.Println(i)
		return true
	})
}
