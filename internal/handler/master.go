package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

// GET /v1/municipalities
func (s *Server) listMunicipalities(c *gin.Context) {
	ms, err := repository.ListMunicipalities(c.Request.Context(), s.DB)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"municipalities": ms})
}

// GET /v1/municipalities/:code/areas?q=大井&limit=50
func (s *Server) searchAreas(c *gin.Context) {
	code := c.Param("code")
	q := c.Query("q")
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 1 || limit > 500 {
		errorJSON(c, http.StatusBadRequest, "bad_limit", "limit は 1〜500 の整数で指定してください")
		return
	}

	areas, err := repository.SearchAreas(c.Request.Context(), s.DB, code, q, limit)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if areas == nil {
		areas = []repository.Area{} // 0件でも null ではなく [] を返す
	}
	c.JSON(http.StatusOK, gin.H{"areas": areas})
}

// GET /v1/areas/:id
func (s *Server) getArea(c *gin.Context) {
	id, ok := areaIDParam(c)
	if !ok {
		return
	}
	if area := s.loadArea(c, id); area != nil {
		c.JSON(http.StatusOK, area)
	}
}
