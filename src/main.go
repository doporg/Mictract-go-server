package main

import (
	"github.com/fvbock/endless"
	ii "mictract/init"
	"mictract/router"
)

func main() {
	//(&kubernetes.Tools{}).Create()
	//(&kubernetes.Mysql{}).Create()
	//
	//time.Sleep(20 * time.Second)
	defer ii.Close()
	// TODO: start mysql and tools
	r := router.GetRouter()
	s := endless.NewServer("0.0.0.0:8080", r)

	_ = s.ListenAndServe()
}
