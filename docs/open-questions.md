# Open questions

Registers whose meaning or scale is not confirmed. Readings are from an EG4 18kPV on 2026-11-03 while it charged from
the generator, with no grid connected.

## Generator quick start: holding 110 bit 7 (confirmed)

Captured from the EG4 website on 2026-11-03. Each time, the cloud read holding 110 first and then wrote it back with
only bit 7 changed, which is what the `gen_quick_start` switch does.

| Time (UTC) | Event |
|---|---|
| 03:19:38 | Gen Quick Start: holding 110 written 1056 → 1184 (bit 7 set) |
| 03:19:42 | Input 105: 0 → 8, dry contact closed (bit 3) |
| 03:19:47 | Generator voltage (input 121) rising, 242 V ten seconds later |
| 03:21:56 | Input 77: 224 → 225, inverter connected to the generator (bit 0) |
| 03:22:11 | Stop: holding 110 written 1184 → 1056 (bit 7 cleared) |
| 03:22:15 | Input 105: 8 → 12, cooling down (bit 2) with the contact still closed |
| 03:24:19 | Input 105: 12 → 0, contact open |

This confirms that input 105 bit 3 is the generator dry contact (not bit 4 as pylxpweb documents) and that bit 2 is the
cool-down. The generator voltage was still 242 V after the contact opened, so the generator has its own shutdown delay.

**Still to confirm:** whether the inverter clears holding 110 bit 7 by itself after an uninterrupted 20 minute run, and
how the cool-down length relates to holding 237 (`gen_cool_down_time`, 30, which is 3.0 minutes in 0.1 minute units; about 2 minutes was
observed, but block 80 is only read every `long_read`).

## Load power on battery: input 170

Register 170 (`p_load`) tracked `p_gen - p_rec` while on generator. On battery it is expected to drop to about 0, with
the load showing in `p_eps` (registers 129 + 130), so whole-house load would be `p_eps + p_load` in every mode.

**To confirm:** compare `p_load` and `p_eps` with the generator off.

## Per-leg generator voltage and rectifier power: inputs 195-196 and 199-200

These read 0 while the generator ran at 241 V and the inverter charged from it at about 11.9 kW. They may only count
power from the grid input, or not be filled in by this firmware.

**To confirm:** check them with the grid connected and with the generator running. Remove them if they stay at 0.

## AFCI registers: inputs 140-152

The bridge publishes these as AFCI current and arc counts, but the values read (1874, 1860, 19200, 8982, 1537, 12584 and
32256) do not look like AFCI data, and pylxpweb leaves these registers out for the 18kPV.

## AC charge type: holding 120 bits 1-3

Exposed with the labels 0 time, 1 SOC/voltage, 2 both. The test unit reads 1 and is set up to start AC charging by SOC,
which fits, but the labels are not confirmed. Check against the EG4 app before relying on them.

## Units of holding 66, 82 and 177

Treated as 100 W units. pylxpweb confirms this for 66 and 82 against the EG4 panel. For 177 the test unit reads 170
(17.0 kW). Check against the generator rating set in the EG4 app.

## Unidentified input registers

185 (constant 100), and the bits 5-7 of input 77 (set while on generator, value 225). See
[input registers](input-registers.md#read-but-not-published).
