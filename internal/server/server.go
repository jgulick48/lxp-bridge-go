package server

import (
	"github.com/jgulick48/lxp-bridge-go/internal/models"
)

type Server interface{}

type server struct {
	config models.Config
}
