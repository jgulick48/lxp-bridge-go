package modbus

import "github.com/jgulick48/lxp-bridge-go/internal/registers"

type ParserCallback interface {
	ReportValue(register registers.Register, value int32, dataLogger string)
	ReportPowerOpenEVSE(soc, batteryPower int32)
}

type BaseParserCallback struct{}

func (cb *BaseParserCallback) ReportValue(register registers.Register, value int32, dataLogger string) {
}
func (cb *BaseParserCallback) ReportPowerOpenEVSE(soc, batteryPower int32) {}
