package errors

import "net/http"

type ErrorType string

const (
	ErrInvalidRequest      ErrorType = "INVALID_REQUEST"
	ErrUnauthorized        ErrorType = "UNAUTHORIZED"
	ErrForbidden           ErrorType = "FORBIDDEN"
	ErrNotFound            ErrorType = "NOT_FOUND"
	ErrNotAllowed          ErrorType = "NOT_ALLOWED"
	ErrInternalServerError ErrorType = "INTERNAL_SERVER_ERROR"
	ErrServiceUnavailable  ErrorType = "SERVICE_UNAVAILABLE"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorMessage struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

type ErrorResponse struct {
	HTTPStatusCode int               `json:"status"`
	ErrorMessage   ErrorMessage      `json:"error"`
	Validation     []ValidationError `json:"validation,omitempty"`
}

type Builder struct {
	httpStatusCode int
	errorCode      string
	message        string
	validation     []ValidationError
}

func (b *Builder) WithStatusCode(code int) *Builder {
	b.httpStatusCode = code
	return b
}

func (b *Builder) WithMessage(message string) *Builder {
	b.message = message
	return b
}

func (b *Builder) WithErrorCode(errorCode string) *Builder {
	b.errorCode = errorCode
	return b
}

func (b *Builder) WithValidation(validation []ValidationError) *Builder {
	b.validation = validation
	return b
}

func (b *Builder) Build() *ErrorResponse {
	return &ErrorResponse{
		HTTPStatusCode: b.httpStatusCode,
		ErrorMessage: ErrorMessage{
			ErrorCode:    b.errorCode,
			ErrorMessage: b.message,
		},
		Validation: b.validation,
	}
}

func Custom() *Builder {
	return &Builder{}
}

func InvalidRequest() *Builder {
	return &Builder{
		httpStatusCode: http.StatusBadRequest,
		errorCode:      string(ErrInvalidRequest),
		message:        "Invalid Request",
	}
}

func Unauthorized() *Builder {
	return &Builder{
		httpStatusCode: http.StatusUnauthorized,
		errorCode:      string(ErrUnauthorized),
		message:        "Unauthorized",
	}
}

func Forbidden() *Builder {
	return &Builder{
		httpStatusCode: http.StatusForbidden,
		errorCode:      string(ErrForbidden),
		message:        "Forbidden",
	}
}

func NotFound() *Builder {
	return &Builder{
		httpStatusCode: http.StatusNotFound,
		errorCode:      string(ErrNotFound),
		message:        "Not Found",
	}
}

func NotAllowed() *Builder {
	return &Builder{
		httpStatusCode: http.StatusMethodNotAllowed,
		errorCode:      string(ErrNotAllowed),
		message:        "Not Allowed",
	}
}

func InternalServerError() *Builder {
	return &Builder{
		httpStatusCode: http.StatusInternalServerError,
		errorCode:      string(ErrInternalServerError),
		message:        "Internal Server Error",
	}
}

func ServiceUnavailable() *Builder {
	return &Builder{
		httpStatusCode: http.StatusServiceUnavailable,
		errorCode:      string(ErrServiceUnavailable),
		message:        "Service Not Available",
	}
}

func GenerateByStatusCode(code int) *Builder {
	switch code {
	case http.StatusBadRequest:
		return InvalidRequest()
	case http.StatusUnauthorized:
		return Unauthorized()
	case http.StatusForbidden:
		return Forbidden()
	case http.StatusNotFound:
		return NotFound()
	case http.StatusMethodNotAllowed:
		return NotAllowed()
	case http.StatusServiceUnavailable:
		return ServiceUnavailable()
	default:
		return InternalServerError().WithStatusCode(code).WithMessage(http.StatusText(code))
	}
}
