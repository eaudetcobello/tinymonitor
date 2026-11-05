package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/service"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type Server struct {
	MetricsService *service.Metrics
}

func Start(ctx context.Context, metricsService *service.Metrics) *echo.Echo {
	s := &Server{
		MetricsService: metricsService,
	}
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/metrics", s.handleGetMetrics)

	config := echo.StartConfig{
		Address:         ":1323",
		GracefulTimeout: 10 * time.Second,
	}

	if err := config.Start(ctx, e); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}

	e.Logger.Info("Server shutting down...")

	return e
}

type GetMetricsParams struct {
	AgentID uint32 `query:"agent_id"`
	From    string `query:"from"`
	To      string `query:"to"`
}

func (s *Server) handleGetMetrics(c *echo.Context) error {
	var params GetMetricsParams

	if err := c.Bind(&params); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	from, err := time.Parse(time.RFC3339Nano, params.From)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid from timestamp")
	}

	to, err := time.Parse(time.RFC3339Nano, params.To)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid to timestamp")
	}

	metrics, err := s.MetricsService.Get(
		c.Request().Context(),
		params.AgentID,
		from,
		to,
	)
	if err != nil {
		return fmt.Errorf("metrics service get: %w", err)
	}

	return c.JSON(http.StatusOK, metrics)
}
