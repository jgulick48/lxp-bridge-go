// Code generated by "stringer -linecomment -type InputRegister"; DO NOT EDIT.

package registers

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[AC_INPUT_TYPE-0]
	_ = x[AFCI_ARC_CH1-1]
	_ = x[AFCI_ARC_CH2-2]
	_ = x[AFCI_ARC_CH3-3]
	_ = x[AFCI_ARC_CH4-4]
	_ = x[AFCI_CURR_CH1-5]
	_ = x[AFCI_CURR_CH2-6]
	_ = x[AFCI_CURR_CH3-7]
	_ = x[AFCI_CURR_CH4-8]
	_ = x[AFCI_MAX_ARC_CH1-9]
	_ = x[AFCI_MAX_ARC_CH2-10]
	_ = x[AFCI_MAX_ARC_CH3-11]
	_ = x[AFCI_MAX_ARC_CH4-12]
	_ = x[BATT_CAPACITY-13]
	_ = x[BATT_CURRENT_BMS-14]
	_ = x[BATT_PARRALlEL_NUM-15]
	_ = x[BATT_STATUS_0_BMS-16]
	_ = x[BATT_STATUS_1_BMS-17]
	_ = x[BATT_STATUS_2_BMS-18]
	_ = x[BATT_STATUS_3_BMS-19]
	_ = x[BATT_STATUS_4_BMS-20]
	_ = x[BATT_STATUS_5_BMS-21]
	_ = x[BATT_STATUS_6_BMS-22]
	_ = x[BATT_STATUS_7_BMS-23]
	_ = x[BATT_STATUS_8_BMS-24]
	_ = x[BATT_STATUS_9_BMS-25]
	_ = x[BATT_STATUS_INV-26]
	_ = x[BAT_VOLT_SMPL_INV-27]
	_ = x[BMS_FW_UPDATE_STATE-28]
	_ = x[CHRG_VOLT_REF-29]
	_ = x[CYCLE_CNT_BMS-30]
	_ = x[DISCHRG_CUT_VOLT-31]
	_ = x[E_CHG_ALL-32]
	_ = x[E_CHG_DAY-33]
	_ = x[E_DISCHG_ALL-34]
	_ = x[E_DISCHG_DAY-35]
	_ = x[E_EPS_ALL-36]
	_ = x[E_EPS_ALL_L1-37]
	_ = x[E_EPS_ALL_L2-38]
	_ = x[E_EPS_DAY-39]
	_ = x[E_EPS_DAY_L1-40]
	_ = x[E_EPS_DAY_L2-41]
	_ = x[E_GEN_ALL-42]
	_ = x[E_GEN_DAY-43]
	_ = x[E_INV_ALL-44]
	_ = x[E_INV_DAY-45]
	_ = x[E_PV_1_ALL-46]
	_ = x[E_PV_2_ALL-47]
	_ = x[E_PV_3_ALL-48]
	_ = x[E_PV_ALL-49]
	_ = x[E_PV_DAY-50]
	_ = x[E_PV_DAY_1-51]
	_ = x[E_PV_DAY_2-52]
	_ = x[E_PV_DAY_3-53]
	_ = x[E_REC_ALL-54]
	_ = x[E_REC_DAY-55]
	_ = x[E_TO_GRID_ALL-56]
	_ = x[E_TO_GRID_DAY-57]
	_ = x[E_TO_USER_ALL-58]
	_ = x[E_TO_USER_DAY-59]
	_ = x[F_AC-60]
	_ = x[F_EPS-61]
	_ = x[F_GEN-62]
	_ = x[Fault_Code-63]
	_ = x[Fault_Code_BMS-64]
	_ = x[INV_RMS-65]
	_ = x[Internal_Fault-66]
	_ = x[MAX_CELL_TEMP_BMS-67]
	_ = x[MAX_CELL_VOLT_BMS-68]
	_ = x[MAX_CHG_CURR-69]
	_ = x[MAX_DISCHG_CURR-70]
	_ = x[MIN_CELL_TEMP_BMS-71]
	_ = x[MIN_CELL_VOLT_BMS-72]
	_ = x[PF-73]
	_ = x[P_Battery-74]
	_ = x[P_Charge-75]
	_ = x[P_Discharge-76]
	_ = x[P_EPS-77]
	_ = x[P_EPS_L1-78]
	_ = x[P_EPS_L2-79]
	_ = x[P_GEN-80]
	_ = x[P_INV-81]
	_ = x[P_PV-82]
	_ = x[P_PV_1-83]
	_ = x[P_PV_2-84]
	_ = x[P_PV_3-85]
	_ = x[P_REC-86]
	_ = x[P_TO_GRID-87]
	_ = x[P_TO_USER-88]
	_ = x[Runtime-89]
	_ = x[SOC-90]
	_ = x[SOH-91]
	_ = x[S_EPS-92]
	_ = x[S_EPS_L1-93]
	_ = x[S_EPS_L2-94]
	_ = x[State-95]
	_ = x[T1-96]
	_ = x[T2-97]
	_ = x[T3-98]
	_ = x[T4-99]
	_ = x[T5-100]
	_ = x[T_BAT-101]
	_ = x[T_INNER-102]
	_ = x[T_RAD_1-103]
	_ = x[T_RAD_2-104]
	_ = x[V_AC_R-105]
	_ = x[V_AC_S-106]
	_ = x[V_AC_T-107]
	_ = x[V_BAT-108]
	_ = x[V_BUS_1-109]
	_ = x[V_BUS_2-110]
	_ = x[V_BUS_P-111]
	_ = x[V_EPS_L1-112]
	_ = x[V_EPS_L2-113]
	_ = x[V_EPS_R-114]
	_ = x[V_EPS_S-115]
	_ = x[V_EPS_T-116]
	_ = x[V_GEN-117]
	_ = x[V_PV_1-118]
	_ = x[V_PV_2-119]
	_ = x[V_PV_3-120]
	_ = x[Warning_Code-121]
	_ = x[Warning_Code_BMS-122]
}

