package main

import (
	"fmt"
	weather "go-weather/pkg"
	"time"
)

func main() {
	w := weather.New()
	future := time.Now().Add(time.Hour * 24 * 7)
	w.Get("Nishikubocho", future)
	fmt.Printf("%+v", w)
}
