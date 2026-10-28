package rpc

// LocalRPC is an in-memory RPC backend for development and tests only.
type LocalRPC struct {
	messages chan RPCMessage
}

func NewLocalRPC() *LocalRPC {
	return &LocalRPC{messages: make(chan RPCMessage, 64)}
}

func (l *LocalRPC) Send(msg RPCMessage) error {
	l.messages <- msg
	return nil
}

func (l *LocalRPC) Receive() (RPCMessage, error) {
	return <-l.messages, nil
}
