package main

import (
	"fmt"
	"math/rand"
)

func main() {
	mainNum := []rune("0123456789")
	mainChar := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	sliceNum := []rune("0123456789")
	sliceChar := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	for i := range sliceNum {
		j := rand.Intn(i + 1)
		sliceNum[i], sliceNum[j] = sliceNum[j], sliceNum[i]
	}
	for i := range sliceChar {
		j := rand.Intn(i + 1)
		sliceChar[i], sliceChar[j] = sliceChar[j], sliceChar[i]
	}

	fmt.Printf("var mapping = map[rune]rune{\n")
	for i := 0; i < len(mainNum); i++ {
		fmt.Printf("\t'%s': '%s',\n", string(mainNum[i]), string(sliceNum[i]))
	}
	for i := 0; i < len(mainChar); i++ {
		fmt.Printf("\t'%s': '%s',\n", string(mainChar[i]), string(sliceChar[i]))
	}
	fmt.Printf("}")

	fmt.Printf("var reverseMapping = map[rune]rune{\n")
	for i := 0; i < len(mainNum); i++ {
		fmt.Printf("\t'%s': '%s',\n", string(sliceNum[i]), string(mainNum[i]))
	}
	for i := 0; i < len(mainChar); i++ {
		fmt.Printf("\t'%s': '%s',\n", string(sliceChar[i]), string(mainChar[i]))
	}
	fmt.Printf("}")
}
