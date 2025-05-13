package registers

import (
	"fmt"
	"strings"
)

//go:generate go run golang.org/x/tools/cmd/stringer -linecomment -type InputRegister
//go:generate go run golang.org/x/tools/cmd/stringer -linecomment -type HoldRegister
//go:generate go run golang.org/x/tools/cmd/stringer -linecomment -type RegisterType
//go:generate go run golang.org/x/tools/cmd/stringer -linecomment -type HomeAssistantType

type InputRegister int
type HoldRegister int
type RegisterType int
type HomeAssistantType int

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
	MAX_CELL_TEMP
	MAX_CELL_VOLT
	MAX_CHG_CURR
	MAX_DISCHG_CURR
	MIN_CELL_TEMP
	MIN_CELL_VOLT
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

const (
	Input RegisterType = iota
	Hold
)

const (
	Charge_Power_Percent_CMD HoldRegister = iota
	Connect_Time
	Gen_Chg_Start_Volt
	Gen_Chg_End_Volt
	Gen_Chg_Start_SOC
	Gen_Chg_End_SOC
	Gen_Rate_power
	Grid_Volt_Conn_Low
	Grid_Volt_Conn_High
	Grid_Freq_Conn_Low
	Grid_Freq_Conn_High
	Language
	Max_Gen_Chg_Bat_Curr
	Reconnect_Time
	Start_PV_Volt
	PV_Input_Model
)

const (
	Sensor HomeAssistantType = iota
	Switch
	Number
	Text
)

type Register struct {
	ShortName string
	RegisterType
	HomeAssistantType
	Writeable      bool
	RegisterHAKey  string
	RegisterLength int
	Name           string
	Multiplier     float32
	EntityCategory string
	StateClass     string
	DeviceClass    string
	Pattern        string
	Unit           string
	Min            int
	Max            int
	Step           float32
}

func (r *Register) ToJson(id int, device Device, namespace, datalogger string) RegisterJson {
	var commandTopic string
	if r.RegisterType == Hold {
		if r.Writeable {
			commandTopic = fmt.Sprintf("%s/cmd/%s/set/hold/%v", namespace, datalogger, HoldRegisterToIDMap[HoldRegister(id)])
		}

	}
	idName := strings.ToLower(r.RegisterHAKey)
	if idName == "" {
		idName = strings.ToLower(r.ShortName)
	}
	return RegisterJson{
		UniqueID:          fmt.Sprintf("%s_%s", device.Name, idName),
		Device:            device,
		Name:              r.Name,
		StateTopic:        fmt.Sprintf("%s/%s/%s/%s", namespace, datalogger, strings.ToLower(r.RegisterType.String()), strings.ToLower(r.ShortName)),
		CommandTopic:      commandTopic,
		EntityCategory:    r.EntityCategory,
		StateClass:        r.StateClass,
		DeviceClass:       r.DeviceClass,
		ValueTemplate:     "",
		UnitOfMeasurement: r.Unit,
		Min:               float32(r.Min) * r.Multiplier,
		Max:               float32(r.Max) * r.Multiplier,
		Step:              r.Step * r.Multiplier,
	}
}

type RegisterJson struct {
	UniqueID          string            `json:"unique_id"`
	Device            Device            `json:"device"`
	Name              string            `json:"name"`
	StateTopic        string            `json:"state_topic"`
	CommandTopic      string            `json:"command_topic,omitempty"`
	EntityCategory    string            `json:"entity_category,omitempty"`
	StateClass        string            `json:"state_class,omitempty"`
	DeviceClass       string            `json:"device_class,omitempty"`
	ValueTemplate     string            `json:"value_template,omitempty"`
	UnitOfMeasurement string            `json:"unit_of_measurement"`
	Availability      map[string]string `json:"availability,omitempty"`
	Min               float32           `json:"min,omitempty"`
	Max               float32           `json:"max,omitempty"`
	Step              float32           `json:"step,omitempty"`
	Pattern           string            `json:"pattern,omitempty"`
}

