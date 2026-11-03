# lxp-bridge-go

lxp-bridge-go is a tool to communicate with a LuxPower inverter (commonly used with home-battery and solar setups), written in Go and is based
off of the work from [celsworth/lxp-bridge](https://github.com/celsworth/lxp-bridge/).

It allows you to monitor and control your inverter locally without any dependence on the manufacturer's own servers in China.

## Home Assistant add-on
Click the icon below to add this repository to your Home Assistant instance or follow the procedure highlighted on the [Home Assistant website](https://home-assistant.io/hassio/installing_third_party_addons).

[![Install lxp-bridge add-on repo.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fjgulick48%2Flxp-bridge-go)

## Pre-built images
You can use pre-built images from here: https://hub.docker.com/r/jgulick48/lxp-bridge-go

## Documentation

Full documentation is now in the [Wiki](https://github.com/jgulick48/lxp-bridge-go/wiki).

## Operations

### Configuration keys
Top level: `loglevel` (`debug|info|warn|error`, default `info`; the `-logLevel` flag overrides it), `mqtt`, `statsd`, `openEVSE`, `inverters`.
Flags: `-configFile`, `-logLevel`, `-healthAddr` (default `0.0.0.0:8080`).

Per inverter (`inverters[]`): `enabled` (inverters that are not enabled are skipped), `host`, `port`, `datalog`, `serial`,
`read_timeout` (reconnect if no data for this long, default 300s), `short_read` / `long_read` (poll intervals, `-1s` disables),
`publish_holdings_on_connect` (read the holding registers every time the inverter connects),
`health_check_duration` (default 5m, see below) and
`crc_mode`:
* `log` (default): translated-data frames are CRC16/MODBUS checked; a mismatch is counted (`result:crc_error`) and logged at debug, but the frame is still decoded.
* `drop`: frames with a bad CRC are discarded.

The connection to the datalogger is re-established automatically with exponential backoff (1s up to 30s). MQTT also reconnects
on its own; on every (re)connect the command topics are resubscribed and the Home Assistant discovery configs are republished.
The MQTT broker being down never stops the inverter side. Leave `mqtt.host` empty to disable MQTT.

### Health endpoint
`GET /health` returns `204` when every enabled inverter has delivered a valid frame within its `health_check_duration`
(5m when unset; before the first frame the clock starts at process start). Otherwise it returns `503` with a short text body such as
`inverter BA31101550: no valid frame for 6m12s (limit 3m0s)`. MQTT publishing is not part of the check.

### Bridge metrics (DogStatsD)
Register gauges keep their names: `<namespace>_<register shortname lowercase>` tagged `deploymentID:<datalog>`.
Additionally, with the same tag:
* `<namespace>_bridge_connected` gauge, 1 or 0 (every 10s)
* `<namespace>_bridge_last_frame_age_s` gauge (every 10s)
* `<namespace>_bridge_frames` count, tagged `result:ok|crc_error|parse_error`
* `<namespace>_bridge_reconnects` count
* `<namespace>_bridge_commands_dropped` count (command queue of 32 was full)

### MQTT writes
Commands to `<namespace>/cmd/<datalog>/set/hold/<register>` are only accepted for single-register hold registers that are
exposed to Home Assistant as controllable entities (`Writeable` Number/Switch), with an integer payload inside the register's
`Min..Max` (raw register units; 0..65535 when no range is defined). Currently: 194 (Gen charge start volt, 384..520),
195 (Gen charge end volt, 480..590), 196 (Gen charge start SOC, 0..90), 197 (Gen charge end SOC, 20..100).
Everything else is rejected and logged at warn; accepted writes are logged at info.

### Building
Version information is injected with `-ldflags "-X main.version=<v> -X main.commit=<sha>"` and logged at startup.
