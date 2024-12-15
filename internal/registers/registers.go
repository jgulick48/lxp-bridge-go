package registers

//go:generate stringer -linecomment -type InputRegister

type InputRegister int

const (
	AC_INPUT_TYPE InputRegister = iota
	AFCI_ARC_CH1
	AFCI_ARC_CH2
	AFCI_ARC_CH3
	AFCI_ARC_CH4
	AFCI_CURR_CH1
	AFCI_CURR_CH2
	AFCI_CURR_CH3
	AFCI_CURR_CH4
	AFCI_MAX_ARC_CH1
	AFCI_MAX_ARC_CH2
	AFCI_MAX_ARC_CH3
	AFCI_MAX_ARC_CH4
	BATT_CAPACITY
	BATT_CURRENT_BMS
	BATT_PARRALlEL_NUM
	BATT_STATUS_0_BMS
	BATT_STATUS_1_BMS
	BATT_STATUS_2_BMS
	BATT_STATUS_3_BMS
	BATT_STATUS_4_BMS
	BATT_STATUS_5_BMS
	BATT_STATUS_6_BMS
	BATT_STATUS_7_BMS
	BATT_STATUS_8_BMS
	BATT_STATUS_9_BMS
	BATT_STATUS_INV
	BAT_VOLT_SMPL_INV
	BMS_FW_UPDATE_STATE
	CHRG_VOLT_REF
	CYCLE_CNT_BMS
	DISCHRG_CUT_VOLT
	E_CHG_ALL
	E_CHG_DAY
	E_DISCHG_ALL
	E_DISCHG_DAY
	E_EPS_ALL
	E_EPS_ALL_L1
	E_EPS_ALL_L2
	E_EPS_DAY
	E_EPS_DAY_L1
	E_EPS_DAY_L2
	E_GEN_ALL
	E_GEN_DAY
	E_INV_ALL
	E_INV_DAY
	E_PV_1_ALL
	E_PV_2_ALL
	E_PV_3_ALL
	E_PV_ALL
	E_PV_DAY
	E_PV_DAY_1
	E_PV_DAY_2
	E_PV_DAY_3
	E_REC_ALL
	E_REC_DAY
	E_TO_GRID_ALL
	E_TO_GRID_DAY
	E_TO_USER_ALL
	E_TO_USER_DAY
	F_AC
	F_EPS
	F_GEN
	Fault_Code
	Fault_Code_BMS
	INV_RMS
	Internal_Fault
	MAX_CELL_TEMP_BMS
	MAX_CELL_VOLT_BMS
	MAX_CHG_CURR
	MAX_DISCHG_CURR
	MIN_CELL_TEMP_BMS
	MIN_CELL_VOLT_BMS
	PF
	P_Battery
	P_Charge
	P_Discharge
	P_EPS
	P_EPS_L1
	P_EPS_L2
	P_GEN
	P_INV
	P_PV
	P_PV_1
	P_PV_2
	P_PV_3
	P_REC
	P_TO_GRID
	P_TO_USER
	Runtime
	SOC
	SOH
	S_EPS
	S_EPS_L1
	S_EPS_L2
	State
	T1
	T2
	T3
	T4
	T5
	T_BAT
	T_INNER
	T_RAD_1
	T_RAD_2
	V_AC_R
	V_AC_S
	V_AC_T
	V_BAT
	V_BUS_1
	V_BUS_2
	V_BUS_P
	V_EPS_L1
	V_EPS_L2
	V_EPS_R
	V_EPS_S
	V_EPS_T
	V_GEN
	V_PV_1
	V_PV_2
	V_PV_3
	Warning_Code
	Warning_Code_BMS
)

type Register struct {
	ShortName      string
	Name           string
	Multiplier     float64
	EntityCategory string
	StateClass     string
	DeviceClass    string
	Unit           string
}

type RegisterJson struct {
	UniqueID          string            `json:"unique_id"`
	Name              string            `json:"name"`
	StateTopic        string            `json:"state_topic"`
	EntityCategory    string            `json:"entity_category,omitempty"`
	StateClass        string            `json:"state_class,omitempty"`
	DeviceClass       string            `json:"device_class,omitempty"`
	ValueTemplate     string            `json:"value_template,omitempty"`
	UnitOfMeasurement string            `json:"unit_of_measurement"`
	Availability      map[string]string `json:"availability"`
}

