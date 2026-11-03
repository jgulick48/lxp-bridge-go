# Input registers

Input registers are read-only measurements, read with Modbus function 4. Every register below is published to
`<namespace>/<datalog>/input/<name>` and announced to Home Assistant. The published value is the raw value
multiplied by the scale.

**Poll** is `short` for registers read every `short_read`, and `long` for block 80 (registers 80-119), which is read
every `long_read` (default 30s). See [polling and MQTT](polling-and-mqtt.md).

**Example** is the raw value read from an EG4 18kPV (split-phase, 48 V battery bank) on 2026-11-03 while it charged from
the generator with no grid connected.

| Register | MQTT name | Home Assistant name | Scale | Unit | Poll | Example | Notes |
|---|---|---|---|---|---|---|---|
| 0 | `state` | Inverter Operating Mode | 1 |  | short | 32 | Operating mode code. 32 while charging from the generator. |
| 1 | `v_pv_1` | PV Voltage (String 1) | 0.1 | V | short | 0 |  |
| 2 | `v_pv_2` | PV Voltage (String 2) | 0.1 | V | short | 6 |  |
| 3 | `v_pv_3` | PV Voltage (String 3) | 0.1 | V | short | 19 |  |
| 4 | `v_bat` | Battery Voltage | 0.1 | V | short | 535 |  |
| 5 | `soc` | State of Charge | 1 | % | short | 25656 | Low byte of register 5. |
| 5 | `soh` | State of Health | 1 | % | short | 25656 | High byte of register 5. |
| 6 | `internal_fault` | Internal Falt Code | 1 |  | short | 13572 |  |
| 7-9 | `p_pv` | PV Power (Array) | 1 | W | short |  | Sum of registers 7, 8 and 9. |
| 7 | `p_pv_1` | PV Power (String 1) | 1 | W | short | 0 |  |
| 8 | `p_pv_2` | PV Power (String 2) | 1 | W | short | 0 |  |
| 9 | `p_pv_3` | PV Power (String 3) | 1 | W | short | 0 |  |
| 10-11 | `p_battery` | Battery Power (discharge is negative) | 1 | W | short |  | Register 10 minus register 11: positive while charging. |
| 10 | `p_charge` | Battery Charge | 1 | W | short | 11286 |  |
| 11 | `p_discharge` | Battery Discharge | 1 | W | short | 0 |  |
| 12 | `v_ac_r` | Grid Voltage (R-Phase) | 0.1 | V | short | 0 | Grid L1-L2 voltage on split-phase. |
| 13 | `v_ac_s` | Grid Voltage (S-Phase) | 0.1 | V | short | 256 | Only meaningful on three-phase models; floats on split-phase. |
| 14 | `v_ac_t` | Grid Voltage (T-Phase) | 0.1 | V | short | 122 | Only meaningful on three-phase models; floats on split-phase. |
| 15 | `f_ac` | Grid Frequency | 0.01 | Hz | short | 0 |  |
| 16 | `p_inv` | Inverter Output Power | 1 | W | short | 0 |  |
| 17 | `p_rec` | AC Charge Power | 1 | W | short | 11929 |  |
| 18 | `inv_rms` | Inverter Current RMS | 0.01 | A | short | 5004 |  |
| 19 | `pf` | Power Factor | 0.001 | % | short | 0 | A 0-1 factor, although the unit is published as %. |
| 20 | `v_eps_r` | EPS Voltage (R-Phase) | 0.1 | V | short | 2410 | EPS L1-L2 voltage on split-phase. |
| 21 | `v_eps_s` | EPS Voltage (S-Phase) | 0.1 | V | short | 68 | Only meaningful on three-phase models; floats on split-phase. |
| 22 | `v_eps_t` | EPS Voltage (T-Phase) | 0.1 | V | short | 15418 | Only meaningful on three-phase models; floats on split-phase. |
| 23 | `f_eps` | EPS Frequency | 0.01 | Hz | short | 6020 |  |
| 24 | `p_eps` | Active EPS Power | 1 | W | short | 0 |  |
| 25 | `s_eps` | Active EPS Apparent Power | 1 | VA | short | 0 |  |
| 26 | `p_to_grid` | Export Power to Grid | 1 | W | short | 0 |  |
| 27 | `p_to_user` | Import Power from Grid | 1 | W | short | 0 |  |
| 28-30 | `e_pv_day` | PV Generation (Today) | 0.1 | kWh | short |  | Sum of registers 28, 29 and 30. |
| 28 | `e_pv_day_1` | PV Generation (Today) (String 1) | 0.1 | kWh | short | 47 |  |
| 29 | `e_pv_day_2` | PV Generation (Today) (String 2) | 0.1 | kWh | short | 24 |  |
| 30 | `e_pv_day_3` | PV Generation (Today) (String 3) | 0.1 | kWh | short | 29 |  |
| 31 | `e_inv_day` | Inverter Output Energy (Today) | 0.1 | kWh | short | 0 |  |
| 32 | `e_rec_day` | AC Charge Energy (Today) | 0.1 | kWh | short | 188 |  |
| 33 | `e_chg_day` | Battery Charged Energy (Today) | 0.1 | kWh | short | 192 |  |
| 34 | `e_dischg_day` | Battery Discharged Energy (Today) | 0.1 | kWh | short | 120 |  |
| 35 | `e_eps_day` | EPS Energy (Today) | 0.1 | kWh | short | 181 |  |
| 36 | `e_to_grid_day` | Export Energy to Grid (Today) | 0.1 | kWh | short | 0 |  |
| 37 | `e_to_user_day` | Import Energy from Grid (Today) | 0.1 | kWh | short | 0 |  |
| 38 | `v_bus_1` | Bus 1 Voltage | 0.1 | V | short | 3706 |  |
| 39 | `v_bus_2` | Bus 2 Voltage | 0.1 | V | short | 3385 |  |
| 40-41 | `e_pv_1_all` | PV Generation (All Time) (String 1) | 0.1 | kWh | short | 22923, 3 | 32-bit, low word first. |
| 40-45 | `e_pv_all` | PV Generation (All Time) | 0.1 | kWh | short |  | Sum of the three 32-bit string totals. |
| 42-43 | `e_pv_2_all` | PV Generation (All Time) (String 2) | 0.1 | kWh | short | 44693, 0 | 32-bit, low word first. |
| 44-45 | `e_pv_3_all` | PV Generation (All Time) (String 3) | 0.1 | kWh | short | 27295, 0 | 32-bit, low word first. |
| 46-47 | `e_inv_all` | Inverter Output Energy (All Time) | 0.1 | kWh | short | 11, 0 | 32-bit, low word first. |
| 48-49 | `e_rec_all` | AC Charge Energy (All Time) | 0.1 | kWh | short | 50398, 0 | 32-bit, low word first. |
| 50-51 | `e_chg_all` | Battery Charged Energy (All Time) | 0.1 | kWh | short | 61014, 2 | 32-bit, low word first. |
| 52-53 | `e_dischg_all` | Battery Discharged Energy (All Time) | 0.1 | kWh | short | 42225, 2 | 32-bit, low word first. |
| 54-55 | `e_eps_all` | EPS Energy (All Time) | 0.1 | kWh | short | 26251, 4 | 32-bit, low word first. |
| 56-57 | `e_to_grid_all` | Export Energy to Grid (All Time) | 0.1 | kWh | short | 0, 0 | 32-bit, low word first. |
| 58-59 | `e_to_user_all` | Import Energy from Grid (All Time) | 0.1 | kWh | short | 0, 0 | 32-bit, low word first. |
| 60-61 | `fault_code` | Fault Code | 1 |  | short | 0, 0 | 32-bit bitfield. |
| 62-63 | `warning_code` | Warning Code | 1 |  | short | 0, 0 | 32-bit bitfield. |
| 64 | `t_inner` | Internal Ring Temperature | 1 | °C | short | 42 |  |
| 65 | `t_rad_1` | Radiator 1 Temperature | 1 | °C | short | 85 |  |
| 66 | `t_rad_2` | Radiator 2 Temperature | 1 | °C | short | 57 |  |
| 67 | `t_bat` | Battery Temperature | 1 | °C | short | 91 |  |
| 69-70 | `runtime` | Total Runtime | 1 | s | short | 37391, 1357 | 32-bit. |
| 72 | `i_pv_1` | PV Current (String 1) | 0.01 | A | short | 0 |  |
| 73 | `i_pv_2` | PV Current (String 2) | 0.01 | A | short | 0 |  |
| 74 | `i_pv_3` | PV Current (String 3) | 0.01 | A | short | 0 |  |
| 77 | `ac_input_type` | AC Input Type (0 = Grid, 1 = Generator) | 1 |  | short | 225 | Bitfield. Bit 0 is 1 when the AC input is the generator. Bits 5-7 are set on generator (225) but undocumented. |
| 81 | `max_chg_curr` | Maximum Charge Current | 0.1 | A | long | 4500 |  |
| 82 | `max_dischg_curr` | Maximum Discharge Current | 0.1 | A | long | 9000 |  |
| 83 | `chrg_volt_ref` | BMS Recommended Charging Voltage | 0.1 | V | long | 560 |  |
| 84 | `dischrg_cut_volt` | BMS Recommended Discharge Cut-Off Voltage | 0.1 | V | long | 450 |  |
| 85 | `batt_status_0_bms` | BMC Status Information 0 | 1 |  | long | 0 |  |
| 86 | `batt_status_1_bms` | BMC Status Information 1 | 1 |  | long | 0 |  |
| 87 | `batt_status_2_bms` | BMC Status Information 2 | 1 |  | long | 0 |  |
| 88 | `batt_status_3_bms` | BMC Status Information 3 | 1 |  | long | 0 |  |
| 89 | `batt_status_4_bms` | BMC Status Information 4 | 1 |  | long | 0 |  |
| 90 | `batt_status_5_bms` | BMC Status Information 5 | 1 |  | long | 192 |  |
| 91 | `batt_status_6_bms` | BMC Status Information 6 | 1 |  | long | 0 |  |
| 92 | `batt_status_7_bms` | BMC Status Information 7 | 1 |  | long | 0 |  |
| 93 | `batt_status_8_bms` | BMC Status Information 8 | 1 |  | long | 0 |  |
| 94 | `batt_status_9_bms` | BMC Status Information 9 | 1 |  | long | 0 |  |
| 95 | `batt_status_inv` | Inverter Summarized Lithium Battery Status | 1 |  | long | 3 |  |
| 96 | `batt_parrallel_num` | Number of Batteries in Parallel | 1 |  | long | 9 |  |
| 97 | `batt_capacity` | Battery Capacity in AH | 1 |  | long | 900 |  |
| 98 | `batt_current_bms` | Battery Capacity in AH | 0.1 | A | long | 2052 | Signed. |
| 99 | `fault_code_bms` | Fault Code BMS | 1 |  | long | 0 |  |
| 100 | `warning_code_bms` | Warning Code BMS | 1 |  | long | 0 |  |
| 101 | `max_cell_volt` | Maximum Cell Voltage | 0.001 | V | long | 3357 |  |
| 102 | `min_cell_volt` | Minimum Cell Voltage | 0.001 | V | long | 3343 |  |
| 103 | `max_cell_temp` | Maximum Cell Temperature | 0.1 | °C | long | 300 | Signed. |
| 104 | `min_cell_temp` | Minimum Cell Temperature | 0.1 | °C | long | 260 | Signed. |
| 105 | `bms_fw_update_state` | BMS Firmware Update State | 1 |  | long | 8 | Bits 0-2 of register 105: 1 upgrading, 2 upgrade done, 3 upgrade failed. |
| 105 | `gen_dry_contact` | Generator Dry Contact | 1 |  | long | 8 | Bit 3 of register 105. 1 while the inverter is running the generator through its dry contact. Confirmed: it went from 0 to 1 four seconds after a generator quick start. |
| 106 | `cycle_cnt_bms` | Number of Charge and Discharge Cycles | 1 |  | long | 304 |  |
| 107 | `bat_volt_smpl_inv` | Inverter Battery Voltage Sample | 0.1 | V | long | 533 | Scale was 0.001 before this version, which published 0.5 V instead of 53 V. |
| 108 | `t1` | 12K BT Temperature | 0.1 | °C | long | 398 |  |
| 109 | `t2` | 12K BT Temperature | 0.1 | °C | long | 0 |  |
| 110 | `t3` | 12K BT Temperature | 0.1 | °C | long | 0 |  |
| 111 | `t4` | 12K BT Temperature | 0.1 | °C | long | 0 |  |
| 112 | `t5` | 12K BT Temperature | 0.1 | °C | long | 0 |  |
| 114 | `consumption_power` | Consumption Power | 1 | W | long | 1222 | Equals register 170 on an 18kPV. Zero on some other models. |
| 120 | `v_bus_p` | Half Bus Voltage | 0.1 | V | short | 1854 |  |
| 121 | `v_gen` | Generator Voltage | 0.1 | V | short | 2410 |  |
| 122 | `f_gen` | Generator Frequency | 0.01 | Hz | short | 6020 |  |
| 123 | `p_gen` | Generator Power | 1 | W | short | 13118 |  |
| 124 | `e_gen_day` | Energy Imported From Generator (Today) | 0.1 | kWh | short | 207 |  |
| 125-126 | `e_gen_all` | Energy Imported From Generator (All Time) | 0.1 | kWh | short | 63356, 0 | 32-bit. |
| 127 | `v_eps_l1` | EPS Voltage L1 | 0.1 | V | short | 1202 |  |
| 128 | `v_eps_l2` | EPS Voltage L2 | 0.1 | V | short | 1206 |  |
| 129 | `p_eps_l1` | Active EPS Power L1 | 1 | W | short | 0 |  |
| 130 | `p_eps_l2` | Active EPS Power L2 | 1 | W | short | 0 |  |
| 131 | `s_eps_l1` | Active EPS Apparent Power L1 | 1 | VA | short | 0 |  |
| 132 | `s_eps_l2` | Active EPS Apparent Power L2 | 1 | VA | short | 0 |  |
| 133 | `e_eps_day_l1` | EPS Output Energy L1 (Today) | 0.1 | kWh | short | 181 |  |
| 134 | `e_eps_day_l2` | EPS Output Energy L2 (Today) | 0.1 | kWh | short | 181 |  |
| 135-136 | `e_eps_all_l1` | EPS Output Energy L1 (All Time) | 0.1 | kWh | short | 26251, 4 | 32-bit. |
| 137-138 | `e_eps_all_l2` | EPS Output Energy L2 (All Time) | 0.1 | kWh | short | 26251, 4 | 32-bit. |
| 140 | `afci_curr_ch1` | AFCI Current Channel 1 | 0.001 | A | short | 1873 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 141 | `afci_curr_ch2` | AFCI Current Channel 2 | 0.001 | A | short | 1859 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 142 | `afci_curr_ch3` | AFCI Current Channel 3 | 0.001 | A | short | 19200 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 143 | `afci_curr_ch4` | AFCI Current Channel 4 | 0.001 | A | short | 8982 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 145 | `afci_arc_ch1` | Real Time ARC of CH1 | 1 |  | short | 12584 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 146 | `afci_arc_ch2` | Real Time ARC of CH2 | 1 |  | short | 0 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 147 | `afci_arc_ch3` | Real Time ARC of CH3 | 1 |  | short | 0 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 148 | `afci_arc_ch4` | Real Time ARC of CH4 | 1 |  | short | 261 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 149 | `afci_max_arc_ch1` | Max ARC of CH1 | 1 |  | short | 0 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 150 | `afci_max_arc_ch2` | Max ARC of CH2 | 1 |  | short | 0 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 151 | `afci_max_arc_ch3` | Max ARC of CH3 | 1 |  | short | 32256 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 152 | `afci_max_arc_ch4` | Max ARC of CH4 | 1 |  | short | 32256 | Values on an 18kPV do not look like AFCI data. See [open questions](open-questions.md). |
| 170 | `p_load` | Load Power | 1 | W | short | 1247 | Signed. Load supplied through the grid or generator input. Expected to be about 0 on battery, where the load shows in `p_eps`. See [open questions](open-questions.md). |
| 171 | `e_load_day` | Load Energy (Today) | 0.1 | kWh | short | 200 |  |
| 172-173 | `e_load_all` | Load Energy (Total) | 0.1 | kWh | short | 38834, 4 | 32-bit. |
| 193 | `v_grid_l1` | Grid Voltage (L1) | 0.1 | V | short | 0 | 0 with no grid connected. |
| 194 | `v_grid_l2` | Grid Voltage (L2) | 0.1 | V | short | 0 | 0 with no grid connected. |
| 195 | `v_gen_l1` | Generator Voltage (L1) | 0.1 | V | short | 0 | Read 0 while the generator ran. See [open questions](open-questions.md). |
| 196 | `v_gen_l2` | Generator Voltage (L2) | 0.1 | V | short | 0 | Read 0 while the generator ran. See [open questions](open-questions.md). |
| 197 | `p_inv_l1` | Inverter Power (L1) | 1 | W | short | 0 |  |
| 198 | `p_inv_l2` | Inverter Power (L2) | 1 | W | short | 0 |  |
| 199 | `p_rec_l1` | Rectifier Power (L1) | 1 | W | short | 0 | Read 0 while charging from the generator. See [open questions](open-questions.md). |
| 200 | `p_rec_l2` | Rectifier Power (L2) | 1 | W | short | 0 | Read 0 while charging from the generator. See [open questions](open-questions.md). |
| 201 | `p_to_grid_l1` | Export Power to Grid (L1) | 1 | W | short | 0 | 0 with no grid connected. |
| 202 | `p_to_grid_l2` | Export Power to Grid (L2) | 1 | W | short | 0 | 0 with no grid connected. |
| 203 | `p_to_user_l1` | Import Power from Grid (L1) | 1 | W | short | 0 | 0 with no grid connected. |
| 204 | `p_to_user_l2` | Import Power from Grid (L2) | 1 | W | short | 0 | 0 with no grid connected. |

## Read but not published

These registers come back in the blocks the bridge reads but are not decoded.

| Register | Example | What it is |
|---|---|---|
| 68 | 0 | Battery control temperature (pylxpweb). |
| 72-74 | 0, 0, 0 | Decoded as PV string currents; listed above. |
| 75 | 0 | Battery current measured by the inverter (pylxpweb). Read 0 while charging at about 211 A. |
| 80 | 2 | Battery type and BMS communication type. |
| 113 | 261 | Parallel configuration: bits 0-1 role, bits 2-3 phase, bits 8-15 number of units. |
| 115-119 | 12595, 12857, 14134, 12336, 13622 | Inverter serial number as ASCII. |
| 153 | 13171 | AC couple power (pylxpweb). Tracks generator power (123) on an 18kPV. |
| 185 | 100 | Unknown. Constant 100. "On-grid power, T phase" in one project, which does not fit a split-phase unit. |
| 210 | 0 | Quick charge remaining time in seconds (pylxpweb). |
| 217-231 |  | PV strings 4-6 on models with more than 3 strings. |
| 232 | 0 | Smart load power (pylxpweb). |
| 233-268 |  | Mirror configuration values (for example 244 bootloader version, 245 flash size, 253 SOC hysteresis), not measurements. |
