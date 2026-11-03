package modbus

import (
	"errors"
	"math/rand"
	"os"
	"testing"

	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recorder struct {
	ParserCallback
	values map[string]int32
}

func newRecorder() *recorder { return &recorder{values: map[string]int32{}} }

func (r *recorder) ReportValue(reg registers.Register, value int32, dataLogger string) {
	r.values[reg.ShortName] = value
}
func (r *recorder) ReportPowerOpenEVSE(soc, batteryPower int32) {}

// Real captured response frames (translated data, hold reads) from an inverter.
var holdVectors = map[string][]byte{
	"hold0":   {161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 3, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 0, 0, 80, 192, 130, 9, 0, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 102, 65, 65, 66, 3, 27, 28, 1, 1, 0, 24, 12, 18, 21, 50, 57, 1, 0, 1, 0, 0, 0, 0, 0, 44, 0, 7, 0, 85, 99, 120, 5, 44, 1, 44, 1, 152, 8, 216, 9, 62, 23, 122, 23, 64, 8, 80, 10, 52, 8, 20, 5, 176, 4, 64, 11, 200, 0, 16, 0, 176, 4, 64, 11, 200, 0, 244, 225},
	"hold40":  {161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 3, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 40, 0, 80, 16, 0, 80, 10, 218, 22, 232, 23, 48, 117, 48, 117, 18, 22, 56, 24, 16, 0, 16, 0, 18, 22, 56, 24, 16, 0, 16, 0, 44, 0, 48, 9, 160, 8, 144, 9, 32, 10, 0, 0, 100, 0, 100, 0, 232, 3, 232, 3, 100, 0, 100, 0, 120, 0, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 120, 0, 100, 0, 0, 0, 0, 0, 0, 0, 0, 0, 31, 122},
	"hold120": {161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 3, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 120, 0, 80, 2, 0, 0, 0, 0, 0, 0, 0, 160, 24, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 108, 23, 64, 22, 20, 0, 52, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 28, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 144, 1, 48, 2, 215, 86},
	"hold160": {161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 3, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 160, 0, 80, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 144, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 224, 1, 0, 0, 64, 17, 100, 0, 240, 9, 80, 10, 10, 0, 20, 0, 96, 9, 44, 1, 0, 0, 44, 0, 0, 0, 50, 0, 100, 0, 100, 0, 20, 0, 144, 1, 48, 2, 20, 0, 70, 0, 250, 0, 0, 0, 8, 226},
}

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.PanicLevel)
	os.Exit(m.Run())
}

func clone(b []byte) []byte { return append([]byte(nil), b...) }

func TestCRCVerifiesRealVectors(t *testing.T) {
	// The CRC16/MODBUS covers [addr][func][serial][register][...] i.e. the
	// bytes after the 2 byte length prefix and before the trailing CRC.
	for name, v := range holdVectors {
		t.Run(name, func(t *testing.T) {
			assert.True(t, verifyCRC(v[translatedHeaderLen:]))
		})
	}
	built := BuildPacket("BA31101550", "3192670065", 0, 0x3, 0, GetBytesForUInt16(40)...)
	assert.True(t, verifyCRC(built[translatedHeaderLen:]), "BuildPacket output must verify")
}

func TestDecodeCRCModes(t *testing.T) {
	corrupt := clone(holdVectors["hold160"])
	corrupt[40] ^= 0xFF // flip a value byte; CRC is now wrong

	tests := []struct {
		name         string
		frame        []byte
		mode         CRCMode
		wantErr      error
		wantMismatch bool
		wantValues   bool
	}{
		{"valid/log", holdVectors["hold160"], CRCModeLog, nil, false, true},
		{"valid/drop", holdVectors["hold160"], CRCModeDrop, nil, false, true},
		{"corrupt/log decodes anyway", corrupt, CRCModeLog, nil, true, true},
		{"corrupt/drop discards", corrupt, CRCModeDrop, ErrCRCMismatch, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := newRecorder()
			p, err := DecodeWithMode(tt.frame, rec, tt.mode)
			if tt.wantErr != nil {
				require.True(t, errors.Is(err, tt.wantErr))
				assert.Nil(t, p)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantMismatch, p.CRCMismatch)
			}
			assert.Equal(t, tt.wantValues, len(rec.values) > 0)
		})
	}
}

