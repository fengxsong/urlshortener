package main

import (
	"github.com/fengxsong/urlshortener/src"
	"time"
)

func main() {
	s, err := urlshortener.NewSrv(1024, 5, 5*time.Minute)
	if err != nil {
		panic(err)
	}
	s.Run(":8000")
}
