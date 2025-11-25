package types

type ValidationResult struct {
	Code    int
	Message string
}

const (
	StatusOK                 = 200
	StatusBadRequest         = 400
	StatusNotFound           = 404
	StatusConflict           = 409
	StatusInternalServerError = 500
	StatusServiceUnavailable = 503
)
