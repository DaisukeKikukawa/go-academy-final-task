// Package handler は HTTP の入口。リクエストを読み、repository / rule を呼び、JSON を返す。
package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Server はハンドラが共有するもの(今は DB だけ)。
type Server struct {
	DB *sql.DB
}

// NewRouter は全ルートを登録した gin のエンジンを返す。
func NewRouter(s *Server) *gin.Engine {
	r := gin.Default()

	r.GET("/healthz", s.healthz)

	v1 := r.Group("/v1")
	{
		v1.GET("/municipalities", s.listMunicipalities)
		v1.GET("/municipalities/:code/areas", s.searchAreas)
		v1.GET("/areas/:id", s.getArea)
		v1.GET("/areas/:id/schedule", s.getSchedule)
		v1.GET("/areas/:id/today", s.getToday)
	}
	return r
}

// errorJSON は全エンドポイント共通のエラー形式 {"error": {"code": ..., "message": ...}} を返す。
func errorJSON(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

func (s *Server) healthz(c *gin.Context) {
	if err := s.DB.PingContext(c.Request.Context()); err != nil {
		errorJSON(c, http.StatusServiceUnavailable, "db_unavailable", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
