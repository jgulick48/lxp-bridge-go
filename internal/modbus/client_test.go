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
		config: models.LXPConfig{
			Host:    "10.0.20.10",
			Port:    "8000",
			DataLog: "BA31101550",
			Serial:  "3192670065",
		},
		messages: make(chan byte),
		commands: make(chan []byte),
		callBack: &TestLogger{},
	}
	go func() {
		suite.c.processMessages()
	}()
}

func (suite *ClientTestSuite) TestLongInput() {

}

func (suite *ClientTestSuite) TeardownAllSuite() {
	close(suite.c.messages)
	close(suite.c.commands)
	suite.c.Close()
}
