package modbus

import (
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ClientTestSuite struct {
	suite.Suite
	c client
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) SetupTest() {
	suite.c = client{
		config:   models.LXPConfig{},
		messages: make(chan byte),
		commands: make(chan []byte),
	}
	go func() {
		suite.c.processMessages()
	}()
}

func (suite *ClientTestSuite) TestLongInput() {
	testdata := []byte{}
	for i := range testdata {
		suite.c.messages <- testdata[i]
	}
}

func (suite *ClientTestSuite) TeardownAllSuite() {
	close(suite.c.messages)
	close(suite.c.commands)
}
