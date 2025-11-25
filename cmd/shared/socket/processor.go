package socket

// RequestProcessor defines the interface for processing socket commands
// Each service implements this interface with their specific command handling logic
type RequestProcessor interface {
	ProcessRequest(data []byte) CommandResponse
}
