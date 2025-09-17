package logs_querying

type ValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}
