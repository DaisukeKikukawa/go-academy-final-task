package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
)

// areaIDParam は :id を整数にする。失敗したら 400 を返して false。
func areaIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		errorJSON(c, http.StatusBadRequest, "bad_id", "id は整数で指定してください")
		return 0, false
	}
	return id, true
}

// loadArea は地区を取り、無ければ 404 を返して nil。
func (s *Server) loadArea(c *gin.Context, id int) *repository.Area {
	area, err := repository.GetArea(c.Request.Context(), s.DB, id)
	if errors.Is(err, sql.ErrNoRows) {
		errorJSON(c, http.StatusNotFound, "area_not_found", "指定された地区はありません")
		return nil
	}
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "db_error", err.Error())
		return nil
	}
	return area
}

// GET /v1/areas/:id/rules
func (s *Server) listRules(c *gin.Context) {
	id, ok := areaIDParam(c)
	if !ok {
		return
	}
	area := s.loadArea(c, id)
	if area == nil {
		return
	}
	rules, err := repository.ListRulesByArea(c.Request.Context(), s.DB, id)
	if err != nil {
		errorJSON(c, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if rules == nil {
		rules = []repository.CollectionRule{}
	}
	c.JSON(http.StatusOK, gin.H{"area": area, "rules": rules})
}
