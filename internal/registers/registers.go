package registers

type Register struct {
	ShortName  string
	Name       string
	Location   int
	Length     int
	Multiplier float64
	Unit       string
}

// ShortRegisters Short Registers are values that can be read at a higher frequency and do not trigger a full send of all data to the remote server
var ShortRegisters = []Register{
	{
		ShortName:  "State",
		Name:       "Inverter Operating Mode",
		Location:   0,
		Length:     1,
		Multiplier: 1,
		Unit:       "",
	}, {
		ShortName:  "V_PV1",
		Name:       "Voltage for PV String 1",
		Location:   1,
		Length:     1,
		Multiplier: 0.1,
		Unit:       "V",
	}, {
		ShortName:  "V_PV2",
		Name:       "Voltage for PV String 2",
		Location:   2,
		Length:     1,
		Multiplier: 0.1,
		Unit:       "V",
	}, {
		ShortName:  "V_PV1",
		Name:       "Voltage for PV String 2",
		Location:   3,
		Length:     1,
		Multiplier: 0.1,
		Unit:       "V",
	},
}

// LongRegisters Long Registeres are values that should be polled at least 1 minute apart as they trigger the sending of all registers to the remote server
var LongRegisters = []Register{
	{
		ShortName:  "MaxChgCurr",
		Name:       "Maximum Charge Current Allowed by BMS",
		Location:   81,
		Length:     1,
		Multiplier: 0.01,
		Unit:       "A",
	},
}
