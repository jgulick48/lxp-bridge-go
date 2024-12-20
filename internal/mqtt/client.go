package mqtt

import (
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/sirupsen/logrus"
	"log"
	"strings"
)

type Client interface {
	Close()
	IsEnabled() bool
	Connect()
	SendMessage(topic string, value interface{}, retained bool) error
	SubMultiple(topicList []string)
}

type client struct {
	config     models.MQTTConfig
	done       chan bool
	mqttClient mqtt.Client
	messages   chan mqtt.Message
	debug      bool
	values     map[string]map[string]float64
	soc        int
	callback   MessageCallback
}

func NewClient(config models.MQTTConfig, debug bool, callback MessageCallback) Client {
	if config.Host != "" {
		client := client{
			config:   config,
			done:     make(chan bool),
			messages: make(chan mqtt.Message),
			debug:    debug,
			callback: callback,
		}
		return &client
	}
	return &client{config: config}
}

func (c *client) Close() {
	c.done <- true
}

func (c *client) IsEnabled() bool {
	return c.config.Host != ""
}

func (c *client) Connect() {
	go func() {
		for message := range c.messages {
			c.ProcessData(message.Topic(), message.Payload())
		}
	}()
	logrus.Infof("Connecting to %s", fmt.Sprintf("tcp://%s:%d", c.config.Host, c.config.Port))
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.config.Host, c.config.Port))
	opts.SetClientID(c.config.ClientName)
	opts.SetDefaultPublishHandler(c.messagePubHandler)
	opts.SetUsername(c.config.Username)
	opts.SetPassword(c.config.Password)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = c.connectLostHandler
	c.mqttClient = mqtt.NewClient(opts)
	if token := c.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer c.mqttClient.Disconnect(250)
	c.keepAlive()
}

func (c *client) keepAlive() {
	for {
		select {
		case <-c.done:
			close(c.messages)
			return
		}
	}
}

func (c *client) SubMultiple(topicList []string) {
	topics := make(map[string]byte)
	for _, topic := range topicList {
		topics[topic] = 1
	}
	token := c.mqttClient.SubscribeMultiple(topics, nil)
	token.Wait()
	log.Printf("Subscribed to topics: %s", strings.Join(topicList, ", "))
}

func (c *client) SendMessage(topic string, value interface{}, retained bool) error {
	if c.mqttClient != nil {
		token := c.mqttClient.Publish(topic, 0, retained, value)
		token.Wait()
		return token.Error()
	}
	return errors.New("mqtt client not initialized")
}

func (c *client) messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	c.messages <- msg
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	logrus.Infof("Connected")
}

func (c *client) connectLostHandler(client mqtt.Client, err error) {
	logrus.Infof("Connect lost: %v", err)
	c.done <- true
}

func (c *client) ProcessData(topic string, message []byte) error {
	segments := strings.Split(topic, "/")
	parser := c.GetDataParser(segments, DefaultParser)
	parser(topic, message)
	if c.debug {
		logrus.Infof("Got message from topic: %s %s", topic, message)
	}
	return nil
}

func DefaultParser(topic string, message []byte) {
}
func (c *client) GetDataParser(segments []string, defaultParser func(topic string, message []byte)) func(topic string, message []byte) {
	if len(segments) < 6 {
		return defaultParser
	}
	switch segments[1] {
	case "cmd":
		return c.callback.HandleCommand
	default:
		return defaultParser
	}

}
