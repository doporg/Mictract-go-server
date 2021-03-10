package main

import (
	"github.com/fvbock/endless"
	"mictract/global"
	_ "mictract/init"
	"mictract/router"
)

func main() {
	defer global.Close()
	r := router.GetRouter()
	s := endless.NewServer("0.0.0.0:8080", r)

	_ = s.ListenAndServe()
}
