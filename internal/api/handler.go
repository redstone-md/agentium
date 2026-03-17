package api

import (
	"errors"
	"net/http"

	"agentium/internal/app"
	"agentium/internal/model"
	"agentium/internal/session"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service app.AgentiumService
}

func NewHandler(service app.AgentiumService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(e *echo.Echo) {
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	v1 := e.Group("/v1")
	v1.POST("/sessions", h.createSession)
	v1.DELETE("/sessions/:session_id", h.closeSession)
	v1.GET("/sessions/:session_id/snapshot", h.getSnapshot)
	v1.POST("/sessions/:session_id/action", h.performAction)
}

func (h *Handler) createSession(c echo.Context) error {
	var request model.SessionOptions
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	sessionID, err := h.service.CreateSession(c.Request().Context(), request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"session_id": sessionID})
}

func (h *Handler) closeSession(c echo.Context) error {
	if err := h.service.CloseSession(c.Request().Context(), c.Param("session_id")); err != nil {
		return mapError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) getSnapshot(c echo.Context) error {
	result, err := h.service.GetSnapshot(c.Request().Context(), c.Param("session_id"))
	if err != nil {
		return mapError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func (h *Handler) performAction(c echo.Context) error {
	var request model.ActionRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	result, err := h.service.PerformAction(c.Request().Context(), c.Param("session_id"), request)
	if err != nil {
		return mapError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func mapError(c echo.Context, err error) error {
	status := http.StatusBadRequest
	if errors.Is(err, session.ErrSessionNotFound) {
		status = http.StatusNotFound
	}

	return c.JSON(status, map[string]string{"error": err.Error()})
}
