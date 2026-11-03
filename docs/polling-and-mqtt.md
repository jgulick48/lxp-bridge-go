# Polling and MQTT

## Connection

The bridge connects to the inverter's WiFi datalogger over TCP (usually port 8000) and sends LuxPower "translated data"
frames: a Modbus request wrapped with the datalogger and inverter serial numbers. The datalogger forwards every reply
to **all** connected clients, so replies to requests made by the EG4 cloud dongle or another tool are decoded too.

## What is read and how often

| Function | Registers | Size | When | Contents |
|---|---|---|---|---|
| 4 (input) | 0-39 | 40 | every `short_read` (default 10s) | PV, battery, grid, EPS power and today's energy |
| 4 (input) | 40-79 | 40 | every `short_read` | Lifetime energy, faults, temperatures, PV currents |
| 4 (input) | 120-159 | 40 | every `short_read` | Generator, EPS per leg, AFCI |
| 4 (input) | 160-204 | 45 | every `short_read` | Load power and energy, per-leg grid, generator and inverter values |
| 4 (input) | 80-119 | 40 | every `long_read` (default 30s) | BMS data, cell voltages and temperatures, generator dry contact |
| 3 (holding) | 0-239 | 6 × 40 | on connect, then every `hold_read` (default 15m) | Settings |
| 3 (holding) | one register | 1 | before each bit-field write | The fresh value for a read-modify-write |

With `long_read: -1s` the bridge does not request input block 80 at all and only decodes it when another client (such
as the EG4 cloud dongle) does.

Requests are queued (32 deep) and sent one per second. Polls are skipped while the inverter is disconnected.

## MQTT topics

`<namespace>` is `mqtt.namespace` (default `lxp`) and `<datalog>` is the datalogger serial.

| Topic | Direction | Payload |
|---|---|---|
| `<namespace>/<datalog>/input/<name>` | published, retained | Measurement in display units, see [input registers](input-registers.md) |
| `<namespace>/<datalog>/hold/<name>` | published, retained | Setting in display units, see [holding registers](holding-registers.md) |
| `<namespace>/cmd/<datalog>/set/hold/<register>` | subscribed | New value for a whole-register setting |
| `<namespace>/cmd/<datalog>/set/hold/<register>/<field>` | subscribed | `1`/`0` (or `ON`/`OFF`) for a switch, or a number for a multi-bit field |
| `<namespace>/LWT` | published every 20s, retained | `Online` |

`<name>` is the lowercase MQTT name from the register tables, for example `lxp/BA12345678/input/p_load`.

## Home Assistant discovery

With `mqtt.homeassistant.enabled: true`, every register is announced on each MQTT connect with a retained config at
`<prefix>/<component>/<namespace>_<datalog>/<name>/config`:

| Component | Used for |
|---|---|
| `sensor` | Input registers and read-only settings |
| `binary_sensor` | Generator dry contact |
| `number` | Writable whole-register settings, with min, max and step in display units |
| `switch` | Writable single-bit settings, with payload and state `1`/`0` |

All entities belong to one device named `<namespace>_<datalog>`, with unique IDs `<namespace>_<datalog>_<name>`
(a few older energy sensors keep the IDs they had before, such as `..._pv_generation_today`).
Settings only get a value after the first holding read, which happens when the bridge connects.

## Writes

Every command is checked before anything is sent: the datalogger must be configured, the register must be writable,
and the value must be a whole number of steps inside its range. Rejected commands are logged at warn with the reason;
accepted ones at info.

**Whole-register settings** are written straight away with function 6.

**Bit-field settings** (switches and the AC charge type) share a register with other settings, so writing a whole value
from an old copy could undo a change made in the EG4 app. Instead the bridge:

1. Records the requested bits and reads that one register.
2. When the reply arrives, sets only the requested bits on the fresh value and writes it, or does nothing if the bits
   already have that value.
3. Drops the change if no reply arrives within 30 seconds.

Several bit changes to the same register before the read reply are combined into one write.

The inverter echoes each write, and the echo is decoded like a read, so Home Assistant shows the new value as soon as
the inverter accepts it.
