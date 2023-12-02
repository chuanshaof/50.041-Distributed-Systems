package main

import (
	"fmt"
)

func main() {
	counter := 0

	for {
		// if counter odd, skip
		if counter%2 == 1 {
			counter++
			continue
		}

		fmt.Println(counter)
		counter++
	}
}
