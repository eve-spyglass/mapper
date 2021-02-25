package main

import "log"

func main() {

	em := NewEveMapper()
	err := em.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}
