package godaikin

import "fmt"

type DaikinError struct {
	Message string
	Err     error
}

func (e *DaikinError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("daikin error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("daikin error: %s", e.Message)
}

func (e *DaikinError) Unwrap() error {
	return e.Err
}

func NewDaikinError(message string, err error) *DaikinError {
	return &DaikinError{
		Message: message,
		Err:     err,
	}
}

type ConnectionError struct {
	*DaikinError
}

func NewConnectionError(message string, err error) *ConnectionError {
	return &ConnectionError{
		DaikinError: NewDaikinError(message, err),
	}
}

type AuthenticationError struct {
	*DaikinError
}

func NewAuthenticationError(message string, err error) *AuthenticationError {
	return &AuthenticationError{
		DaikinError: NewDaikinError(message, err),
	}
}

type ParseError struct {
	*DaikinError
}

func NewParseError(message string, err error) *ParseError {
	return &ParseError{
		DaikinError: NewDaikinError(message, err),
	}
}
