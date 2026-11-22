package registers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseModel(t *testing.T) {
	for in, want := range map[string]Model{"": ModelAuto, "auto": ModelAuto, "AUTO": ModelAuto, " auto ": ModelAuto,
		"18KPV": Model18KPV, "18kpv": Model18KPV, "6000XP": Model6000XP, " 6000xp ": Model6000XP, "6000xp": Model6000XP} {
		m, err := ParseModel(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, m, in)
	}
	_, err := ParseModel("foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "18kpv")
	assert.Contains(t, err.Error(), "6000xp")
	assert.Contains(t, err.Error(), "auto")
}

func TestDetectModel(t *testing.T) {
	assert.Equal(t, Model6000XP, DetectModel(38))
	assert.Equal(t, Model6000XP, DetectModel(54))
	assert.Equal(t, Model18KPV, DetectModel(44))
	assert.Equal(t, Model18KPV, DetectModel(0))
}

func TestResolveAuto(t *testing.T) {
	for _, id := range []InputRegister{P_GEN, V_PV_3} {
		_, ok := ModelAuto.Resolve(InputRegisters[id])
		assert.False(t, ok, id.String())
	}
	for _, id := range []HoldRegister{Gen_Quick_Start, Gen_Rate_power} {
		_, ok := ModelAuto.Resolve(HoldRegisters[id])
		assert.False(t, ok, id.String())
	}
	tod := 0
	for _, r := range HoldRegisters {
		if r.TimeOfDay {
			tod++
			_, ok := ModelAuto.Resolve(r)
			assert.False(t, ok, r.ShortName)
		}
	}
	assert.NotZero(t, tod)
	for _, id := range []InputRegister{SOC, V_GEN} {
		r, ok := ModelAuto.Resolve(InputRegisters[id])
		assert.True(t, ok, id.String())
		assert.Equal(t, InputRegisters[id], r)
	}
}

func TestResolve6000XPOmissions(t *testing.T) {
	var omitted []Register
	for _, r := range InputRegisters {
		if omit6000XP[regKey{Input, r.ShortName}] {
			omitted = append(omitted, r)
		}
	}
	for _, id := range []InputRegister{V_PV_3, P_PV_3, E_PV_DAY_3, E_PV_3_ALL, I_PV_3,
		V_AC_S, V_AC_T, V_EPS_S, V_EPS_T, P_GEN, E_GEN_DAY, E_GEN_ALL,
		BMS_FW_UPDATE_STATE, GEN_COOL_DOWN, GEN_DRY_CONTACT,
		CONSUMPTION_POWER, E_LOAD_DAY, E_LOAD_ALL,
		V_GRID_L1, V_GRID_L2, V_GEN_L1, V_GEN_L2, P_INV_L1, P_INV_L2,
		P_REC_L1, P_REC_L2, P_TO_GRID_L1, P_TO_GRID_L2, P_TO_USER_L1, P_TO_USER_L2} {
		r := InputRegisters[id]
		require.Equal(t, id.String(), r.ShortName)
		_, ok := Model6000XP.Resolve(r)
		assert.False(t, ok, id.String())
		_, ok = Model18KPV.Resolve(r)
		assert.True(t, ok, id.String())
	}
	assert.Len(t, omitted, 30)

	tod := 0
	for _, r := range HoldRegisters {
		if r.TimeOfDay {
			tod++
			_, ok := Model6000XP.Resolve(r)
			assert.False(t, ok, r.ShortName)
			_, ok = Model18KPV.Resolve(r)
			assert.True(t, ok, r.ShortName)
		}
	}
	assert.Equal(t, 18, tod)
	for _, id := range []HoldRegister{Gen_Quick_Start, Gen_Cool_Down_Time} {
		r := HoldRegisters[id]
		_, ok := Model6000XP.Resolve(r)
		assert.False(t, ok, id.String())
		_, ok = Model18KPV.Resolve(r)
		assert.True(t, ok, id.String())
	}
}

func TestResolve6000XPKeeps(t *testing.T) {
	for _, id := range []InputRegister{V_GEN, F_GEN, SOC, V_PV_1, V_PV_2, P_LOAD, PARALLEL_ROLE, PARALLEL_PHASE, PARALLEL_NUM, AC_COUPLE_POWER} {
		r, ok := Model6000XP.Resolve(InputRegisters[id])
		assert.True(t, ok, id.String())
		assert.Equal(t, InputRegisters[id], r)
	}
}

func TestResolveGenRatePower(t *testing.T) {
	base := HoldRegisters[Gen_Rate_power]
	r, ok := Model6000XP.Resolve(base)
	require.True(t, ok)
	assert.Equal(t, float32(1), r.Multiplier)
	assert.Equal(t, "W", r.Unit)
	assert.False(t, r.Writeable)
	assert.Equal(t, Sensor, r.DiscoveryType())

	r, ok = Model18KPV.Resolve(base)
	require.True(t, ok)
	assert.Equal(t, base, r)
	assert.Equal(t, float32(0.1), r.Multiplier)
	assert.True(t, r.Writeable)
}
