// Code generated by "stringer -linecomment -type HomeAssistantType"; DO NOT EDIT.

package registers

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Sensor-0]
	_ = x[Switch-1]
	_ = x[Number-2]
	_ = x[Text-3]
}

const _HomeAssistantType_name = "SensorSwitchNumberText"

var _HomeAssistantType_index = [...]uint8{0, 6, 12, 18, 22}

func (i HomeAssistantType) String() string {
	if i < 0 || i >= HomeAssistantType(len(_HomeAssistantType_index)-1) {
		return "HomeAssistantType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _HomeAssistantType_name[_HomeAssistantType_index[i]:_HomeAssistantType_index[i+1]]
}
