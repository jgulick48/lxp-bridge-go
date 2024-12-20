package mqtt

type MessageCallback interface {
	HandleCommand(topic string, message []byte)
}

type BaseMessageCallback struct{}

func (cb *BaseMessageCallback) HandleCommand(topic string, message []byte) {}
