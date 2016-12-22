package urlshortener

import (
	"github.com/gin-gonic/gin"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	seeds       = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	seedsLength = len(seeds)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Srv struct {
	urls map[string]*Url
	pool chan string
	mu   *sync.Mutex
	cfg  struct {
		strLength  int
		expiration time.Duration
	}
}

func NewSrv(poolSize, strl int, expiration time.Duration) *Srv {
	srv := &Srv{
		urls: make(map[string]*Url),
		pool: make(chan string, poolSize),
		mu:   new(sync.Mutex),
	}
	srv.cfg.strLength = strl
	srv.cfg.expiration = expiration
	go srv.fillingPool()
	go srv.deleteExpired()
	return srv
}

func (s *Srv) fillingPool() {
	for {
		rs := s.genRandomString(s.cfg.strLength)
		if _, ok := s.urls[rs]; !ok {
			s.pool <- rs
		}
	}
}

func (s *Srv) deleteExpired() {
	for k, v := range s.urls {
		if v.isExpired() {
			delete(s.urls, k)
		}
	}
}

func (s *Srv) cleaner() {
	ticker := time.Tick(s.cfg.expiration)
	for {
		select {
		case <-ticker:
			s.deleteExpired()
		}
	}
}

func (s *Srv) genRandomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = seeds[rand.Intn(seedsLength)]
	}
	return string(bytes)
}

func (s *Srv) Set(orig string) *Url {
	uniqStr := <-s.pool
	u := &Url{
		Short:      uniqStr,
		Orig:       orig,
		Create:     time.Now(),
		Expiration: time.Now().Add(s.cfg.expiration),
	}
	s.mu.Lock()
	s.urls[uniqStr] = u
	s.mu.Unlock()
	return u
}

func (s *Srv) Get(shortUrl string) *Url {
	if u, ok := s.urls[shortUrl]; ok {
		return u
	}
	return nil
}

func (s *Srv) Gen(ctx *gin.Context) {
	var reqBody Req
	if err := ctx.BindJSON(&reqBody); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	orig := strings.TrimSpace(reqBody.Orig)
	if orig != "" && (strings.HasPrefix(orig, "http:") || strings.HasPrefix(orig, "https:")) {
		u := s.Set(orig)
		ctx.JSON(http.StatusOK, u)
		return
	}
	ctx.JSON(http.StatusNotAcceptable, gin.H{"error": "field `orig` should not be empty and it startswith `http:` or `https:`"})
}

func (s *Srv) Redirect(ctx *gin.Context) {
	short := ctx.Param("short")
	u := s.Get(short)
	if u == nil || u.isExpired() {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "invalid short url, please recheck"})
		return
	}
	stats := ctx.Query("stats")
	if stats != "" {
		ctx.JSON(http.StatusOK, u)
		return
	}
	u.Click++
	ctx.Redirect(http.StatusMovedPermanently, u.Orig)
}

func (s *Srv) Run(addr string) {
	r := gin.Default()
	r.GET("/v1/:short", s.Redirect)
	r.POST("/v1/", s.Gen)
	r.Run(addr)
}

type Req struct {
	Orig string `json:"orig"`
}

type Url struct {
	Short      string
	Orig       string
	Create     time.Time
	Click      int64
	Expiration time.Time
}

func (u *Url) isExpired() bool {
	return u.Expiration.Before(time.Now())
}
