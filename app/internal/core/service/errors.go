package service

// ServiceError preserves the original error code and message from the socket service
type ServiceError struct {
	Code    int
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

func NewServiceError(code int, message string) *ServiceError {
	return &ServiceError{Code: code, Message: message}
}
