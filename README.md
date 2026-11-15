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

Reference documentation is in the [docs](docs/) folder:

- [Polling and MQTT](docs/polling-and-mqtt.md): what is read and how often, MQTT topics, Home Assistant discovery and writes
- [Input registers](docs/input-registers.md): every published measurement
- [Holding registers](docs/holding-registers.md): every published or writable setting
- [Open questions](docs/open-questions.md): registers that still need confirming
- [Roadmap](docs/roadmap.md): planned features

More general guides are in the [Wiki](https://github.com/jgulick48/lxp-bridge-go/wiki).

## What it publishes

- **Measurements** from the input registers: PV, battery, grid, generator, EPS and load power and energy, per-leg
  values on split-phase models, BMS cell data and the generator dry contact.
- **Settings** from the holding registers, including AC and generator charging, discharge cut-offs, current limits,
  forced charge and discharge, peak shaving, smart load and quick charge. Most of them can be changed from MQTT or Home
  Assistant.
- **Home Assistant discovery** for all of the above as sensors, binary sensors, numbers and switches.

## Operations

### Example configuration

```yaml
loglevel: info
inverters:
  - enabled: true
    host: 192.168.1.50       # datalogger IP
    port: 8000
    datalog: BA12345678      # datalogger serial
    serial: 1234567890       # inverter serial
    read_timeout: 300s
    short_read: 30s          # input blocks 0, 40, 120 and 160-204
    long_read: 30s           # input block 80 (BMS data)
    hold_read: 15m           # settings
    health_check_duration: 3m
mqtt:
  enabled: true
  host: 192.168.1.10
  port: 1883
  namespace: lxp
  homeassistant:
    enabled: true
    prefix: homeassistant
statsd:
  server: ""
prometheus:
  enabled: false
```

### Configuration keys
Top level: `loglevel` (`debug|info|warn|error`, default `info`; the `-logLevel` flag overrides it), `mqtt`, `statsd`, `prometheus`, `openEVSE`, `inverters`.
Flags: `-configFile`, `-logLevel`, `-healthAddr` (default `0.0.0.0:8080`).

Per inverter (`inverters[]`):
* `enabled`: inverters that are not enabled are skipped.
* `host`, `port`, `datalog`, `serial`.
* `read_timeout`: reconnect if no data arrives for this long (default 300s).
* `short_read`: how often input blocks 0, 40, 120 and 160-204 are read (default 10s, `-1s` disables).
* `long_read`: how often input block 80 (BMS data and the generator dry contact) is read (default 30s, `-1s` disables).
  When disabled, block 80 is only decoded when another client such as the EG4 cloud
  dongle reads it.
* `hold_read`: how often the holding (settings) registers are read (default 15m, `-1s` disables). When enabled they
  are also read every time the inverter connects.
* `publish_holdings_on_connect`: read the holding registers every time the inverter connects, even with `hold_read: -1s`.
* `health_check_duration`: default 5m, see below.
* `crc_mode`:
  * `log` (default): translated-data frames are CRC16/MODBUS checked; a mismatch is counted (`result:crc_error`) and logged at debug, but the frame is still decoded.
  * `drop`: frames with a bad CRC are discarded.
* `clock_sync`: keeps the inverter clock (holding registers 12-14, local time) in step with the host, because
  time-of-use charge windows follow the inverter clock and the datalogger clock drifts.
  * `enabled`: default `false` (opt-in).
  * `timezone`: IANA zone name such as `America/Denver`. Required when `enabled` is true, there is no default
    (containers usually run in UTC and a wrong zone would shift charge windows by hours). Startup fails if it is
    missing or unknown. When set, the drift gauge below is reported even with `enabled: false`.
  * `max_drift`: allowed difference before the clock is rewritten (default 60s).

  The clock is written with one function 0x10 (write multiple registers) command followed by a read of holding block 0
  to verify. At most one write per hour per inverter. If three verifications in a row still show drift, syncing stops
  until restart and an error is logged (the datalogger may not support function 0x10).

The connection to the datalogger is re-established automatically with exponential backoff (1s up to 30s). MQTT also reconnects
on its own; on every (re)connect the command topics are resubscribed and the Home Assistant discovery configs are republished.
The MQTT broker being down never stops the inverter side. Leave `mqtt.host` empty to disable MQTT.

### Health endpoint
`GET /health` returns `204` when every enabled inverter has delivered a valid frame within its `health_check_duration`
(5m when unset; before the first frame the clock starts at process start). Otherwise it returns `503` with a short text body such as
`inverter BA12345678: no valid frame for 6m12s (limit 3m0s)`. MQTT publishing is not part of the check.

### Bridge metrics (DogStatsD)
Register gauges keep their names: `<namespace>_<register shortname lowercase>` tagged `deploymentID:<datalog>`.
Additionally, with the same tag:
* `<namespace>_bridge_connected` gauge, 1 or 0 (every 10s)
* `<namespace>_bridge_last_frame_age_s` gauge (every 10s)
* `<namespace>_bridge_frames` count, tagged `result:ok|crc_error|parse_error`
* `<namespace>_bridge_reconnects` count
* `<namespace>_bridge_commands_dropped` count (command queue of 32 was full)
* `<namespace>_bridge_clock_drift_s` gauge, inverter clock minus host clock in seconds (each holding read of
  registers 12-14; only when `clock_sync.timezone` is set)

### Prometheus metrics
Set `prometheus.enabled: true` to serve the same metrics at `GET /metrics` (Prometheus text format 0.0.4) on the
`-healthAddr` server, so Prometheus or VictoriaMetrics can scrape directly. It runs alongside DogStatsD and is off by default.
* Names are unchanged, e.g. `lxp_p_battery{deploymentID="BA12345678"}`; tags become labels.
* Counts are exposed as counters with a `_total` suffix, e.g. `lxp_bridge_frames_total{deploymentID="...",result="ok"}`.
* Negative gauge values are exposed as-is (the statsd zero reset is only sent to statsd).
* While an inverter is offline (the same rule as its `<namespace>/<datalog>/status` topic), all of its gauges are
  dropped except the `bridge_*` health gauges, so its last readings are not scraped as current. They come back with
  the next reading. Counters are never dropped.
* Optionally, `prometheus.stale_after` (e.g. `30m`) also hides any single gauge not updated for that long. It is off by
  default because settings are only re-read every `hold_read` (15m).

### Availability
The bridge publishes retained availability payloads (`online` / `offline`) so Home Assistant marks entities
unavailable instead of showing stale values:
* `<namespace>/status` is `online` while the bridge is connected to the broker. The broker publishes `offline` for
  it (MQTT last will) if the bridge disappears, and the bridge sends `offline` on a clean shutdown.
* `<namespace>/<datalog>/status` is `online` when the inverter connection is up and a valid frame has arrived since the
  bridge started, no longer ago than `health_check_duration` (5m when unset). Otherwise it is `offline`. It is
  re-evaluated every 10s and published when it changes, and again on every MQTT reconnect.

Every discovered entity lists both topics with `availability_mode: all`, so it is available only when both are `online`.

### MQTT writes
Settings are changed by publishing to `<namespace>/cmd/<datalog>/set/hold/<register>` with the value in display units
(for example `48.5` for a voltage or `8.5` for 8.5 kW). Settings that are single bits of a shared register, such as
AC charge enable, use `<namespace>/cmd/<datalog>/set/hold/<register>/<field>` with `1`/`0` (`ON`/`OFF` also work); the
bridge reads the register first and changes only those bits, so other settings in it are never overwritten.

Only the settings marked writable in [holding registers](docs/holding-registers.md) are accepted, and only inside their
range. Everything else is rejected and logged at warn; accepted writes are logged at info. Home Assistant uses these
topics automatically through discovery.

### Building
Version information is injected with `-ldflags "-X main.version=<v> -X main.commit=<sha>"` and logged at startup.
