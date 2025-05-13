package modbus

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type Client interface {
	Connect()
	SendCommand(command []byte)
	Close()
	GetTimeSinceLastMessage() time.Duration
}

type client struct {
	writer              *bufio.Writer
	reader              *bufio.Reader
	config              models.LXPConfig
	logger              *logrus.Logger
	messages            chan byte
	returnedMessages    chan byte
	commands            chan []byte
	done                chan bool
	callBack            ParserCallback
	lastMessageReceived time.Time
	mux                 sync.Mutex
}

func NewClient(config models.LXPConfig, logger *logrus.Logger, callback ParserCallback) Client {
	c := client{
		config:   config,
		messages: make(chan byte),
		commands: make(chan []byte),
		done:     make(chan bool),
		logger:   logger,
		callBack: callback,
	}
	if c.config.ReadTimeout == 0 {
		c.config.ReadTimeout = 300 * time.Second
	}
	go c.processMessages()
	go c.sendCommands()
	return &c
}

func (c *client) GetTimeSinceLastMessage() time.Duration {
	c.mux.Lock()
	defer c.mux.Unlock()
	return time.Now().Sub(c.lastMessageReceived)
}

func (c *client) Connect() {
	go func() {
		var done bool
		for !done {
			logrus.Infof("Connecting to %s:%s with timeout of %s", c.config.Host, c.config.Port, c.config.ReadTimeout)
			conn, err := net.Dial("tcp", fmt.Sprintf("%s:%v", c.config.Host, c.config.Port))
			for conn != nil {
				var message byte
				c.reader = bufio.NewReader(conn)
				c.writer = bufio.NewWriter(conn)
				select {
				case <-c.done:
					_ = conn.Close()
					conn = nil
					done = true
					continue
				default:
					for {
						_ = conn.SetDeadline(time.Now().Add(c.config.ReadTimeout))
						message, err = c.reader.ReadByte()
						if err != nil {
							fmt.Println(fmt.Sprintf("Error reading connection: %s", err.Error()))
						}
						c.messages <- message
					}
				}
			}
		}
	}()
}

func (c *client) Close() {
	close(c.messages)
	close(c.commands)
	c.done <- true
}

func (c *client) SendCommand(command []byte) {
	c.commands <- command
}

func (c *client) sendCommands() {
	for m := range c.commands {
		//timer := time.Now()
		_, err := c.writer.Write(m)
		if err != nil {
			logrus.Errorf("Error sending command: %s", err.Error())
		}
		_ = c.writer.Flush()
		//logrus.Infof("Wrote %v bytes to connection in %s %v", n, time.Now().Sub(timer), m)
		time.Sleep(time.Second)
	}
}

func (c *client) processMessages() {
	byteCount := 0
	messageLengthRemaining := uint16(0)
	message := make([]byte, 0)
	for m := range c.messages {
		message = append(message, m)
		if byteCount == 0 {
			if m != 161 {
				byteCount = 0
				message = make([]byte, 0)
				messageLengthRemaining = uint16(0)
				continue
			}
		}
		if byteCount == 1 {
			if m != 26 {
				byteCount = 0
				message = make([]byte, 0)
				messageLengthRemaining = uint16(0)
				continue
			}
		}
		if byteCount == 5 {
			messageLengthRemaining = binary.LittleEndian.Uint16(message[4:6])
		}
		if byteCount > 5 && messageLengthRemaining > 0 {
			messageLengthRemaining--
		}
		if byteCount > 5 && messageLengthRemaining == 0 {
			c.mux.Lock()
			c.lastMessageReceived = time.Now()
			c.mux.Unlock()
			_, _ = Decode(message, c.callBack)
			message = make([]byte, 0)
			byteCount = 0
			continue
		}

		byteCount++
	}
}
