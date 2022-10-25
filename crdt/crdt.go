package crdt

import "fmt"

type CRDT interface {
	Insert(position int, value string) (string, error)
	Delete(position int) string
}

func IsCRDT(c CRDT) {
	// temporary code to check if the CRDT works.
	fmt.Println(c.Insert(1, "a"))
}
