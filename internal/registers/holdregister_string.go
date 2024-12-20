// Code generated by "stringer -linecomment -type HoldRegister"; DO NOT EDIT.

package registers

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Charge_Power_Percent_CMD-0]
	_ = x[Connect_Time-1]
	_ = x[Gen_Chg_Start_Volt-2]
	_ = x[Gen_Chg_End_Volt-3]
	_ = x[Gen_Chg_Start_SOC-4]
	_ = x[Gen_Chg_End_SOC-5]
	_ = x[Gen_Rate_power-6]
	_ = x[Grid_Volt_Conn_Low-7]
	_ = x[Grid_Volt_Conn_High-8]
	_ = x[Grid_Freq_Conn_Low-9]
	_ = x[Grid_Freq_Conn_High-10]
	_ = x[Language-11]
	_ = x[Max_Gen_Chg_Bat_Curr-12]
	_ = x[Reconnect_Time-13]
	_ = x[Start_PV_Volt-14]
	_ = x[PV_Input_Model-15]
}

const _HoldRegister_name = "Charge_Power_Percent_CMDConnect_TimeGen_Chg_Start_VoltGen_Chg_End_VoltGen_Chg_Start_SOCGen_Chg_End_SOCGen_Rate_powerGrid_Volt_Conn_LowGrid_Volt_Conn_HighGrid_Freq_Conn_LowGrid_Freq_Conn_HighLanguageMax_Gen_Chg_Bat_CurrReconnect_TimeStart_PV_VoltPV_Input_Model"

var _HoldRegister_index = [...]uint16{0, 24, 36, 54, 70, 87, 102, 116, 134, 153, 171, 190, 198, 218, 232, 245, 259}

func (i HoldRegister) String() string {
	if i < 0 || i >= HoldRegister(len(_HoldRegister_index)-1) {
		return "HoldRegister(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _HoldRegister_name[_HoldRegister_index[i]:_HoldRegister_index[i+1]]
}
