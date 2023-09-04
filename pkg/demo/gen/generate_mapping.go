package main

import (
	"fmt"
	"math/rand"
)

func main() {
	main := []rune("0123456789")
	//main := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	slice := []rune("0123456789")
	//slice := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}

	for i := 0; i < len(main); i++ {
		fmt.Printf("\t'%s': '%s',\n", string(main[i]), string(slice[i]))
	}
}
