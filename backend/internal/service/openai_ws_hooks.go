package service

type OpenAIWSIngressHooks struct {
	BeforeTurn        func(turn int) error
	BeforeTurnPayload func(turn int, payload []byte, originalModel string) error
	AfterTurn         func(turn int, result *OpenAIForwardResult, turnErr error)
}
