package main

import (
	"log"
	"math/rand"
	"os"
)

func f1() {
	panic("panic") // want "use of panic detected"
}

func f2() {
	log.Fatal("fatal error") // want "log.Fatal used outside main.main"
}

func f3() {
	os.Exit(1) // want "os.Exit used outside main.main"
}

func main() {
	r := rand.New(rand.NewSource(rand.Int63()))
	n := r.Intn(10)

	if n%2 == 0 {
		log.Fatal("fatal error") // no warning
	} else {
		os.Exit(1) // no warning
	}
}
