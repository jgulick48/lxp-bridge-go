package mqtt

import (
	"encoding/json"
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
}

type client struct {
	config     models.MQTTConfig
	done       chan bool
	mqttClient mqtt.Client
	messages   chan mqtt.Message
	debug      bool
	values     map[string]map[string]float64
	soc        int
}

func NewClient(config models.MQTTConfig, debug bool) Client {
	if config.Host != "" {
		client := client{
			config:   config,
			done:     make(chan bool),
			messages: make(chan mqtt.Message),
			debug:    debug,
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
	var payload models.MessageJson
	err := json.Unmarshal(message, &payload)
	if err != nil {
		return err
	}
	segments := strings.Split(topic, "/")
	parser := c.GetDataParser(segments, DefaultParser)
	parser(segments, payload)
	if c.debug {
		logrus.Infof("Got message from topic: %s %s", topic, message)
	}
	return nil
}

func DefaultParser(segments []string, message models.MessageJson) {
}
func (c *client) GetDataParser(segments []string, defaultParser func(topic []string, message models.MessageJson)) func(topic []string, message models.MessageJson) {
	if len(segments) < 4 {
		return defaultParser
	}
	switch segments[3] {
	case "1", "2", "3", "4", "all":
		return c.ParseInputs
	default:
		return defaultParser
	}

}

func (c *client) ParseInputs(segments []string, message models.MessageJson) {
	if len(segments) < 4 {
		return
	}
	for key, value := range message {

		switch key {
		case "time", "runtime", "datalog":
			continue
		case "max_chg_curr", "max_dischg_curr", "bat_current":
			switch v := value.(type) {
			case int:
				if key == "bat_current" && v > 300 {
					v = v - 656
				}
			case float64:
				if key == "bat_current" && v > 300 {
					v = v - 655.36
				}
			}
		default:
			switch v := value.(type) {
			case int:
				if key == "soc" {
					c.soc = v
				}
				if key == "p_battery" {
					power := v
					if c.soc >= 98 {
						power = 1700

					} else if c.soc <= 95 {
						power = -400
					}
					log.Printf("Sending value %v to mqtt", power)
					token := c.mqttClient.Publish(fmt.Sprintf("lxp/sensor/%s/read/power1", segments[1]), 0, true, fmt.Sprintf("%v", 0-power))
					token.Wait()
				}
			case float64:
				if key == "soc" {
					c.soc = int(v)
				}
				if key == "p_battery" {
					power := v
					if c.soc >= 98 {
						power = 1700

					} else if c.soc <= 95 {
						power = -400
					}
					log.Printf("Sending value %v to mqtt", power)
					token := c.mqttClient.Publish(fmt.Sprintf("lxp/sensor/%s/read/power1", segments[1]), 0, true, fmt.Sprintf("%v", 0-power))
					token.Wait()
				}
			}
		}
	}
}