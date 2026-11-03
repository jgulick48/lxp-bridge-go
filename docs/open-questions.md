# Open questions

Registers whose meaning or scale is not confirmed. Readings are from an EG4 18kPV on 2026-11-03 while it charged from
the generator, with no grid connected.

## Generator dry contact: input 105 bit 3

Register 105 read 8 while the inverter was running the generator through its dry contact. Bits 0-2 are the documented
BMS firmware update state (values 1-3), and bit 3 is the only other bit set. pylxpweb documents the dry contact as bit 4,
which would have read 0, so it is probably off by one.

**To confirm:** `input/gen_dry_contact` should go to 0 when the inverter stops the generator.

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
