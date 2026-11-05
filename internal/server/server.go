package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eaudetcobello/tinymonitor/internal/model"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// MetricsReader reads stored metrics for one agent and kind within [from, to], ordered by time ascending.
type MetricsReader interface {
	Get(ctx context.Context, agentID uint32, kind model.Kind, from, to time.Time) ([]model.Metric, error)
}

type Server struct {
	MetricsService MetricsReader
}

// New returns an Echo instance with the middleware and routes registered.
func New(metricsService MetricsReader) *echo.Echo {
	s := &Server{
		MetricsService: metricsService,
	}
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	e.GET("/metrics", s.handleGetMetrics)

	return e
}

func Start(ctx context.Context, metricsService MetricsReader) *echo.Echo {
	e := New(metricsService)

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
	Kind    string `query:"kind"`
	From    string `query:"from"`
	To      string `query:"to"`
}

func (s *Server) handleGetMetrics(c *echo.Context) error {
	var params GetMetricsParams

	if err := c.Bind(&params); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	kind, err := model.ParseKind(params.Kind)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	from, err := time.Parse(time.RFC3339Nano, params.From)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid from timestamp")
	}

	to, err := time.Parse(time.RFC3339Nano, params.To)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid to timestamp")
	}

	if from.After(to) {
		return c.String(http.StatusBadRequest, "from must not be after to")
	}

	metrics, err := s.MetricsService.Get(
		c.Request().Context(),
		params.AgentID,
		kind,
		from,
		to,
	)
	if err != nil {
		return fmt.Errorf("metrics service get: %w", err)
	}

	return c.JSON(http.StatusOK, metrics)
}
