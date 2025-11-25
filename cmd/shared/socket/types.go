package socket

type CommandRequest struct {
	Cmd  string                 `json:"cmd"`
	Args map[string]interface{} `json:"args"`
}

type CommandResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}
