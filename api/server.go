package api

import (
	"context"
	"errors"
	"github.com/artela-network/galxe-integration/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	router *gin.Engine
	server *http.Server

	ctx  context.Context
	conf *config.APIConfig
}

func NewServer(ctx context.Context, config *config.APIConfig) *Server {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.MultiWriter(log.StandardLogger().Out)
	gin.DefaultErrorWriter = io.MultiWriter(log.StandardLogger().Out)

	r := gin.Default()

	if config.Host == "" {
		config.Host = "127.0.0.1"
	}
	if config.Port == 0 {
		config.Port = 9211
	}

	// CORS for https://galxe.com and Setup CORS to allow specific origins, methods, and headers
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://galxe.com"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(gin.RecoveryWithWriter(log.StandardLogger().Out))

	s := &Server{
		router: r,
		conf:   config,
		ctx:    ctx,
		server: &http.Server{
			Addr:    config.Host + ":" + strconv.Itoa(int(config.Port)),
			Handler: r,
		},
	}

	apiGroup := r.Group("/api")
	apiGroup.GET("/ping", s.ping)

	return s
}

func (s *Server) ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func (s *Server) Start() {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("start server fail: %v", err)
		}
	}()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Errorf("shutdown server fail: %v", err)
	} else {
		log.Info("api server has been shutdown")
	}
}
