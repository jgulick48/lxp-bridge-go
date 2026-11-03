# Open questions

Registers whose meaning or scale is not confirmed. Readings are from an EG4 18kPV on 2026-11-03 while it charged from
the generator, with no grid connected.

## Generator quick start: holding 110 bit 7

Pressing Gen Quick Start on the EG4 website made the cloud read holding 110 (1056) and write 1184, which only sets
bit 7. Four seconds later the dry contact bit (input 105 bit 3) went from 0 to 1, and the generator voltage rose to
242 V. The `gen_quick_start` switch sends the same write. This also confirms that input 105 bit 3 is the generator
dry contact, not bit 4 as pylxpweb documents.

**To confirm:** whether the inverter clears bit 7 by itself after the 20 minute run, and whether turning the switch
off stops the generator early.

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