const _InputRegister_name = "AC_INPUT_TYPEAFCI_ARC_CH1AFCI_ARC_CH2AFCI_ARC_CH3AFCI_ARC_CH4AFCI_CURR_CH1AFCI_CURR_CH2AFCI_CURR_CH3AFCI_CURR_CH4AFCI_MAX_ARC_CH1AFCI_MAX_ARC_CH2AFCI_MAX_ARC_CH3AFCI_MAX_ARC_CH4BATT_CAPACITYBATT_CURRENT_BMSBATT_PARRALlEL_NUMBATT_STATUS_0_BMSBATT_STATUS_1_BMSBATT_STATUS_2_BMSBATT_STATUS_3_BMSBATT_STATUS_4_BMSBATT_STATUS_5_BMSBATT_STATUS_6_BMSBATT_STATUS_7_BMSBATT_STATUS_8_BMSBATT_STATUS_9_BMSBATT_STATUS_INVBAT_VOLT_SMPL_INVBMS_FW_UPDATE_STATECHRG_VOLT_REFCYCLE_CNT_BMSDISCHRG_CUT_VOLTE_CHG_ALLE_CHG_DAYE_DISCHG_ALLE_DISCHG_DAYE_EPS_ALLE_EPS_ALL_L1E_EPS_ALL_L2E_EPS_DAYE_EPS_DAY_L1E_EPS_DAY_L2E_GEN_ALLE_GEN_DAYE_INV_ALLE_INV_DAYE_PV_1_ALLE_PV_2_ALLE_PV_3_ALLE_PV_ALLE_PV_DAYE_PV_DAY_1E_PV_DAY_2E_PV_DAY_3E_REC_ALLE_REC_DAYE_TO_GRID_ALLE_TO_GRID_DAYE_TO_USER_ALLE_TO_USER_DAYF_ACF_EPSF_GENFault_CodeFault_Code_BMSINV_RMSInternal_FaultMAX_CELL_TEMP_BMSMAX_CELL_VOLT_BMSMAX_CHG_CURRMAX_DISCHG_CURRMIN_CELL_TEMP_BMSMIN_CELL_VOLT_BMSPFP_BatteryP_ChargeP_DischargeP_EPSP_EPS_L1P_EPS_L2P_GENP_INVP_PVP_PV_1P_PV_2P_PV_3P_RECP_TO_GRIDP_TO_USERRuntimeSOCSOHS_EPSS_EPS_L1S_EPS_L2StateT1T2T3T4T5T_BATT_INNERT_RAD_1T_RAD_2V_AC_RV_AC_SV_AC_TV_BATV_BUS_1V_BUS_2V_BUS_PV_EPS_L1V_EPS_L2V_EPS_RV_EPS_SV_EPS_TV_GENV_PV_1V_PV_2V_PV_3Warning_CodeWarning_Code_BMS"

var _InputRegister_index = [...]uint16{0, 13, 25, 37, 49, 61, 74, 87, 100, 113, 129, 145, 161, 177, 190, 206, 224, 241, 258, 275, 292, 309, 326, 343, 360, 377, 394, 409, 426, 445, 458, 471, 487, 496, 505, 517, 529, 538, 550, 562, 571, 583, 595, 604, 613, 622, 631, 641, 651, 661, 669, 677, 687, 697, 707, 716, 725, 738, 751, 764, 777, 781, 786, 791, 801, 815, 822, 836, 853, 870, 882, 897, 914, 931, 933, 942, 950, 961, 966, 974, 982, 987, 992, 996, 1002, 1008, 1014, 1019, 1028, 1037, 1044, 1047, 1050, 1055, 1063, 1071, 1076, 1078, 1080, 1082, 1084, 1086, 1091, 1098, 1105, 1112, 1118, 1124, 1130, 1135, 1142, 1149, 1156, 1164, 1172, 1179, 1186, 1193, 1198, 1204, 1210, 1216, 1228, 1244}

func (i InputRegister) String() string {
	if i < 0 || i >= InputRegister(len(_InputRegister_index)-1) {
		return "InputRegister(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _InputRegister_name[_InputRegister_index[i]:_InputRegister_index[i+1]]
}
