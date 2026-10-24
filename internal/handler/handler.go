package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DaisukeKikukawa/go-academy-final-task/internal/repository"
	"github.com/DaisukeKikukawa/go-academy-final-task/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/v1/municipalities", h.listMunicipalities)
	r.GET("/v1/municipalities/:code/areas", h.searchAreas)
	r.GET("/v1/areas/:id", h.getArea)
	r.GET("/v1/areas/:id/today", h.getToday)
	r.GET("/v1/areas/:id/schedule", h.getSchedule)
}

func (h *Handler) listMunicipalities(c *gin.Context) {
	list, err := h.svc.Municipalities()
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"municipalities": list})
}

func (h *Handler) searchAreas(c *gin.Context) {
	list, err := h.svc.SearchAreas(c.Param("code"), c.Query("q"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"areas": list})
}

func (h *Handler) getArea(c *gin.Context) {
	a, err := h.svc.Area(areaID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *Handler) getToday(c *gin.Context) {
	res, err := h.svc.Today(areaID(c), time.Now())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) getSchedule(c *gin.Context) {
	from, err1 := parseDate(c.Query("from"), service.StartOfDay(time.Now()))
	to, err2 := parseDate(c.Query("to"), from.AddDate(0, 0, 29))
	if err1 != nil || err2 != nil || to.Before(from) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from と to は YYYY-MM-DD で、to は from 以降にしてください"})
		return
	}
	res, err := h.svc.Schedule(areaID(c), from, to)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func areaID(c *gin.Context) int {
	id, _ := strconv.Atoi(c.Param("id"))
	return id
}

func parseDate(s string, def time.Time) (time.Time, error) {
	if s == "" {
		return def, nil
	}
	return time.ParseInLocation("2006-01-02", s, service.JST)
}

func fail(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "指定された地区はありません"})
		return
	}
	c.AbortWithStatus(http.StatusInternalServerError)
}
