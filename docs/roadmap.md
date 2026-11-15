# Roadmap

Planned features, most valuable first. Nothing here is scheduled; each item becomes its own branch and PR.

## Planned

Nothing is planned right now. Possible follow-ups: retire the legacy `<namespace>/LWT` heartbeat now that
`<namespace>/status` exists, expose the AC-first time windows (holding 152-157), and confirm on hardware that the
inverter accepts the clock sync write (function 0x10).

## Resolved

### Whole-house load
No derived sensor is needed: `p_load` (input 170) is the whole-house load in every mode and is now named "House Load"
in Home Assistant (the entity ID is unchanged). The original plan, `p_eps + p_load`, would have counted the load twice.
Measured on an EG4 18kPV on 2026-11-15:

| Mode | `p_load` | `p_eps` | Other |
|---|---|---|---|
| Battery, no PV (07:00-13:00 UTC, hourly averages) | 963-1027 W | 966-1029 W | |
| Battery and PV (14:00-15:30 UTC, hourly averages) | 1016-1038 W | 1017-1048 W | |
| Generator connected (16:49-17:11 UTC, 112 samples) | 1156 W average | 0 W | `p_gen - p_rec` 1178 W average |

### Inverter clock sync
Done in #27: per-inverter `clock_sync` compares holding 12-14 with the host clock and corrects drift past `max_drift`.
The drift is reported as `<namespace>_bridge_clock_drift_s`. See the README.

### Prometheus `/metrics` endpoint
Done in #28: `prometheus.enabled` serves the same metrics at `GET /metrics` on the health server. See the README.

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
