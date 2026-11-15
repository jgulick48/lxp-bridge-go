# Roadmap

Planned features, most valuable first. Nothing here is scheduled; each item becomes its own branch and PR.

## Planned

### Whole-house load sensor
The original plan was a derived `p_house_load = p_eps + p_load`, assuming `p_load` (input 170) drops to about 0 on
battery. It does not. On battery on 2026-11-15 (06:48-06:50 UTC, no generator, no PV) `p_load` read 993-1038 W while
`p_eps` (inputs 129 + 130) read 972-1024 W: both report the same load, so the sum would count it twice.

`p_load` appears to be the whole-house load in every mode on its own. If that holds while the generator runs (when
`p_eps` reads about 0), no derived sensor is needed: `p_load` just needs its description and Home Assistant name
updated to "House Load".

**Next step:** confirm `p_load` against `p_gen - p_rec` during a generator run (see [open questions](open-questions.md)).

### Inverter clock sync (in progress)
On connect, compare the inverter clock (holding 12-14) with the host clock and correct it when it drifts past a
configurable limit. Time-window charging depends on the inverter clock.

### Prometheus `/metrics` endpoint (in progress)
Serve the register values and bridge metrics on the existing health server for direct scraping (for example by
VictoriaMetrics), as an alternative to DogStatsD. This avoids statsd gauge quirks such as signed values being
treated as deltas (#22). Keep the metric names (`<namespace>_<register>` with label `deploymentID`) so dashboards keep
working.

## Resolved

### Read-only mode
Top-level `read_only: true` (not `mqtt.read_only`, because clock sync also writes) blocks MQTT writes, bit-field writes
and clock sync, and announces Home Assistant entities as sensors only. See the README.

### MQTT over TLS
`mqtt.tls` (`enabled`, `ca_file`, `cert_file`/`key_file` for mutual TLS, `server_name`, `insecure_skip_verify`) secures the
broker connection (`ssl://`, port defaults to 8883). Bad TLS settings fail startup. See the README.

### Charge and discharge time windows
Holding registers 68-73 (AC charge windows 1-3), 76-81 (forced charge) and 84-89 (forced discharge) are exposed as
Home Assistant text entities holding `HH:MM`, readable and settable (`set/hold/68` with payload `07:30`). Each register
holds one time with the hour in the low byte and the minute in the high byte, the opposite of the clock registers; the
encoding comes from ant0nkr/luxpower-ha-integration and celsworth/lxp-bridge, which agree, and was confirmed on the
EG4 18kPV on 2026-11-15: AC charge window 3 set to 01:30-02:00 in the EG4 web UI read back as 72 = 0x1e01 and
73 = 0x0002.

### Home Assistant availability
Done in #25: `<namespace>/status` (with an MQTT last will) and `<namespace>/<datalog>/status`, referenced by every
discovered entity with `availability_mode: all`.

### Duplicate register reports
The same input readings sometimes arrive twice within a few seconds. This is the datalogger, not the bridge: when
input 3 is read, the datalogger sends the response to every connected host, so the bridge also receives readings
requested by other clients such as the EG4 cloud. No change needed; since #22 duplicate negative gauges are harmless.
