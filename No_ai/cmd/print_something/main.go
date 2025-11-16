package main

import (
	"fmt"
)

type cat struct {
	name           string
	colorPrimary   string
	colorSecondary string
	numberLegs     int
	numberEyes     int
}

func newCat(name, colorPrimary, colorSecondary string) *cat {

	c := cat{name: name, colorPrimary: colorPrimary, colorSecondary: colorSecondary}
	c.numberLegs = 4
	c.numberEyes = 2
	return &c
}

func main() {
	fmt.Println("Hello World")
	kitty := newCat("mittens", "black", "yellow")

	kittyInfo := fmt.Sprintf("Name: %s, Color Primary: %s, Color Secondary: %s", kitty.name, kitty.colorPrimary, kitty.colorSecondary)
	fmt.Println(kittyInfo)
}
