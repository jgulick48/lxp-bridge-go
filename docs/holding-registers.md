# Holding registers (settings)

Holding registers are the inverter settings, read with Modbus function 3 and written with function 6. Every register
below is published to `<namespace>/<datalog>/hold/<name>` and announced to Home Assistant: writable values as
`number` entities, single bits as `switch` entities, and read-only values as `sensor` entities.

Command topics are relative to `<namespace>/cmd/<datalog>/`. Payloads are in display units (the raw value times the
scale), so `8.5` on register 66 sets 8.5 kW. **Range** is in display units. Writes outside the range are rejected.

Settings that share a register with other settings (**Bits** filled in) are changed with a read-modify-write: the
bridge reads the register, changes only those bits, and writes it back. See [polling and MQTT](polling-and-mqtt.md).

**Example** is the value read from an EG4 18kPV on 2026-11-03, raw for whole registers and the field value for bits.

| Register | Bits | MQTT name | Home Assistant name | Type | Scale | Unit | Range | Command topic | Example | Notes |
|---|---|---|---|---|---|---|---|---|---|---|
| 16 |  | `language` | Language (0=English 1=German) | sensor | 1 |  |  | read-only | 1 | 0 English, 1 German. |
| 20 |  | `pv_input_model` | PV Input Mode | sensor | 1 |  |  | read-only | 7 |  |
| 21 | 0 | `eps_enable` | EPS (Backup) Enable | switch | 1 |  | 0 / 1 | `set/hold/21/eps_enable` | 1 |  |
| 21 | 7 | `ac_charge_enable` | AC Charge Enable | switch | 1 |  | 0 / 1 | `set/hold/21/ac_charge_enable` | 0 |  |
| 21 | 10 | `forced_dischg_enable` | Forced Discharge Enable | switch | 1 |  | 0 / 1 | `set/hold/21/forced_dischg_enable` | 0 |  |
| 21 | 11 | `forced_chg_enable` | Forced Charge Enable | switch | 1 |  | 0 / 1 | `set/hold/21/forced_chg_enable` | 0 |  |
| 21 | 15 | `feed_in_grid_enable` | Feed-In Grid Enable | switch | 1 |  | 0 / 1 | `set/hold/21/feed_in_grid_enable` | 0 |  |
| 22 |  | `start_pv_volt` | PV Working Starting Voltage | sensor | 0.1 | V |  | read-only | 1400 |  |
| 23 |  | `connect_time` | Grid Connection Waiting Time | sensor | 1 | s |  | read-only | 300 |  |
| 24 |  | `reconnect_time` | Reconnection Waiting Time | sensor | 1 | s |  | read-only | 300 |  |
| 64 |  | `charge_power_percent_cmd` | Charging power percentage setting | number | 1 | % | 0 to 100 | `set/hold/64` | 100 |  |
| 65 |  | `discharge_power_percent_cmd` | Discharging power percentage setting | number | 1 | % | 0 to 100 | `set/hold/65` | 100 |  |
| 66 |  | `ac_charge_power_cmd` | AC Charge Power | number | 0.1 | kW | 0 to 12 | `set/hold/66` | 120 | Raw value in 100 W units: 120 is 12.0 kW. |
| 67 |  | `ac_charge_soc_limit` | AC Charge SOC Limit | number | 1 | % | 0 to 101 | `set/hold/67` | 100 | 101 means never stop, used for cell balancing. |
| 82 |  | `forced_dischg_power_cmd` | Forced Discharge Power | number | 0.1 | kW | 0 to 12 | `set/hold/82` | 120 | Raw value in 100 W units. |
| 83 |  | `forced_dischg_soc_limit` | Forced Discharge SOC Limit | number | 1 | % | 0 to 100 | `set/hold/83` | 20 |  |
| 101 |  | `charge_rate` | Battery Charge Current Limit | number | 1 | A | 0 to 250 | `set/hold/101` | 250 |  |
| 102 |  | `discharge_rate` | Battery Discharge Current Limit | number | 1 | A | 0 to 250 | `set/hold/102` | 250 |  |
| 105 |  | `discharge_cut_off_soc` | On-Grid Discharge Cut-Off SOC | number | 1 | % | 10 to 100 | `set/hold/105` | 10 | Lower limit of 10 follows pylxpweb. |
| 110 | 7 | `gen_quick_start` | Generator Quick Start | switch | 1 |  | 0 / 1 | `set/hold/110/gen_quick_start` | 0 | Same write as the EG4 website's generator quick start, which runs the generator for about 20 minutes. Some projects call this bit the buzzer enable. |
| 120 | 1-3 | `ac_charge_type` | AC Charge Type (0=Time 1=SOC/Volt 2=Time+SOC/Volt) | number | 1 |  | 0 to 2 | `set/hold/120/ac_charge_type` | 1 | Labels 0 time, 1 SOC/voltage, 2 both are not confirmed. |
| 125 |  | `eps_discharge_cut_off_soc` | Off-Grid Discharge Cut-Off SOC | number | 1 | % | 0 to 100 | `set/hold/125` | 0 |  |
| 158 |  | `ac_chg_start_volt` | AC Charging Start Battery Voltage | number | 0.1 | V | 38.4 to 52 | `set/hold/158` | 400 |  |
| 159 |  | `ac_chg_end_volt` | AC Charging End Battery Voltage | number | 0.1 | V | 48 to 59 | `set/hold/159` | 560 |  |
| 160 |  | `ac_chg_start_soc` | AC Charging Start Battery SOC | number | 1 | % | 0 to 90 | `set/hold/160` | 50 |  |
| 161 |  | `ac_chg_end_soc` | AC Charging End Battery SOC | number | 1 | % | 0 to 100 | `set/hold/161` | 0 | Reads 0 on the test unit. |
| 168 |  | `ac_chg_bat_curr` | AC Charge Battery Current Limit | number | 1 | A | 0 to 250 | `set/hold/168` | 0 |  |
| 169 |  | `on_grid_eod_volt` | On-Grid End of Discharge Voltage | number | 0.1 | V | 40 to 56 | `set/hold/169` | 400 |  |
| 177 |  | `gen_rate_power` | Generator Rated Power | number | 0.1 | kW | 0 to 25.5 | `set/hold/177` | 170 | Raw value in 100 W units: 170 is 17.0 kW. Called max generator input power in some documents. |
| 179 | 7 | `grid_peak_shaving_enable` | Grid Peak Shaving Enable | switch | 1 |  | 0 / 1 | `set/hold/179/grid_peak_shaving_enable` | 0 |  |
| 179 | 8 | `gen_peak_shaving_enable` | Generator Peak Shaving Enable | switch | 1 |  | 0 / 1 | `set/hold/179/gen_peak_shaving_enable` | 1 |  |
| 179 | 13 | `smart_load_enable` | Smart Load Enable | switch | 1 |  | 0 / 1 | `set/hold/179/smart_load_enable` | 0 |  |
| 194 |  | `gen_chg_start_volt` | Generator Charging Start Battery Voltage | number | 0.1 | V | 38.4 to 52 | `set/hold/194` | 400 |  |
| 195 |  | `gen_chg_end_volt` | Generator Charging End Battery Voltage | number | 0.1 | V | 48 to 59 | `set/hold/195` | 560 |  |
| 196 |  | `gen_chg_start_soc` | Generator Charging Start Battery SOC | number | 1 | % | 0 to 90 | `set/hold/196` | 21 |  |
| 197 |  | `gen_chg_end_soc` | Generator Charging End Battery SOC | number | 1 | % | 20 to 100 | `set/hold/197` | 90 |  |
| 198 |  | `max_gen_chg_bat_curr` | Generator Charge Battery Current Limit | number | 1 | A | 0 to 250 | `set/hold/198` | 250 |  |
| 202 |  | `stop_discharge_volt` | Stop Discharge Voltage | number | 0.1 | V | 40 to 56 | `set/hold/202` | 400 |  |
| 233 | 0 | `quick_charge_enable` | Quick Charge Enable | switch | 1 |  | 0 / 1 | `set/hold/233/quick_charge_enable` | 0 |  |
| 234 |  | `quick_charge_minute` | Quick Charge Time | number | 1 | min | 0 to 1440 | `set/hold/234` | 0 | Also counts down the remaining minutes while a quick charge runs. |
| 237 |  | `gen_cool_down_time` | Generator Cool Down Time | number | 0.1 | min | 0.1 to 25.5 | `set/hold/237` | 30 | How long the dry contact stays closed after a generator stop. 30 is 3.0 minutes. |

