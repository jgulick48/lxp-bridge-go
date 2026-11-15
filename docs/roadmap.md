# Roadmap

Planned features, most valuable first. Nothing here is scheduled; each item becomes its own branch and PR.

## In progress

### Home Assistant availability
Publish an `online`/`offline` status per inverter and the bridge (with an MQTT last will), and point every discovered
entity at it, so Home Assistant shows entities as unavailable instead of holding stale values when the bridge or the
datalogger goes down.

## Planned

### Whole-house load sensor
When the generator or grid passes power through, the load shows in `p_load` (input 170) and `p_eps` reads about 0. On
battery the load is expected to show in `p_eps` (inputs 129 + 130) instead. Publish a derived `p_house_load =
p_eps + p_load` sensor so there is one load figure in every mode.

**Blocked on:** confirming that `p_load` drops to about 0 on battery (see [open questions](open-questions.md)).

### Charge and discharge time windows
Decode and expose holding registers 68-81 (AC charge and forced charge windows) and 84-89 (forced discharge windows)
as Home Assistant time entities or numbers. The encoding looks like one hour byte and one minute byte per register but
is not confirmed.

**Next step:** capture the EG4 website writing a known window, as was done for generator quick start.

### MQTT over TLS
Add `mqtt.tls` options (CA file, client certificate, `insecure_skip_verify`) so brokers outside the local network can
be used without sending the username and password in plain text.

### Inverter clock sync
On connect, compare the inverter clock (holding 12-14) with the host clock and correct it when it drifts past a
configurable limit. Time-window charging depends on the inverter clock.

### Prometheus `/metrics` endpoint
Serve the register values and bridge metrics on the existing health server for direct scraping (for example by
VictoriaMetrics), as an alternative to DogStatsD. This avoids statsd gauge quirks such as signed values being
treated as deltas (#22). Keep the metric names (`<namespace>_<register>` with label `deploymentID`) so dashboards keep
working.

### Read-only mode
A `mqtt.read_only` option that rejects every write command, for monitoring-only installs.

## Resolved

### Duplicate register reports
The same input readings sometimes arrive twice within a few seconds. This is the datalogger, not the bridge: when
input 3 is read, the datalogger sends the response to every connected host, so the bridge also receives readings
requested by other clients such as the EG4 cloud. No change needed; since #22 duplicate negative gauges are harmless.
