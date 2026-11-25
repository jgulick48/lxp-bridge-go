package modbus

import "github.com/jgulick48/lxp-bridge-go/internal/registers"

// RawFrameCallback is implemented by callbacks that want each valid read reply (function 3 or 4) as the
// complete frame the dongle sent.
type RawFrameCallback interface {
	ReportRawFrame(dataLogger string, function byte, register uint16, frame []byte)
}

type ParserCallback interface {
	ReportValue(register registers.Register, value int32, dataLogger string)
	ReportPowerOpenEVSE(soc, batteryPower int32)
}

type BaseParserCallback struct{}

func (cb *BaseParserCallback) ReportValue(register registers.Register, value int32, dataLogger string) {
}
func (cb *BaseParserCallback) ReportPowerOpenEVSE(soc, batteryPower int32) {}
