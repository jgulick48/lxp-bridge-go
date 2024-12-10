package modbus

import (
	"fmt"
	"log"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/models"
	mod2 "github.com/simonvetter/modbus"
	"github.com/sirupsen/logrus"
)

type Client interface {
	ReadInputs1() error
}

type client struct {
	client *mod2.ModbusClient
	logger *log.Logger
}

func NewClient(config models.LXPConfig, logger *logrus.Logger) Client {
	handler := mod2.NewClient( mod2.ClientConfiguration{
		URL:           "",
		Speed:         0,
		DataBits:      0,
		Parity:        0,
		StopBits:      0,
		Timeout:       0,
		TLSClientCert: nil,
		TLSRootCAs:    nil,
		Logger:        nil,
	}
		fmt.Sprintf("%s:%v", config.Host, config.Port))
	handler.Timeout = time.Duration(config.ReadTimeout) * time.Second
	return &client{
		client: handler,
	}
}

func (c *client) ReadInputs1() error {
	err := c.client.Connect()
	defer c.client.Close()
	if err != nil {
		return err
	}
	modClient := mod2.NewClient(c.client)
	results, err := modClient.ReadInputRegisters(1, 1)
	if err != nil {
		return err
	}
	for i, result := range results {
		c.logger.Printf("Read Input Register %v: %02X", i, result)
	}
	return nil
}