## Known settings that are not exposed

These are documented by EG4 or the community projects but deliberately not writable from MQTT, because a wrong
value can trip the inverter, make it non-compliant with the grid code, or damage the batteries. Change them from the
inverter screen or the EG4 app.

| Register | Setting | Example |
|---|---|---|
| 21 bit 9 | Power on (turning this off shuts the inverter down) |  |
| 21 other bits | Anti-islanding, neutral detect, GFCI, DCI, isolation detect, DRMS, LVRT |  |
| 25-28 | Grid connection voltage and frequency limits | 2200, 2520, 5950, 6010 |
| 59-62 | Reactive power mode and settings | 0, 100, 100, 1000 |
| 68-81 | AC charge and forced charge time windows (encoding not yet confirmed) |  |
| 84-89 | Forced discharge time windows |  |
| 90 | Output voltage | 240 |
| 91 | Output frequency | 60 |
| 99 | Battery charge voltage reference | 560 |
| 100 | Battery discharge cut-off voltage | 400 |
| 110 (other bits) | System function bits (micro-grid, green mode and others) | 1056 |
| 144 | Float charge voltage | 540 |
| 147-151 | Battery capacity, nominal voltage and equalization |  |
| 176 | Max grid input power | 65535 |
| 206-208, 218-219, 232 | Grid peak shaving power, SOC and voltage |  |
| 227-228 | System charge SOC and voltage limits | 101, 595 |