func TestParseCRCMode(t *testing.T) {
	for in, want := range map[string]CRCMode{"": CRCModeLog, "log": CRCModeLog, "LOG": CRCModeLog, "drop": CRCModeDrop} {
		got, err := ParseCRCMode(in)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := ParseCRCMode("bogus")
	assert.Error(t, err)
}

func TestUint16ToInt32(t *testing.T) {
	for _, tt := range []struct {
		in   uint16
		want int32
	}{{0xFFFF, -1}, {0x8000, -32768}, {0x7FFF, 32767}, {0, 0}, {1, 1}} {
		assert.Equal(t, tt.want, uint16ToInt32(tt.in), "%#x", tt.in)
	}
}

func TestDecodeKnownVector(t *testing.T) {
	rec := newRecorder()
	_, err := Decode(clone(holdVectors["hold160"]), rec)
	require.NoError(t, err)
	assert.Contains(t, rec.values, registers.HoldRegisters[registers.Gen_Chg_Start_Volt].ShortName)
}

func TestDecodeShortInputNeverPanics(t *testing.T) {
	rec := newRecorder()
	// Every truncation of every real vector, plus a built packet.
	vectors := [][]byte{BuildPacket("BA31101550", "3192670065", 0, 0x3, 0, GetBytesForUInt16(40)...)}
	for _, v := range holdVectors {
		vectors = append(vectors, v)
	}
	for _, v := range vectors {
		for n := 0; n <= len(v); n++ {
			assert.NotPanics(t, func() { _, _ = Decode(clone(v[:n]), rec) }, "len %d", n)
			// Header claims the full length but the data is truncated/lying.
			trunc := clone(v[:n])
			assert.NotPanics(t, func() { _, _ = ParsePacket(trunc, 0xc2, rec, "x") })
		}
	}
}

func TestDecodeRandomNeverPanics(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	rec := newRecorder()
	functions := []byte{0xc1, 0xc2, 0, 0xff}
	regs := []byte{0, 40, 80, 120, 127, 160}
	for i := 0; i < 20000; i++ {
		buf := make([]byte, rng.Intn(400))
		rng.Read(buf)
		if len(buf) >= 8 && i%2 == 0 {
			// Make it look like a frame so the parsers get exercised.
			buf[0], buf[1] = 0xA1, 0x1A
			buf[4], buf[5] = byte(len(buf)-6), byte((len(buf)-6)>>8)
			buf[7] = functions[rng.Intn(len(functions))]
			if len(buf) > 33 {
				buf[21] = []byte{3, 4, 6}[rng.Intn(3)]
				buf[32] = regs[rng.Intn(len(regs))]
				buf[33] = 0
			}
		}
		assert.NotPanics(t, func() {
			_, _ = Decode(buf, rec)
			_, _ = DecodeWithMode(buf, rec, CRCModeDrop)
		})
	}
	// Function/length combinations on bare data too.
	for fn := 3; fn <= 6; fn++ {
		for _, r := range regs {
			for n := 0; n < 300; n += 7 {
				data := make([]byte, n)
				rng.Read(data)
				if n > 16 {
					data[3] = byte(fn)
					data[14], data[15] = r, 0
				}
				assert.NotPanics(t, func() { _, _ = ParsePacket(data, 0xc2, rec, "x") })
			}
		}
	}
}

func TestBuildPacketLayout(t *testing.T) {
	p := BuildPacket("BA31101550", "3192670065", 1, 0x6, 64, GetBytesForUInt16(50)...)
	require.Equal(t, []byte{0xa1, 0x1a}, p[:2])
	assert.Equal(t, len(p)-6, int(p[4])|int(p[5])<<8)
	assert.True(t, verifyCRC(p[translatedHeaderLen:]))
}
