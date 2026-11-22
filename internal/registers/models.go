package registers

import (
	"fmt"
	"reflect"
	"strings"
)

// Model selects how an inverter's registers are decoded and announced. Some
// registers hold different data on different models.
type Model string

const (
	// Model18KPV is the EG4 18kPV layout every inverter used before models
	// were introduced.
	Model18KPV Model = "18kpv"
	// Model6000XP is the EG4 6000XP (off-grid, split-phase, 2 PV inputs).
	Model6000XP Model = "6000xp"
	// ModelAuto picks the layout from the inverter's device type code. Until
	// the code has been read only registers that mean the same on every
	// model are handled.
	ModelAuto Model = "auto"
)

// ParseModel parses the model configuration value, ignoring case and
// surrounding space. An empty value is ModelAuto.
func ParseModel(s string) (Model, error) {
	switch m := Model(strings.ToLower(strings.TrimSpace(s))); m {
	case "", ModelAuto:
		return ModelAuto, nil
	case Model18KPV, Model6000XP:
		return m, nil
	}
	return ModelAuto, fmt.Errorf("unknown model %q (expected %q, %q or %q)", s, ModelAuto, Model18KPV, Model6000XP)
}

// DetectModel maps the device type code (holding register 19) to a model.
// Codes 38 and 54 are the EG4 6000XP / 12000XP off-grid family (codes from
// pylxpweb); anything else, such as 44 (18kPV), uses the 18kPV layout.
func DetectModel(deviceType uint16) Model {
	if deviceType == 38 || deviceType == 54 {
		return Model6000XP
	}
	return Model18KPV
}

type regKey struct {
	t    RegisterType
	name string
}

func inputKeys(rs ...InputRegister) []regKey {
	k := make([]regKey, len(rs))
	for i, r := range rs {
		k[i] = regKey{Input, r.String()}
	}
	return k
}

func holdKeys(rs ...HoldRegister) []regKey {
	k := make([]regKey, len(rs))
	for i, r := range rs {
		k[i] = regKey{Hold, r.String()}
	}
	return k
}

// omit6000XP lists the registers the 6000XP does not have, by group.
var omit6000XP = func() map[regKey]bool {
	groups := [][]regKey{
		// No third PV input; input 3 mirrors the bus voltage.
		inputKeys(V_PV_3, P_PV_3, E_PV_DAY_3, E_PV_3_ALL, I_PV_3),
		// Inputs 13, 14, 21 and 22 are not voltages on this model.
		inputKeys(V_AC_S, V_AC_T, V_EPS_S, V_EPS_T),
		// Input 123 is a seconds counter and 124-126 are status words.
		inputKeys(P_GEN, E_GEN_DAY, E_GEN_ALL),
		// Input 105 has a different bit layout.
		inputKeys(BMS_FW_UPDATE_STATE, GEN_COOL_DOWN, GEN_DRY_CONTACT),
		// Inputs 114 and 171-173 are not filled in.
		inputKeys(CONSUMPTION_POWER, E_LOAD_DAY, E_LOAD_ALL),
		// Per-leg values from inputs 193-204 are unused (194-199 per the
		// 6kXP protocol document).
		inputKeys(V_GRID_L1, V_GRID_L2, V_GEN_L1, V_GEN_L2, P_INV_L1, P_INV_L2,
			P_REC_L1, P_REC_L2, P_TO_GRID_L1, P_TO_GRID_L2, P_TO_USER_L1, P_TO_USER_L2),
		// Hold 110 bit 7 is the buzzer enable here, not a generator quick
		// start; hold 237 (generator cool down time) is not used.
		holdKeys(Gen_Quick_Start, Gen_Cool_Down_Time),
	}
	m := map[regKey]bool{}
	for _, g := range groups {
		for _, k := range g {
			m[k] = true
		}
	}
	return m
}()

// Resolve returns the register as this model decodes and announces it, and
// false when the model does not have it (the value must be dropped: not
// published, no metric, no discovery, no writes).
func (m Model) Resolve(r Register) (Register, bool) {
	switch m {
	case Model18KPV:
		return r, true
	case ModelAuto:
		// Not detected yet: only registers identical on every model.
		a, aok := Model18KPV.Resolve(r)
		b, bok := Model6000XP.Resolve(r)
		if aok && bok && reflect.DeepEqual(a, b) {
			return r, true
		}
		return r, false
	}
	// The time-of-day windows (holds 68-73, 76-81, 84-89) are not used.
	if r.RegisterType == Hold && r.TimeOfDay {
		return r, false
	}
	k := regKey{r.RegisterType, r.ShortName}
	if omit6000XP[k] {
		return r, false
	}
	if k == (regKey{Hold, Gen_Rate_power.String()}) {
		// Hold 177 is in watts here (100 W units on the 18kPV) and read only.
		r.Multiplier = 1
		r.Unit = "W"
		r.DeviceClass = "power"
		r.Writeable = false
	}
	return r, true
}