type Device struct {
	Manufacturer string   `json:"manufacturer"`
	Name         string   `json:"name"`
	Identifiers  []string `json:"identifiers"`
}

var HoldIDToRegisterMap = map[int]HoldRegister{
	16:  Language,
	20:  PV_Input_Model,
	22:  Start_PV_Volt,
	23:  Connect_Time,
	24:  Reconnect_Time,
	25:  Grid_Volt_Conn_Low,
	26:  Grid_Volt_Conn_High,
	27:  Grid_Freq_Conn_Low,
	28:  Grid_Freq_Conn_High,
	64:  Charge_Power_Percent_CMD,
	194: Gen_Chg_Start_Volt,
	195: Gen_Chg_End_Volt,
	196: Gen_Chg_Start_SOC,
	197: Gen_Chg_End_SOC,
	198: Max_Gen_Chg_Bat_Curr,
}

var HoldRegisterToIDMap = map[HoldRegister]int{
	Language:                 16,
	PV_Input_Model:           20,
	Start_PV_Volt:            22,
	Connect_Time:             23,
	Reconnect_Time:           24,
	Grid_Volt_Conn_Low:       25,
	Grid_Volt_Conn_High:      26,
	Grid_Freq_Conn_Low:       27,
	Grid_Freq_Conn_High:      28,
	Charge_Power_Percent_CMD: 64,
	Gen_Chg_Start_Volt:       194,
	Gen_Chg_End_Volt:         195,
	Gen_Chg_Start_SOC:        196,
	Gen_Chg_End_SOC:          197,
	Max_Gen_Chg_Bat_Curr:     198,
}

// HoldRegisters are settings that can be changed on the inverter
var HoldRegisters = map[HoldRegister]Register{
	Language: {
		RegisterType:      Hold,
		ShortName:         Language.String(),
		Name:              "Language (0=English 1=German)",
		HomeAssistantType: Number,
		Multiplier:        1,
	},
	PV_Input_Model: {
		RegisterType:      Hold,
		ShortName:         PV_Input_Model.String(),
		Name:              "PV Input Mode",
		HomeAssistantType: Number,
		Multiplier:        1,
	},
	Charge_Power_Percent_CMD: {
		RegisterType:      Hold,
		ShortName:         Charge_Power_Percent_CMD.String(),
		Name:              "Charging power percentage setting",
		HomeAssistantType: Number,
		Multiplier:        1,
	},
	Start_PV_Volt: {
		RegisterType:      Hold,
		ShortName:         Start_PV_Volt.String(),
		Name:              "PV Working Starting Voltage",
		HomeAssistantType: Number,
		Multiplier:        0.1,
		Unit:              "V",
		Min:               900,
		Max:               5000,
		Step:              1,
	},
	Connect_Time: {
		RegisterType:      Hold,
		ShortName:         Connect_Time.String(),
		Name:              "Grid Connection Waiting Time",
		HomeAssistantType: Number,
		Multiplier:        1,
		Unit:              "s",
		Min:               30,
		Max:               600,
		Step:              1,
	},
	Reconnect_Time: {
		RegisterType:      Hold,
		ShortName:         Reconnect_Time.String(),
		Name:              "Reconnection Waiting Time",
		HomeAssistantType: Number,
		Multiplier:        1,
		Unit:              "s",
		Min:               0,
		Max:               900,
		Step:              1,
	},
	Gen_Chg_Start_Volt: {
		RegisterType:      Hold,
		ShortName:         Gen_Chg_Start_Volt.String(),
		Name:              "Generator Charging Start Battery Voltage",
		HomeAssistantType: Number,
		Multiplier:        0.1,
		Unit:              "V",
		Min:               384,
		Max:               520,
		Step:              1,
		Writeable:         true,
	},
	Gen_Chg_End_Volt: {
		RegisterType:      Hold,
		ShortName:         Gen_Chg_End_Volt.String(),
		Name:              "Generator Charging End Battery Voltage",
		HomeAssistantType: Number,
		Multiplier:        0.1,
		Unit:              "V",
		Min:               480,
		Max:               590,
		Step:              1,
		Writeable:         true,
	},
	Gen_Chg_Start_SOC: {
		RegisterType:      Hold,
		ShortName:         Gen_Chg_Start_SOC.String(),
		Name:              "Generator Charging Start Battery SOC",
		HomeAssistantType: Number,
		Multiplier:        1,
		Unit:              "%",
		Min:               0,
		Max:               90,
		Step:              1,
		Writeable:         true,
	},
	Gen_Chg_End_SOC: {
		RegisterType:      Hold,
		ShortName:         Gen_Chg_End_SOC.String(),
		Name:              "Generator Charging End Battery SOC",
		HomeAssistantType: Number,
		Multiplier:        1,
		Unit:              "%",
		Min:               20,
		Max:               100,
		Step:              1,
		Writeable:         true,
	},
}

