package mqtt

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/sirupsen/logrus"
)

const (
	publishTimeout   = 5 * time.Second
	disconnectQuiesc = 250 // milliseconds
	closeTimeout     = 2 * time.Second
)

var errDisabled = errors.New("mqtt client not enabled")

// Client publishes to and receives commands from an MQTT broker.
type Client interface {
	// Close disconnects. It is safe to call more than once and does not block for long.
	Close()
	IsEnabled() bool
	// Connect starts connecting in the background and returns immediately.
	// The client keeps retrying and reconnecting on its own.
	Connect()
	SendMessage(topic string, value interface{}, retained bool) error
	// SubMultiple registers topics to subscribe to on every (re)connect.
	SubMultiple(topicList []string)
}

type client struct {
	config    models.MQTTConfig
	debug     bool
	callback  MessageCallback
	onConnect func()

	mqttClient mqtt.Client
	mux        sync.Mutex
	topics     []string
	closeOnce  sync.Once
}

// NewClient creates a client. When config.Host is empty the returned client is
// a disabled no-op. onConnect (may be nil) runs after every successful
// (re)connection, once command topics have been resubscribed.
func NewClient(config models.MQTTConfig, debug bool, callback MessageCallback, onConnect func()) Client {
	return &client{config: config, debug: debug, callback: callback, onConnect: onConnect}
}

func (c *client) IsEnabled() bool {
	return c.config.Host != ""
}

func (c *client) Connect() {
	if !c.IsEnabled() {
		return
	}
	broker := fmt.Sprintf("tcp://%s:%d", c.config.Host, c.config.Port)
	logrus.Infof("Connecting to %s", broker)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(c.config.ClientName)
	opts.SetUsername(c.config.Username)
	opts.SetPassword(c.config.Password)
	opts.SetDefaultPublishHandler(c.messagePubHandler)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(time.Minute)
	opts.SetOnConnectHandler(c.handleConnect)
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		logrus.Warnf("MQTT connection lost: %v (will reconnect)", err)
	})
	c.mux.Lock()
	c.mqttClient = mqtt.NewClient(opts)
	mc := c.mqttClient
	c.mux.Unlock()
	go func() {
		// With ConnectRetry the token completes once the first connection succeeds.
		if token := mc.Connect(); token.Wait() && token.Error() != nil {
			logrus.Errorf("MQTT connect failed: %v", token.Error())
		}
	}()
}

func (c *client) handleConnect(mc mqtt.Client) {
	logrus.Info("MQTT connected")
	c.mux.Lock()
	topics := append([]string(nil), c.topics...)
	c.mux.Unlock()
	if len(topics) > 0 {
		c.subscribe(mc, topics)
	}
	if c.onConnect != nil {
		c.onConnect()
	}
}

func (c *client) subscribe(mc mqtt.Client, topicList []string) {
	topics := make(map[string]byte)
	for _, topic := range topicList {
		topics[topic] = 1
	}
	token := mc.SubscribeMultiple(topics, nil)
	if !token.WaitTimeout(publishTimeout) {
		logrus.Warn("Timed out subscribing to MQTT topics")
		return
	}
	if token.Error() != nil {
		logrus.Errorf("Error subscribing to MQTT topics: %v", token.Error())
		return
	}
	logrus.Infof("Subscribed to topics: %s", strings.Join(topicList, ", "))
}

func (c *client) SubMultiple(topicList []string) {
	if !c.IsEnabled() {
		return
	}
	c.mux.Lock()
	c.topics = append(c.topics, topicList...)
	mc := c.mqttClient
	c.mux.Unlock()
	// If already connected subscribe now; otherwise OnConnect will do it.
	if mc != nil && mc.IsConnectionOpen() {
		c.subscribe(mc, topicList)
	}
}

func (c *client) SendMessage(topic string, value interface{}, retained bool) error {
	if !c.IsEnabled() {
		return errDisabled
	}
	c.mux.Lock()
	mc := c.mqttClient
	c.mux.Unlock()
	if mc == nil {
		return errors.New("mqtt client not initialized")
	}
	token := mc.Publish(topic, 0, retained, value)
	if !token.WaitTimeout(publishTimeout) {
		return errors.New("timed out publishing mqtt message")
	}
	return token.Error()
}

func (c *client) Close() {
	c.closeOnce.Do(func() {
		c.mux.Lock()
		mc := c.mqttClient
		c.mux.Unlock()
		if mc == nil {
			return
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			mc.Disconnect(disconnectQuiesc)
		}()
		select {
		case <-done:
		case <-time.After(closeTimeout):
			logrus.Warn("Timed out disconnecting from MQTT")
		}
	})
}

func (c *client) messagePubHandler(_ mqtt.Client, msg mqtt.Message) {
	c.ProcessData(msg.Topic(), msg.Payload())
}

func (c *client) ProcessData(topic string, message []byte) {
	segments := strings.Split(topic, "/")
	parser := c.GetDataParser(segments, DefaultParser)
	parser(topic, message)
	if c.debug {
		logrus.Debugf("Got message from topic: %s %s", topic, message)
	}
}

func DefaultParser(topic string, message []byte) {
}

func (c *client) GetDataParser(segments []string, defaultParser func(topic string, message []byte)) func(topic string, message []byte) {
	if len(segments) < 6 || c.callback == nil {
		return defaultParser
	}
	switch segments[1] {
	case "cmd":
		return c.callback.HandleCommand
	default:
		return defaultParser
	}
}
