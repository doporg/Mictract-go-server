package main

import (
	"mictract/global"
	_ "mictract/init"
	"mictract/router"
	"github.com/fvbock/endless"
)

func main() {
	defer global.Close()
	r := router.GetRouter()
	s := endless.NewServer("0.0.0.0:8080", r)

	_ = s.ListenAndServe()
}
