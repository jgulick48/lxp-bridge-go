package modbus

import (
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestServer(t *testing.T) {
	messageSendFunction := func(register registers.Register, value float64) {
		logrus.Infof("%s: %v %s", register.Name, value*register.Multiplier, register.Unit)
	}

	Decode([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 4, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 0, 0, 80, 192, 0, 79, 0, 165, 4, 5, 0, 17, 2, 84, 100, 0, 59, 0, 0, 13, 0, 0, 0, 0, 0, 149, 6, 0, 0, 5, 1, 251, 0, 0, 0, 0, 0, 0, 0, 132, 2, 0, 0, 110, 9, 73, 0, 58, 51, 112, 23, 48, 6, 215, 6, 0, 0, 0, 0, 211, 0, 43, 0, 0, 0, 0, 0, 0, 0, 152, 0, 116, 0, 189, 0, 0, 0, 0, 0, 157, 15, 84, 12, 104, 31}, messageSendFunction)
	Decode([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 4, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 40, 0, 80, 194, 52, 1, 0, 12, 23, 0, 0, 14, 0, 0, 0, 5, 0, 0, 0, 223, 30, 0, 0, 84, 191, 0, 0, 207, 172, 0, 0, 58, 46, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 41, 0, 50, 0, 49, 0, 246, 0, 0, 0, 106, 65, 195, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 100, 169}, messageSendFunction)
	//Decode([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 4, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 120, 0, 80, 205, 7, 0, 0, 0, 0, 0, 0, 0, 0, 241, 46, 0, 0, 176, 4, 175, 4, 92, 2, 128, 3, 20, 3, 205, 3, 189, 0, 189, 0, 58, 46, 1, 0, 58, 46, 1, 0, 0, 0, 228, 7, 216, 7, 0, 0, 45, 0, 0, 0, 136, 34, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 134, 65})
	//Decode([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 4, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 0, 0, 80, 192, 0, 56, 0, 160, 4, 5, 0, 17, 2, 84, 100, 0, 59, 0, 0, 13, 0, 0, 0, 0, 0, 140, 6, 0, 0, 5, 1, 251, 0, 0, 0, 0, 0, 0, 0, 102, 2, 0, 0, 103, 9, 73, 0, 58, 51, 112, 23, 12, 6, 224, 6, 0, 0, 0, 0, 211, 0, 43, 0, 0, 0, 0, 0, 0, 0, 152, 0, 116, 0, 189, 0, 0, 0, 0, 0, 158, 15, 84, 12, 126, 126})
	//Decode([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 4, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 40, 0, 80, 194, 52, 1, 0, 12, 23, 0, 0, 14, 0, 0, 0, 5, 0, 0, 0, 223, 30, 0, 0, 84, 191, 0, 0, 207, 172, 0, 0, 58, 46, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 42, 0, 50, 0, 49, 0, 246, 0, 0, 0, 116, 65, 195, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 26, 156})
	//Decode([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 4, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 120, 0, 80, 205, 7, 0, 0, 0, 0, 0, 0, 0, 0, 241, 46, 0, 0, 174, 4, 175, 4, 118, 2, 148, 3, 233, 2, 230, 3, 189, 0, 189, 0, 58, 46, 1, 0, 58, 46, 1, 0, 0, 0, 229, 7, 216, 7, 0, 0, 45, 0, 0, 0, 136, 34, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 200, 17})
	//Decode([]byte{161, 26, 5, 0, 13, 0, 1, 193, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 0})
	//Decode([]byte{161, 26, 5, 0, 29, 1, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 15, 1, 1, 4, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 0, 0, 254, 192, 0, 44, 0, 157, 4, 5, 0, 17, 2, 84, 100, 0, 59, 0, 0, 13, 0, 0, 0, 0, 0, 146, 6, 0, 0, 5, 1, 251, 0, 0, 0, 0, 0, 0, 0, 137, 2, 0, 0, 101, 9, 73, 0, 58, 51, 112, 23, 31, 6, 238, 6, 0, 0, 0, 0, 211, 0, 43, 0, 0, 0, 0, 0, 0, 0, 152, 0, 116, 0, 189, 0, 0, 0, 0, 0, 158, 15, 84, 12, 194, 52, 1, 0, 12, 23, 0, 0, 14, 0, 0, 0, 5, 0, 0, 0, 223, 30, 0, 0, 84, 191, 0, 0, 207, 172, 0, 0, 58, 46, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 41, 0, 50, 0, 49, 0, 246, 0, 0, 0, 122, 65, 195, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 2, 0, 184, 11, 112, 23, 48, 2, 194, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 192, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 6, 0, 88, 2, 180, 254, 0, 0, 0, 0, 240, 12, 232, 12, 230, 0, 180, 0, 0, 0, 93, 0, 10})
	//Decode([]byte{161, 26, 5, 0, 111, 0, 1, 194, 66, 65, 51, 49, 49, 48, 49, 53, 53, 48, 97, 0, 1, 4, 51, 49, 57, 50, 54, 55, 48, 48, 54, 53, 0, 0, 80, 64, 0, 0, 0, 4, 0, 5, 0, 16, 2, 73, 100, 0, 57, 0, 0, 0, 0, 0, 0, 0, 0, 185, 6, 0, 0, 5, 1, 251, 0, 0, 0, 0, 0, 0, 0, 183, 2, 0, 0, 110, 9, 73, 0, 58, 51, 112, 23, 52, 6, 66, 7, 0, 0, 0, 0, 211, 0, 43, 0, 0, 0, 0, 0, 0, 0, 152, 0, 150, 0, 220, 0, 0, 0, 0, 0, 156, 15, 78, 12, 113, 174})
	command := BuildPacket("BA31101550", "3192670065", 0, 0x3, 0, 40)
	logrus.Infof("%v", command)
}
