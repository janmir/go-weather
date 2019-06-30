package main

import (
	"log"
	"fmt"
	weather "go-weather/pkg"
	"time"
)

func main() {
	w := weather.New()
	future := time.Now().Add(time.Hour * 24 * 7)
	err := w.Get("Nishikubocho", future)
	if err != nil{
		log.Fatal(err)
	}
	fmt.Printf("%+v", w)
}
