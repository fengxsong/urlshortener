package main

import (
	"github.com/fengxsong/urlshortener/src"
	"time"
)

func main() {
	s := urlshortener.NewSrv(1024, 5, 5*time.Minute)
	s.Run(":8000")
}
