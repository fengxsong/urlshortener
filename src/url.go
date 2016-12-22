package urlshortener

import (
	"bytes"
	"encoding/gob"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	seeds       = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	seedsLength = len(seeds)
	gobf        = "urlshortener.gob"
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

func NewSrv(poolSize, strl int, expiration time.Duration) (*Srv, error) {
	srv := &Srv{
		urls: make(map[string]*Url),
		pool: make(chan string, poolSize),
		mu:   new(sync.Mutex),
	}
	srv.cfg.strLength = strl
	srv.cfg.expiration = expiration
	err := srv.load(gobf)
	if err != nil {
		return nil, err
	}
	go srv.fillingPool()
	go srv.cleaner()
	go srv.dump(gobf)
	return srv, nil
}

func (s *Srv) fillingPool() {
	for {
		rs := s.genRandomString(s.cfg.strLength)
		if _, ok := s.urls[rs]; !ok {
			s.pool <- rs
		}
	}
}

func (s *Srv) dump(fn string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-c:
			s.deleteExpired()
			if len(s.urls) != 0 {
				f, err := os.Create(fn)
				if err != nil {
					panic(err)
				}
				defer f.Close()
				enc := gob.NewEncoder(f)
				err = enc.Encode(s.urls)
				if err != nil {
					panic(err)
				}
			} else {
				os.Remove(fn)
			}
			os.Exit(1)
		}
	}
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func (s *Srv) load(fn string) error {
	if isExist(fn) {
		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		defer f.Close()
		urls := make(map[string]*Url)
		dec := gob.NewDecoder(f)
		err = dec.Decode(&urls)
		if err != nil {
			return err
		}
		s.urls = urls
	}
	return nil
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

func (s *Srv) reset() {
	s.urls = make(map[string]*Url)
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

func (s *Srv) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("KEY\tEXPIRE\n------------------\n")
	for k, v := range s.urls {
		buf.WriteString(k)
		buf.WriteString("\t")
		buf.WriteString(strconv.FormatBool(v.isExpired()))
		buf.WriteString("\n")
	}
	return buf.String()
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
