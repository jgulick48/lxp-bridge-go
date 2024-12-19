package coordinator

import (
	"encoding/json"
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/jgulick48/lxp-bridge-go/internal/metrics"
	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/jgulick48/lxp-bridge-go/internal/mqtt"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sirupsen/logrus"
	"log"
	"strings"
	"time"
)

type Client interface {
	Done()
}

type client struct {
	config models.Config
	modbus.ParserCallback
	mqttClient    mqtt.Client
	modbusClients map[string]modClient
}
type modClient struct {
	modbusClient modbus.Client
	done         chan bool
}

func NewClient(config models.Config, logger *logrus.Logger) Client {
	c := client{
		config: config,
	}
	c.mqttClient = mqtt.NewClient(config.MQTTConfig, false)
	if config.StatsDConfig.StatsServer != "" {
		var err error
		metrics.Metrics, err = statsd.New(config.StatsDConfig.StatsServer)
		if err != nil {
			log.Printf("Error creating stats client %s", err.Error())
		} else {
			metrics.StatsEnabled = true
		}
	}
	go c.mqttClient.Connect()
	time.Sleep(5 * time.Second)
	c.modbusClients = make(map[string]modClient)
	for _, modConfig := range config.Inverters {
		if modConfig.ReadTimeout == 0 {
			modConfig.ReadTimeout = 300 * time.Second
		}
		if modConfig.LongRead == 0 {
			modConfig.LongRead = 60 * time.Second
		}
		if modConfig.ShortRead == 0 {
			modConfig.ShortRead = 10 * time.Second
		}
		c.modbusClients[modConfig.DataLog] = modClient{
			modbusClient: modbus.NewClient(modConfig, logger, &c),
			done:         make(chan bool),
		}
		c.modbusClients[modConfig.DataLog].modbusClient.Connect()
		go c.setupPolling(modConfig)
		if config.MQTTConfig.HomeAssistant.Enabled {
			c.reportHomeAssistant(modConfig)
		}
	}
	return &c
}

func (c *client) ReportValue(register registers.Register, value int32, dataLogger string) {
	topic := fmt.Sprintf("%s/%s/%s/%s", c.config.MQTTConfig.NameSpace, dataLogger, strings.ToLower(register.RegisterType.String()), strings.ToLower(register.ShortName))
	floatVal := float32(value)
	floatVal = floatVal * register.Multiplier
	_ = c.mqttClient.SendMessage(topic, fmt.Sprintf("%v", floatVal), false)
	if metrics.StatsEnabled {
		metrics.SendGaugeMetric(fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, strings.ToLower(register.ShortName)), []string{metrics.FormatTag("deploymentID", dataLogger)}, float64(floatVal))
	}
}

func (c *client) ReportPowerOpenEVSE(soc, batteryPower int32) {
	if !c.config.OpenEVSE.Enabled || !c.config.MQTTConfig.Enabled {
		return
	}
	exportPower := -batteryPower
	if soc < c.config.OpenEVSE.SOCChargeStart {
		exportPower = c.config.OpenEVSE.ChargeStopValue
	}
	if soc >= c.config.OpenEVSE.SOCChargeMax {
		exportPower = c.config.OpenEVSE.ChargeMaxValue
	}
	_ = c.mqttClient.SendMessage(c.config.OpenEVSE.Topic, fmt.Sprintf("%v", exportPower), true)
}

func (c *client) Done() {
	for _, inverterClient := range c.modbusClients {
		inverterClient.modbusClient.Close()
		inverterClient.done <- true
	}
	c.mqttClient.Close()
}

func (c *client) reportHomeAssistant(config models.LXPConfig) {
	//availability := map[string]string{
	//	"topic": fmt.Sprintf("%s/LWT", c.config.MQTTConfig.NameSpace),
	//}
	device := registers.Device{
		Manufacturer: "LuxPower",
		Name:         fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, config.DataLog),
		Identifiers:  []string{fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, config.DataLog)},
	}
	for _, register := range registers.InputRegisters {
		registerJson := register.ToJson(device, c.config.MQTTConfig.NameSpace, config.DataLog)
		registerString, _ := json.Marshal(registerJson)
		err := c.mqttClient.SendMessage(fmt.Sprintf("%s/sensor/%s/%s/config", strings.ToLower(c.config.MQTTConfig.HomeAssistant.Prefix), device.Name, strings.ToLower(register.ShortName)),
			registerString, false)
		if err != nil {
			logrus.Errorf("Error sending sensor config to homeassistant %s", err.Error())
		}
	}
}

func (c *client) setupPolling(inverter models.LXPConfig) {
	ticker1 := time.NewTicker(inverter.ShortRead)
	ticker2 := time.NewTicker(inverter.LongRead)
	ticker3 := time.NewTicker(20 * time.Second)
	if inverter.ShortRead == -1 {
		ticker1.Stop()
	}
	if inverter.LongRead == -1 {
		ticker2.Stop()
	}
	inverterClient, ok := c.modbusClients[inverter.DataLog]
	if !ok {
		logrus.Errorf("No configured modbus client for Data Logger %s", inverter.DataLog)
	}
	for {
		select {
		case <-ticker1.C:
			inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x4, 0, 40))
			inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x4, 40, 40))
			inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x4, 120, 40))
		case <-ticker2.C:
			inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x4, 80, 40))
		case <-ticker3.C:
			c.mqttClient.SendMessage(fmt.Sprintf("%s/LWT", c.config.MQTTConfig.NameSpace), "Online", true)
		case <-inverterClient.done:
			return
		}
	}
}
