package weather

import (
	"fmt"
	"testing"
	"time"

	util "github.com/janmir/go-util"
)

func TestAll(t *testing.T) {
	util.Logger("___WEATHER___")
	w := New()
	future := time.Now().Add(time.Hour * 24 * 7)
	w.Get("Yokohama", future)
	fmt.Printf("%+v", w)
}
