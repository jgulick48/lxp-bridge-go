# lxp-bridge-go documentation

- [Polling and MQTT](polling-and-mqtt.md): which registers are read and how often, the MQTT topics, Home Assistant
  discovery, and how writes work.
- [Input registers](input-registers.md): every measurement the bridge publishes, with register addresses, scale and
  unit.
- [Holding registers](holding-registers.md): every setting the bridge publishes or can change, with ranges and command
  topics, plus the settings that are deliberately left read-only.
- [Open questions](open-questions.md): registers whose meaning is not confirmed yet, and how to check them.
- [Legacy](legacy/): notes and register lists from the original lxp-bridge project.

Register addresses are decimal. The register maps were checked against an EG4 18kPV (split-phase), the EG4 18kPV Modbus protocol PDF, and the community projects
[joyfulhouse/pylxpweb](https://github.com/joyfulhouse/pylxpweb) and
[ant0nkr/luxpower-ha-integration](https://github.com/ant0nkr/luxpower-ha-integration). Other LuxPower and EG4 models
use the same protocol but may leave some registers at 0.
