package urlshortener

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

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
	if u == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "invalid short url, please recheck"})
		return
	} else if u.isExpired() {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "short url is expired"})
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
	r.GET("/v1/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, s.String())
	})
	r.GET("/v1/:short", s.Redirect)
	r.POST("/v1/", s.Gen)
	r.Run(addr)
}

type Req struct {
	Orig string `json:"orig"`
}
