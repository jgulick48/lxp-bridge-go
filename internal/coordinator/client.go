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
	"strconv"
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
	config       models.LXPConfig
	done         chan bool
}

func NewClient(config models.Config, logger *logrus.Logger) Client {
	c := client{
		config: config,
	}
	c.mqttClient = mqtt.NewClient(config.MQTTConfig, false, &c)
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
	subTopics := make([]string, 0, len(config.Inverters))
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
			config:       modConfig,
		}
		c.modbusClients[modConfig.DataLog].modbusClient.Connect()
		go c.setupPolling(modConfig)
		if config.MQTTConfig.HomeAssistant.Enabled {
			c.reportHomeAssistant(modConfig)
		}
		subTopics = append(subTopics, fmt.Sprintf("%s/cmd/%s/set/#", config.MQTTConfig.NameSpace, modConfig.DataLog))
	}
	c.mqttClient.SubMultiple(subTopics)
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

func (c *client) HandleCommand(topic string, message []byte) {
	segments := strings.Split(topic, "/")
	if len(segments) < 6 {
		return
	}
	if segments[0] != c.config.MQTTConfig.NameSpace {
		return
	}
	if segments[1] != "cmd" {
		return
	}
	if inverter, ok := c.modbusClients[segments[2]]; ok {
		if segments[3] == "set" {
			if segments[4] == "hold" {
				registerID, err := strconv.Atoi(segments[5])
				if err != nil {
					return
				}
				if holdRegister, ok := registers.HoldIDToRegisterMap[registerID]; ok {
					register := registers.HoldRegisters[holdRegister]
					if register.RegisterLength == 0 {
						value, err := strconv.Atoi(string(message))
						if err != nil {
							return
						}
						bitValue := modbus.GetBytesForUInt16(uint16(value))
						command := modbus.BuildPacket(inverter.config.DataLog, inverter.config.Serial, 1, 0x6, uint16(registerID), bitValue...)
						inverter.modbusClient.SendCommand(command)
					}
				}
			}
		}
	}
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
	for id, register := range registers.InputRegisters {
		registerJson := register.ToJson(int(id), device, c.config.MQTTConfig.NameSpace, config.DataLog)
		registerString, _ := json.Marshal(registerJson)
		err := c.mqttClient.SendMessage(fmt.Sprintf("%s/sensor/%s/%s/config", strings.ToLower(c.config.MQTTConfig.HomeAssistant.Prefix), device.Name, strings.ToLower(register.ShortName)),
			registerString, false)
		if err != nil {
			logrus.Errorf("Error sending sensor config to homeassistant %s", err.Error())
		}
	}
	for id, register := range registers.HoldRegisters {
		registerJson := register.ToJson(int(id), device, c.config.MQTTConfig.NameSpace, config.DataLog)
		registerString, _ := json.Marshal(registerJson)
		err := c.mqttClient.SendMessage(fmt.Sprintf("%s/%s/%s/%s/config", strings.ToLower(c.config.MQTTConfig.HomeAssistant.Prefix), strings.ToLower(register.HomeAssistantType.String()), device.Name, strings.ToLower(register.ShortName)),
			registerString, false)
		if err != nil {
			logrus.Errorf("Error sending sensor config to homeassistant %s", err.Error())
		}
	}
}

func (c *client) setupPolling(inverter models.LXPConfig) {
	var ticker1, ticker2, ticker3 *time.Ticker
	ticker3 = time.NewTicker(20 * time.Second)
	if inverter.ShortRead == -1*time.Second {
		ticker1 = time.NewTicker(time.Hour)
		ticker1.Stop()
	} else {
		ticker1 = time.NewTicker(inverter.ShortRead)
	}
	if inverter.LongRead == -1*time.Second {
		ticker2 = time.NewTicker(time.Hour)
		ticker2.Stop()
	} else {
		ticker2 = time.NewTicker(inverter.LongRead)
	}
	inverterClient, ok := c.modbusClients[inverter.DataLog]
	if !ok {
		logrus.Errorf("No configured modbus client for Data Logger %s", inverter.DataLog)
	}
	defaultDataLength := modbus.GetBytesForUInt16(40)
	time.Sleep(5 * time.Second)
	if inverter.PublishHoldingsOnConnect {
		inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x3, 0, defaultDataLength...))
		inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x3, 40, defaultDataLength...))
		inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x3, 80, defaultDataLength...))
		inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x3, 120, defaultDataLength...))
		inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x3, 160, defaultDataLength...))
	}
	for {
		select {
		case <-ticker1.C:
			logrus.Info("Polling for inputs 1, 2, and 4")
			inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x4, 0, defaultDataLength...))
			inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x4, 40, defaultDataLength...))
			inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x4, 120, defaultDataLength...))
		case <-ticker2.C:
			logrus.Info("Polling for input 3")
			inverterClient.modbusClient.SendCommand(modbus.BuildPacket(inverter.DataLog, inverter.Serial, 0, 0x3, 160, defaultDataLength...))
		case <-ticker3.C:
			c.mqttClient.SendMessage(fmt.Sprintf("%s/LWT", c.config.MQTTConfig.NameSpace), "Online", true)
		case <-inverterClient.done:
			return
		}
	}
}
