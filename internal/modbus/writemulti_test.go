package modbus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWriteMulti(t *testing.T) {
	// Nov 14 2026 23:53:44: 0x0b1a, 0x170e, 0x2c35.
	p := BuildWriteMulti("BA31101550", "3192670065", 12, []uint16{0x0b1a, 0x170e, 0x2c35})
	want := []byte{
		0xa1, 0x1a, 0x01, 0x00, 0x27, 0x00, // header, protocol 1, length 39
		0x01, 0xc2, // tcp function: translated data
		'B', 'A', '3', '1', '1', '0', '1', '5', '5', '0', // datalog
		0x19, 0x00, // data length 25
		0x01, 0x10, // action, function 0x10
		'3', '1', '9', '2', '6', '7', '0', '0', '6', '5', // serial
		0x0c, 0x00, // start register 12
		0x03, 0x00, // register count 3
		0x06,                               // byte count
		0x1a, 0x0b, 0x0e, 0x17, 0x35, 0x2c, // values
		0x3d, 0x4d, // CRC16/MODBUS
	}
	assert.Equal(t, want, p)
	assert.True(t, verifyCRC(p[translatedHeaderLen:]))
	assert.Equal(t, len(p)-6, int(p[4])|int(p[5])<<8)
	assert.Equal(t, byte(0x10), p[translatedHeaderLen+3])
}

func TestDecodeWriteMultiAckIsHarmless(t *testing.T) {
	rec := newRecorder()
	// Ack: [start register][register count].
	ack := BuildPacket("BA31101550", "3192670065", 1, 0x10, 12, GetBytesForUInt16(3)...)
	pkt, err := Decode(ack, rec)
	require.NoError(t, err)
	assert.False(t, pkt.CRCMismatch)
	assert.Empty(t, rec.values)
	// Even the request frame itself, or a bare ack, must not error or panic.
	_, err = Decode(BuildWriteMulti("BA31101550", "3192670065", 12, []uint16{1, 2, 3}), rec)
	require.NoError(t, err)
	_, err = Decode(BuildPacket("BA31101550", "3192670065", 1, 0x10, 12), rec)
	require.NoError(t, err)
}