// InputRegisters are read only values from the inverter.
var InputRegisters = map[InputRegister]Register{
	State: {
		RegisterType:   Input,
		ShortName:      State.String(),
		Name:           "Inverter Operating Mode",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	V_PV_1: {
		RegisterType: Input,
		ShortName:    V_PV_1.String(),
		Name:         "PV Voltage (String 1)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_PV_2: {
		RegisterType: Input,
		ShortName:    V_PV_2.String(),
		Name:         "PV Voltage (String 2)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_PV_3: {
		RegisterType: Input,
		ShortName:    V_PV_3.String(),
		Name:         "PV Voltage (String 3)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_BAT: {
		RegisterType: Input,
		ShortName:    V_BAT.String(),
		Name:         "Battery Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	SOC: {
		RegisterType: Input,
		ShortName:    SOC.String(),
		Name:         "State of Charge",
		StateClass:   "measurement",
		DeviceClass:  "battery",
		Multiplier:   1,
		Unit:         "%",
	},
	SOH: {
		RegisterType: Input,
		ShortName:    SOH.String(),
		Name:         "State of Health",
		StateClass:   "measurement",
		DeviceClass:  "battery",
		Multiplier:   1,
		Unit:         "%",
	},
	Internal_Fault: {
		ShortName:  Internal_Fault.String(),
		Name:       "Internal Falt Code",
		Multiplier: 1,
		Unit:       "",
	},
	P_PV: {
		RegisterType: Input,
		ShortName:    P_PV.String(),
		Name:         "PV Power (Array)",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_PV_1: {
		RegisterType: Input,
		ShortName:    P_PV_1.String(),
		Name:         "PV Power (String 1)",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_PV_2: {
		RegisterType: Input,
		ShortName:    P_PV_2.String(),
		Name:         "PV Power (String 2)",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_PV_3: {
		RegisterType: Input,
		ShortName:    P_PV_3.String(),
		Name:         "PV Power (String 3)",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_Battery: {
		RegisterType: Input,
		ShortName:    P_Battery.String(),
		Name:         "Battery Power (discharge is negative)",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_Charge: {
		RegisterType: Input,
		ShortName:    P_Charge.String(),
		Name:         "Battery Charge",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_Discharge: {
		RegisterType: Input,
		ShortName:    P_Discharge.String(),
		Name:         "Battery Discharge",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	V_AC_R: {
		RegisterType: Input,
		ShortName:    V_AC_R.String(),
		Name:         "Grid Voltage (R-Phase)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_AC_S: {
		RegisterType: Input,
		ShortName:    V_AC_S.String(),
		Name:         "Grid Voltage (S-Phase)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_AC_T: {
		RegisterType: Input,
		ShortName:    V_AC_T.String(),
		Name:         "Grid Voltage (T-Phase)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	F_AC: {
		RegisterType: Input,
		ShortName:    F_AC.String(),
		Name:         "Grid Frequency",
		StateClass:   "measurement",
		DeviceClass:  "frequency",
		Multiplier:   0.01,
		Unit:         "Hz",
	},
	P_INV: {
		RegisterType: Input,
		ShortName:    P_INV.String(),
		Name:         "Inverter Output Power",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_REC: {
		RegisterType: Input,
		ShortName:    P_REC.String(),
		Name:         "AC Charge Power",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	INV_RMS: {
		RegisterType: Input,
		ShortName:    INV_RMS.String(),
		Name:         "Inverter Current RMS",
		StateClass:   "measurement",
		DeviceClass:  "Current",
		Multiplier:   0.01,
		Unit:         "A",
	},
	PF: {
		RegisterType: Input,
		ShortName:    PF.String(),
		Name:         "Power Factor",
		StateClass:   "measurement",
		DeviceClass:  "power_factor",
		Multiplier:   0.001,
		Unit:         "%",
	},
	V_EPS_R: {
		RegisterType: Input,
		ShortName:    V_EPS_R.String(),
		Name:         "EPS Voltage (R-Phase)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_EPS_S: {
		RegisterType: Input,
		ShortName:    V_EPS_S.String(),
		Name:         "EPS Voltage (S-Phase)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_EPS_T: {
		RegisterType: Input,
		ShortName:    V_EPS_T.String(),
		Name:         "EPS Voltage (T-Phase)",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	F_EPS: {
		RegisterType: Input,
		ShortName:    F_EPS.String(),
		Name:         "EPS Frequency",
		StateClass:   "measurement",
		DeviceClass:  "frequency",
		Multiplier:   0.01,
		Unit:         "Hz",
	},
	P_EPS: {
		RegisterType: Input,
		ShortName:    P_EPS.String(),
		Name:         "Active EPS Power",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	S_EPS: {
		RegisterType: Input,
		ShortName:    S_EPS.String(),
		Name:         "Active EPS Apparent Power",
		StateClass:   "measurement",
		DeviceClass:  "apparent_power",
		Multiplier:   1,
		Unit:         "VA",
	},
	P_TO_GRID: {
		RegisterType: Input,
		ShortName:    P_TO_GRID.String(),
		Name:         "Export Power to Grid",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_TO_USER: {
		RegisterType: Input,
		ShortName:    P_TO_USER.String(),
		Name:         "Import Power from Grid",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	E_PV_DAY: {
		RegisterType:  Input,
		ShortName:     E_PV_DAY.String(),
		RegisterHAKey: "pv_generation_today",
		Name:          "PV Generation (Today)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_PV_DAY_1: {
		RegisterType:  Input,
		ShortName:     E_PV_DAY_1.String(),
		RegisterHAKey: "pv_generation_today_string_1",
		Name:          "PV Generation (Today) (String 1)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_PV_DAY_2: {
		RegisterType:  Input,
		ShortName:     E_PV_DAY_2.String(),
		RegisterHAKey: "pv_generation_today_string_2",
		Name:          "PV Generation (Today) (String 2)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_PV_DAY_3: {
		RegisterType:  Input,
		ShortName:     E_PV_DAY_3.String(),
		RegisterHAKey: "pv_generation_today_string_3",
		Name:          "PV Generation (Today) (String 3)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_INV_DAY: {
		RegisterType: Input,
		ShortName:    E_INV_DAY.String(),
		Name:         "Inverter Output Energy (Today)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_REC_DAY: {
		RegisterType: Input,
		ShortName:    E_REC_DAY.String(),
		Name:         "AC Charge Energy (Today)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_CHG_DAY: {
		RegisterType:  Input,
		ShortName:     E_CHG_DAY.String(),
		RegisterHAKey: "battery_charge_today",
		Name:          "Battery Charged Energy (Today)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_DISCHG_DAY: {
		RegisterType:  Input,
		ShortName:     E_DISCHG_DAY.String(),
		RegisterHAKey: "battery_discharge_today",
		Name:          "Battery Discharged Energy (Today)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_EPS_DAY: {
		RegisterType: Input,
		ShortName:    E_EPS_DAY.String(),
		Name:         "EPS Energy (Today)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_TO_GRID_DAY: {
		RegisterType: Input,
		ShortName:    E_TO_GRID_DAY.String(),
		Name:         "Export Energy to Grid (Today)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_TO_USER_DAY: {
		RegisterType: Input,
		ShortName:    E_TO_USER_DAY.String(),
		Name:         "Import Energy from Grid (Today)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	V_BUS_1: {
		RegisterType: Input,
		ShortName:    V_BUS_1.String(),
		Name:         "Bus 1 Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_BUS_2: {
		RegisterType: Input,
		ShortName:    V_BUS_2.String(),
		Name:         "Bus 2 Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	E_PV_ALL: {
		RegisterType:  Input,
		ShortName:     E_PV_ALL.String(),
		RegisterHAKey: "pv_generation_all_time",
		Name:          "PV Generation (All Time)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_PV_1_ALL: {
		RegisterType:  Input,
		ShortName:     E_PV_1_ALL.String(),
		RegisterHAKey: "pv_generation_all_time_string_1",
		Name:          "PV Generation (All Time) (String 1)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_PV_2_ALL: {
		RegisterType:  Input,
		ShortName:     E_PV_2_ALL.String(),
		RegisterHAKey: "pv_generation_all_time_string_2",
		Name:          "PV Generation (All Time) (String 2)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_PV_3_ALL: {
		RegisterType:  Input,
		ShortName:     E_PV_3_ALL.String(),
		RegisterHAKey: "pv_generation_all_time_string_3",
		Name:          "PV Generation (All Time) (String 3)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_INV_ALL: {
		RegisterType: Input,
		ShortName:    E_INV_ALL.String(),
		Name:         "Inverter Output Energy (All Time)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_REC_ALL: {
		RegisterType: Input,
		ShortName:    E_REC_ALL.String(),
		Name:         "AC Charge Energy (All Time)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_CHG_ALL: {
		RegisterType:  Input,
		ShortName:     E_CHG_ALL.String(),
		RegisterHAKey: "battery_charge_all_time",
		Name:          "Battery Charged Energy (All Time)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_DISCHG_ALL: {
		RegisterType:  Input,
		ShortName:     E_DISCHG_ALL.String(),
		RegisterHAKey: "battery_discharge_all_time",
		Name:          "Battery Discharged Energy (All Time)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_EPS_ALL: {
		RegisterType: Input,
		ShortName:    E_EPS_ALL.String(),
		Name:         "EPS Energy (All Time)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_TO_GRID_ALL: {
		RegisterType: Input,
		ShortName:    E_TO_GRID_ALL.String(),
		Name:         "Export Energy to Grid (All Time)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_TO_USER_ALL: {
		RegisterType: Input,
		ShortName:    E_TO_USER_ALL.String(),
		Name:         "Import Energy from Grid (All Time)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	Fault_Code: {
		RegisterType:   Input,
		ShortName:      Fault_Code.String(),
		Name:           "Fault Code",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	Warning_Code: {
		RegisterType:   Input,
		ShortName:      Warning_Code.String(),
		Name:           "Warning Code",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	T_INNER: {
		RegisterType: Input,
		ShortName:    T_INNER.String(),
		Name:         "Internal Ring Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   1,
		Unit:         "°C",
	},
	T_RAD_1: {
		RegisterType: Input,
		ShortName:    T_RAD_1.String(),
		Name:         "Radiator 1 Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   1,
		Unit:         "°C",
	},
	T_RAD_2: {
		RegisterType: Input,
		ShortName:    T_RAD_2.String(),
		Name:         "Radiator 2 Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   1,
		Unit:         "°C",
	},
	T_BAT: {
		RegisterType: Input,
		ShortName:    T_BAT.String(),
		Name:         "Battery Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   1,
		Unit:         "°C",
	},
	Runtime: {
		RegisterType: Input,
		ShortName:    Runtime.String(),
		Name:         "Total Runtime",
		StateClass:   "total_increasing",
		DeviceClass:  "duration",
		Multiplier:   1,
		Unit:         "s",
	},
	AC_INPUT_TYPE: {
		RegisterType:   Input,
		ShortName:      AC_INPUT_TYPE.String(),
		Name:           "AC Input Type (0 = Grid, 1 = Generator)",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	MAX_CHG_CURR: {
		RegisterType: Input,
		ShortName:    MAX_CHG_CURR.String(),
		Name:         "Maximum Charge Current",
		StateClass:   "measurement",
		DeviceClass:  "current",
		Multiplier:   0.1,
		Unit:         "A",
	},
	MAX_DISCHG_CURR: {
		RegisterType: Input,
		ShortName:    MAX_DISCHG_CURR.String(),
		Name:         "Maximum Discharge Current",
		StateClass:   "measurement",
		DeviceClass:  "current",
		Multiplier:   0.1,
		Unit:         "A",
	},
	CHRG_VOLT_REF: {
		RegisterType: Input,
		ShortName:    CHRG_VOLT_REF.String(),
		Name:         "BMS Recommended Charging Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	DISCHRG_CUT_VOLT: {
		RegisterType: Input,
		ShortName:    DISCHRG_CUT_VOLT.String(),
		Name:         "BMS Recommended Discharge Cut-Off Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	BATT_STATUS_0_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_0_BMS.String(),
		Name:           "BMC Status Information 0",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_1_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_1_BMS.String(),
		Name:           "BMC Status Information 1",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_2_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_2_BMS.String(),
		Name:           "BMC Status Information 2",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_3_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_3_BMS.String(),
		Name:           "BMC Status Information 3",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_4_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_4_BMS.String(),
		Name:           "BMC Status Information 4",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_5_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_5_BMS.String(),
		Name:           "BMC Status Information 5",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_6_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_6_BMS.String(),
		Name:           "BMC Status Information 6",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_7_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_7_BMS.String(),
		Name:           "BMC Status Information 7",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_8_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_8_BMS.String(),
		Name:           "BMC Status Information 8",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_9_BMS: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_9_BMS.String(),
		Name:           "BMC Status Information 9",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_STATUS_INV: {
		RegisterType:   Input,
		ShortName:      BATT_STATUS_INV.String(),
		Name:           "Inverter Summarized Lithium Battery Status",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_PARRALlEL_NUM: {
		RegisterType:   Input,
		ShortName:      BATT_PARRALlEL_NUM.String(),
		Name:           "Number of Batteries in Parallel",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_CAPACITY: {
		RegisterType:   Input,
		ShortName:      BATT_CAPACITY.String(),
		Name:           "Battery Capacity in AH",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BATT_CURRENT_BMS: {
		RegisterType: Input,
		ShortName:    BATT_CURRENT_BMS.String(),
		Name:         "Battery Capacity in AH",
		StateClass:   "measurement",
		DeviceClass:  "current",
		Multiplier:   0.1,
		Unit:         "A",
	},
	Fault_Code_BMS: {
		RegisterType:   Input,
		ShortName:      Fault_Code_BMS.String(),
		Name:           "Fault Code BMS",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	Warning_Code_BMS: {
		RegisterType:   Input,
		ShortName:      Warning_Code_BMS.String(),
		Name:           "Warning Code BMS",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	MAX_CELL_VOLT: {
		RegisterType: Input,
		ShortName:    MAX_CELL_VOLT.String(),
		Name:         "Maximum Cell Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.001,
		Unit:         "V",
	},
	MIN_CELL_VOLT: {
		RegisterType: Input,
		ShortName:    MIN_CELL_VOLT.String(),
		Name:         "Minimum Cell Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.001,
		Unit:         "V",
	},
	MAX_CELL_TEMP: {
		RegisterType: Input,
		ShortName:    MAX_CELL_TEMP.String(),
		Name:         "Maximum Cell Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   0.1,
		Unit:         "°C",
	},
	MIN_CELL_TEMP: {
		RegisterType: Input,
		ShortName:    MIN_CELL_TEMP.String(),
		Name:         "Minimum Cell Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   0.1,
		Unit:         "°C",
	},
	BMS_FW_UPDATE_STATE: {
		RegisterType:   Input,
		ShortName:      BMS_FW_UPDATE_STATE.String(),
		Name:           "BMS Firmware Update State",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	CYCLE_CNT_BMS: {
		RegisterType:   Input,
		ShortName:      CYCLE_CNT_BMS.String(),
		Name:           "Number of Charge and Discharge Cycles",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	BAT_VOLT_SMPL_INV: {
		RegisterType: Input,
		ShortName:    BAT_VOLT_SMPL_INV.String(),
		Name:         "Inverter Battery Voltage Sample",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.001,
		Unit:         "V",
	},
	T1: {
		RegisterType: Input,
		ShortName:    T1.String(),
		Name:         "12K BT Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   0.1,
		Unit:         "°C",
	},
	T2: {
		RegisterType: Input,
		ShortName:    T2.String(),
		Name:         "12K BT Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   0.1,
		Unit:         "°C",
	},
	T3: {
		RegisterType: Input,
		ShortName:    T3.String(),
		Name:         "12K BT Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   0.1,
		Unit:         "°C",
	},
	T4: {
		RegisterType: Input,
		ShortName:    T4.String(),
		Name:         "12K BT Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   0.1,
		Unit:         "°C",
	},
	T5: {
		RegisterType: Input,
		ShortName:    T5.String(),
		Name:         "12K BT Temperature",
		StateClass:   "measurement",
		DeviceClass:  "temperature",
		Multiplier:   0.1,
		Unit:         "°C",
	},
	V_BUS_P: {
		RegisterType: Input,
		ShortName:    V_BUS_P.String(),
		Name:         "Half Bus Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_GEN: {
		RegisterType: Input,
		ShortName:    V_GEN.String(),
		Name:         "Generator Voltage",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	F_GEN: {
		RegisterType: Input,
		ShortName:    F_GEN.String(),
		Name:         "Generator Frequency",
		StateClass:   "measurement",
		DeviceClass:  "frequency",
		Multiplier:   0.01,
		Unit:         "Hz",
	},
	P_GEN: {
		RegisterType: Input,
		ShortName:    P_GEN.String(),
		Name:         "Generator Power",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	E_GEN_DAY: {
		RegisterType:  Input,
		ShortName:     E_GEN_DAY.String(),
		RegisterHAKey: "energy_of_generator_all_today",
		Name:          "Energy Imported From Generator (Today)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	E_GEN_ALL: {
		RegisterType:  Input,
		ShortName:     E_GEN_ALL.String(),
		RegisterHAKey: "energy_of_generator_all_time",
		Name:          "Energy Imported From Generator (All Time)",
		StateClass:    "total_increasing",
		DeviceClass:   "energy",
		Multiplier:    0.1,
		Unit:          "kWh",
	},
	V_EPS_L1: {
		RegisterType: Input,
		ShortName:    V_EPS_L1.String(),
		Name:         "EPS Voltage L1",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	V_EPS_L2: {
		RegisterType: Input,
		ShortName:    V_EPS_L2.String(),
		Name:         "EPS Voltage L2",
		StateClass:   "measurement",
		DeviceClass:  "voltage",
		Multiplier:   0.1,
		Unit:         "V",
	},
	P_EPS_L1: {
		RegisterType: Input,
		ShortName:    P_EPS_L1.String(),
		Name:         "Active EPS Power L1",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	P_EPS_L2: {
		RegisterType: Input,
		ShortName:    P_EPS_L2.String(),
		Name:         "Active EPS Power L2",
		StateClass:   "measurement",
		DeviceClass:  "power",
		Multiplier:   1,
		Unit:         "W",
	},
	S_EPS_L1: {
		RegisterType: Input,
		ShortName:    S_EPS_L1.String(),
		Name:         "Active EPS Apparent Power L1",
		StateClass:   "measurement",
		DeviceClass:  "apparent_power",
		Multiplier:   1,
		Unit:         "VA",
	},
	S_EPS_L2: {
		RegisterType: Input,
		ShortName:    S_EPS_L2.String(),
		Name:         "Active EPS Apparent Power L2",
		StateClass:   "measurement",
		DeviceClass:  "apparent_power",
		Multiplier:   1,
		Unit:         "VA",
	},
	E_EPS_DAY_L1: {
		RegisterType: Input,
		ShortName:    E_EPS_DAY_L1.String(),
		Name:         "EPS Output Energy L1 (Today)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_EPS_DAY_L2: {
		RegisterType: Input,
		ShortName:    E_EPS_DAY_L2.String(),
		Name:         "EPS Output Energy L2 (Today)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_EPS_ALL_L1: {
		RegisterType: Input,
		ShortName:    E_EPS_ALL_L1.String(),
		Name:         "EPS Output Energy L1 (All Time)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	E_EPS_ALL_L2: {
		RegisterType: Input,
		ShortName:    E_EPS_ALL_L2.String(),
		Name:         "EPS Output Energy L2 (All Time)",
		StateClass:   "total_increasing",
		DeviceClass:  "energy",
		Multiplier:   0.1,
		Unit:         "kWh",
	},
	AFCI_CURR_CH1: {
		RegisterType: Input,
		ShortName:    AFCI_CURR_CH1.String(),
		Name:         "AFCI Current Channel 1",
		StateClass:   "measurement",
		DeviceClass:  "current",
		Multiplier:   0.001,
		Unit:         "A",
	},
	AFCI_CURR_CH2: {
		RegisterType: Input,
		ShortName:    AFCI_CURR_CH2.String(),
		Name:         "AFCI Current Channel 2",
		StateClass:   "measurement",
		DeviceClass:  "current",
		Multiplier:   0.001,
		Unit:         "A",
	},
	AFCI_CURR_CH3: {
		RegisterType: Input,
		ShortName:    AFCI_CURR_CH3.String(),
		Name:         "AFCI Current Channel 3",
		StateClass:   "measurement",
		DeviceClass:  "current",
		Multiplier:   0.001,
		Unit:         "A",
	},
	AFCI_CURR_CH4: {
		RegisterType: Input,
		ShortName:    AFCI_CURR_CH4.String(),
		Name:         "AFCI Current Channel 4",
		StateClass:   "measurement",
		DeviceClass:  "current",
		Multiplier:   0.001,
		Unit:         "A",
	},
	AFCI_ARC_CH1: {
		RegisterType:   Input,
		ShortName:      AFCI_ARC_CH1.String(),
		Name:           "Real Time ARC of CH1",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_ARC_CH2: {
		RegisterType:   Input,
		ShortName:      AFCI_ARC_CH2.String(),
		Name:           "Real Time ARC of CH2",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_ARC_CH3: {
		RegisterType:   Input,
		ShortName:      AFCI_ARC_CH3.String(),
		Name:           "Real Time ARC of CH3",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_ARC_CH4: {
		RegisterType:   Input,
		ShortName:      AFCI_ARC_CH4.String(),
		Name:           "Real Time ARC of CH4",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_MAX_ARC_CH1: {
		RegisterType:   Input,
		ShortName:      AFCI_MAX_ARC_CH1.String(),
		Name:           "Max ARC of CH1",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_MAX_ARC_CH2: {
		RegisterType:   Input,
		ShortName:      AFCI_MAX_ARC_CH2.String(),
		Name:           "Max ARC of CH2",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_MAX_ARC_CH3: {
		RegisterType:   Input,
		ShortName:      AFCI_MAX_ARC_CH3.String(),
		Name:           "Max ARC of CH3",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
	AFCI_MAX_ARC_CH4: {
		RegisterType:   Input,
		ShortName:      AFCI_MAX_ARC_CH4.String(),
		Name:           "Max ARC of CH4",
		EntityCategory: "diagnostic",
		Multiplier:     1,
	},
}