type Device struct {
	Manufacturer string   `json:"manufacturer"`
	Name         string   `json:"name"`
	Identifiers  []string `json:"identifiers"`
}

// InputRegisters are read only values from the inverter.
var InputRegisters = map[InputRegister]Register{
	State: {
		ShortName:      State.String(),
		Name:           "Inverter Operating Mode",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	V_PV_1: {
		ShortName:   State.String(),
		Name:        "PV Voltage (String 1)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	V_PV_2: {
		ShortName:   State.String(),
		Name:        "PV Voltage (String 2)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	V_PV_3: {
		ShortName:   State.String(),
		Name:        "PV Voltage (String 3)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	V_BAT: {
		ShortName:   State.String(),
		Name:        "Battery Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	SOC: {
		ShortName:   State.String(),
		Name:        "State of Charge",
		StateClass:  "measurement",
		DeviceClass: "battery",
		Multiplier:  1,
		Unit:        "%",
	},
	SOH: {
		ShortName:   State.String(),
		Name:        "State of Health",
		StateClass:  "measurement",
		DeviceClass: "battery",
		Multiplier:  1,
		Unit:        "%",
	},
	Internal_Fault: {
		ShortName:  State.String(),
		Name:       "Internal Falt Code",
		Multiplier: 1,
		Unit:       "",
	},
	P_PV: {
		ShortName:   State.String(),
		Name:        "PV Power (Array)",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	P_PV_1: {
		ShortName:   State.String(),
		Name:        "PV Power (String 1)",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	P_PV_2: {
		ShortName:   State.String(),
		Name:        "PV Power (String 2)",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	P_PV_3: {
		ShortName:   State.String(),
		Name:        "PV Power (String 3)",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	P_Battery: {
		ShortName:   State.String(),
		Name:        "Battery Power (discharge is negative)",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	P_Charge: {
		ShortName:   State.String(),
		Name:        "Battery Charge",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	P_Discharge: {
		ShortName:   State.String(),
		Name:        "Battery Discharge",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	V_AC_R: {
		ShortName:   State.String(),
		Name:        "Grid Voltage (R-Phase)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	V_AC_S: {
		ShortName:   State.String(),
		Name:        "Grid Voltage (S-Phase)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	V_AC_T: {
		ShortName:   State.String(),
		Name:        "Grid Voltage (T-Phase)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	F_AC: {
		ShortName:   State.String(),
		Name:        "Grid Frequency",
		StateClass:  "measurement",
		DeviceClass: "frequency",
		Multiplier:  0.01,
		Unit:        "Hz",
	},
	P_INV: {
		ShortName:   State.String(),
		Name:        "Inverter Output Power",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	P_REC: {
		ShortName:   State.String(),
		Name:        "AC Charge Power",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	INV_RMS: {
		ShortName:   State.String(),
		Name:        "Inverter Current RMS",
		StateClass:  "measurement",
		DeviceClass: "Current",
		Multiplier:  0.01,
		Unit:        "A",
	},
	PF: {
		ShortName:   State.String(),
		Name:        "Power Factor",
		StateClass:  "measurement",
		DeviceClass: "power_factor",
		Multiplier:  0.001,
		Unit:        "%",
	},
	V_EPS_R: {
		ShortName:   State.String(),
		Name:        "EPS Voltage (R-Phase)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	V_EPS_S: {
		ShortName:   State.String(),
		Name:        "EPS Voltage (S-Phase)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	V_EPS_T: {
		ShortName:   State.String(),
		Name:        "EPS Voltage (T-Phase)",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "V",
	},
	F_EPS: {
		ShortName:   State.String(),
		Name:        "EPS Frequency",
		StateClass:  "measurement",
		DeviceClass: "frequency",
		Multiplier:  0.01,
		Unit:        "Hz",
	},
	P_EPS: {
		ShortName:   State.String(),
		Name:        "Active EPS Power",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	S_EPS: {
		ShortName:   State.String(),
		Name:        "Active EPS Apparent Power",
		StateClass:  "measurement",
		DeviceClass: "apparent_power",
		Multiplier:  1,
		Unit:        "VA",
	},
	P_TO_GRID: {
		ShortName:   State.String(),
		Name:        "Export Power to Grid",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "w",
	},
	P_TO_USER: {
		ShortName:   State.String(),
		Name:        "Import Power from Grid",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "w",
	},
	E_PV_DAY: {
		ShortName:   State.String(),
		Name:        "PV Generation (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_PV_DAY_1: {
		ShortName:   State.String(),
		Name:        "PV Generation (Today) (String 1)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_PV_DAY_2: {
		ShortName:   State.String(),
		Name:        "PV Generation (Today) (String 2)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_PV_DAY_3: {
		ShortName:   State.String(),
		Name:        "PV Generation (Today) (String 3)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_INV_DAY: {
		ShortName:   State.String(),
		Name:        "Inverter Output Energy (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_REC_DAY: {
		ShortName:   State.String(),
		Name:        "AC Charge Energy (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_CHG_DAY: {
		ShortName:   State.String(),
		Name:        "Charged Energy (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_DISCHG_DAY: {
		ShortName:   State.String(),
		Name:        "Discharged Energy (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_EPS_DAY: {
		ShortName:   State.String(),
		Name:        "EPS Energy (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_TO_GRID_DAY: {
		ShortName:   State.String(),
		Name:        "Export Energy to Grid (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_TO_USER_DAY: {
		ShortName:   State.String(),
		Name:        "Import Energy from Grid (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	V_BUS_1: {
		ShortName:   State.String(),
		Name:        "Bus 1 Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "v",
	},
	V_BUS_2: {
		ShortName:   State.String(),
		Name:        "Bus 2 Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "v",
	},
	E_PV_ALL: {
		ShortName:   State.String(),
		Name:        "PV Generation (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_PV_1_ALL: {
		ShortName:   State.String(),
		Name:        "PV Generation (All Time) (String 1)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_PV_2_ALL: {
		ShortName:   State.String(),
		Name:        "PV Generation (All Time) (String 2)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_PV_3_ALL: {
		ShortName:   State.String(),
		Name:        "PV Generation (All Time) (String 3)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_INV_ALL: {
		ShortName:   State.String(),
		Name:        "Inverter Output Energy (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_REC_ALL: {
		ShortName:   State.String(),
		Name:        "AC Charge Energy (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_CHG_ALL: {
		ShortName:   State.String(),
		Name:        "Charged Energy (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_DISCHG_ALL: {
		ShortName:   State.String(),
		Name:        "Discharged Energy (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_EPS_ALL: {
		ShortName:   State.String(),
		Name:        "EPS Energy (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_TO_GRID_ALL: {
		ShortName:   State.String(),
		Name:        "Export Energy to Grid (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_TO_USER_ALL: {
		ShortName:   State.String(),
		Name:        "Import Energy from Grid (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	Fault_Code: {
		ShortName:      State.String(),
		Name:           "Fault Code",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	Warning_Code: {
		ShortName:      State.String(),
		Name:           "Warning Code",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	T_INNER: {
		ShortName:   State.String(),
		Name:        "Internal Ring Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  1,
		Unit:        "°C",
	},
	T_RAD_1: {
		ShortName:   State.String(),
		Name:        "Radiator 1 Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  1,
		Unit:        "°C",
	},
	T_RAD_2: {
		ShortName:   State.String(),
		Name:        "Radiator 2 Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  1,
		Unit:        "°C",
	},
	T_BAT: {
		ShortName:   State.String(),
		Name:        "Battery Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  1,
		Unit:        "°C",
	},
	Runtime: {
		ShortName:   State.String(),
		Name:        "Total Runtime",
		StateClass:  "total_increasing",
		DeviceClass: "duration",
		Multiplier:  1,
		Unit:        "s",
	},
	AC_INPUT_TYPE: {
		ShortName:      State.String(),
		Name:           "AC Input Type (0 = Grid, 1 = Generator)",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	MAX_CHG_CURR: {
		ShortName:   State.String(),
		Name:        "Maximum Charge Current",
		StateClass:  "measurement",
		DeviceClass: "current",
		Multiplier:  0.01,
		Unit:        "A",
	},
	MAX_DISCHG_CURR: {
		ShortName:   State.String(),
		Name:        "Maximum Discharge Current",
		StateClass:  "measurement",
		DeviceClass: "current",
		Multiplier:  0.01,
		Unit:        "A",
	},
	CHRG_VOLT_REF: {
		ShortName:   State.String(),
		Name:        "BMS Recommended Charging Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "v",
	},
	DISCHRG_CUT_VOLT: {
		ShortName:   State.String(),
		Name:        "BMS Recommended Discharge Cut-Off Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "v",
	},
	BATT_STATUS_0_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 0",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_1_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 1",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_2_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 2",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_3_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 3",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_4_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 4",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_5_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 5",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_6_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 6",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_7_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 7",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_8_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 8",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_9_BMS: {
		ShortName:      State.String(),
		Name:           "BMC Status Information 9",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_INV: {
		ShortName:      State.String(),
		Name:           "Inverter Summarized Lithium Battery Status",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_PARRALlEL_NUM: {
		ShortName:      State.String(),
		Name:           "Number of Batteries in Parallel",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_CAPACITY: {
		ShortName:      State.String(),
		Name:           "Battery Capacity in AH",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_CURRENT_BMS: {
		ShortName:   State.String(),
		Name:        "Battery Capacity in AH",
		StateClass:  "measurement",
		DeviceClass: "current",
		Multiplier:  0.01,
		Unit:        "a",
	},
	Fault_Code_BMS: {
		ShortName:      State.String(),
		Name:           "Fault Code BMS",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	Warning_Code_BMS: {
		ShortName:      State.String(),
		Name:           "Warning Code BMS",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	MAX_CELL_VOLT_BMS: {
		ShortName:   State.String(),
		Name:        "Maximum Cell Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.001,
		Unit:        "v",
	},
	MIN_CELL_VOLT_BMS: {
		ShortName:   State.String(),
		Name:        "Minimum Cell Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.001,
		Unit:        "v",
	},
	MAX_CELL_TEMP_BMS: {
		ShortName:   State.String(),
		Name:        "Maximum Cell Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  0.1,
		Unit:        "°C",
	},
	MIN_CELL_TEMP_BMS: {
		ShortName:   State.String(),
		Name:        "Minimum Cell Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  0.1,
		Unit:        "°C",
	},
	BMS_FW_UPDATE_STATE: {
		ShortName:      State.String(),
		Name:           "BMS Firmware Update State",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	CYCLE_CNT_BMS: {
		ShortName:      State.String(),
		Name:           "Number of Charge and Discharge Cycles",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BAT_VOLT_SMPL_INV: {
		ShortName:   State.String(),
		Name:        "Inverter Battery Voltage Sample",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.001,
		Unit:        "v",
	},
	T1: {
		ShortName:   State.String(),
		Name:        "12K BT Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  0.1,
		Unit:        "°C",
	},
	T2: {
		ShortName:   State.String(),
		Name:        "12K BT Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  0.1,
		Unit:        "°C",
	},
	T3: {
		ShortName:   State.String(),
		Name:        "12K BT Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  0.1,
		Unit:        "°C",
	},
	T4: {
		ShortName:   State.String(),
		Name:        "12K BT Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  0.1,
		Unit:        "°C",
	},
	T5: {
		ShortName:   State.String(),
		Name:        "12K BT Temperature",
		StateClass:  "measurement",
		DeviceClass: "temperature",
		Multiplier:  0.1,
		Unit:        "°C",
	},
	V_BUS_P: {
		ShortName:   State.String(),
		Name:        "Half Bus Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "v",
	},
	V_GEN: {
		ShortName:   State.String(),
		Name:        "Generator Voltage",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "v",
	},
	F_GEN: {
		ShortName:   State.String(),
		Name:        "Generator Frequency",
		StateClass:  "measurement",
		DeviceClass: "frequency",
		Multiplier:  0.01,
		Unit:        "Hz",
	},
	P_GEN: {
		ShortName:   State.String(),
		Name:        "Generator Power",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	E_GEN_DAY: {
		ShortName:   State.String(),
		Name:        "Generator Import Power (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_GEN_ALL: {
		ShortName:   State.String(),
		Name:        "Generator Import Power (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	V_EPS_L1: {
		ShortName:   State.String(),
		Name:        "EPS Voltage L1",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "v",
	},
	V_EPS_L2: {
		ShortName:   State.String(),
		Name:        "EPS Voltage L2",
		StateClass:  "measurement",
		DeviceClass: "voltage",
		Multiplier:  0.1,
		Unit:        "v",
	},
	P_EPS_L1: {
		ShortName:   State.String(),
		Name:        "Active EPS Power L1",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	P_EPS_L2: {
		ShortName:   State.String(),
		Name:        "Active EPS Power L2",
		StateClass:  "measurement",
		DeviceClass: "power",
		Multiplier:  1,
		Unit:        "W",
	},
	S_EPS_L1: {
		ShortName:   State.String(),
		Name:        "Active EPS Apparent Power L1",
		StateClass:  "measurement",
		DeviceClass: "apparent_power",
		Multiplier:  1,
		Unit:        "VA",
	},
	S_EPS_L2: {
		ShortName:   State.String(),
		Name:        "Active EPS Apparent Power L2",
		StateClass:  "measurement",
		DeviceClass: "apparent_power",
		Multiplier:  1,
		Unit:        "VA",
	},
	E_EPS_DAY_L1: {
		ShortName:   State.String(),
		Name:        "EPS Output Energy L1 (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_EPS_DAY_L2: {
		ShortName:   State.String(),
		Name:        "EPS Output Energy L2 (Today)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_EPS_ALL_L1: {
		ShortName:   State.String(),
		Name:        "EPS Output Energy L1 (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	E_EPS_ALL_L2: {
		ShortName:   State.String(),
		Name:        "EPS Output Energy L2 (All Time)",
		StateClass:  "total_increasing",
		DeviceClass: "energy",
		Multiplier:  0.1,
		Unit:        "kwh",
	},
	AFCI_CURR_CH1: {
		ShortName:   State.String(),
		Name:        "AFCI Current Channel 1",
		StateClass:  "measurement",
		DeviceClass: "current",
		Multiplier:  0.001,
		Unit:        "a",
	},
	AFCI_CURR_CH2: {
		ShortName:   State.String(),
		Name:        "AFCI Current Channel 2",
		StateClass:  "measurement",
		DeviceClass: "current",
		Multiplier:  0.001,
		Unit:        "a",
	},
	AFCI_CURR_CH3: {
		ShortName:   State.String(),
		Name:        "AFCI Current Channel 3",
		StateClass:  "measurement",
		DeviceClass: "current",
		Multiplier:  0.001,
		Unit:        "a",
	},
	AFCI_CURR_CH4: {
		ShortName:   State.String(),
		Name:        "AFCI Current Channel 4",
		StateClass:  "measurement",
		DeviceClass: "current",
		Multiplier:  0.001,
		Unit:        "a",
	},
	AFCI_ARC_CH1: {
		ShortName:      State.String(),
		Name:           "Real Time ARC of CH1",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_ARC_CH2: {
		ShortName:      State.String(),
		Name:           "Real Time ARC of CH2",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_ARC_CH3: {
		ShortName:      State.String(),
		Name:           "Real Time ARC of CH3",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_ARC_CH4: {
		ShortName:      State.String(),
		Name:           "Real Time ARC of CH4",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_MAX_ARC_CH1: {
		ShortName:      State.String(),
		Name:           "Max ARC of CH1",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_MAX_ARC_CH2: {
		ShortName:      State.String(),
		Name:           "Max ARC of CH2",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_MAX_ARC_CH3: {
		ShortName:      State.String(),
		Name:           "Max ARC of CH3",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_MAX_ARC_CH4: {
		ShortName:      State.String(),
		Name:           "Max ARC of CH4",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
}
