package httpx

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// Bind decodes the JSON request body into v, returning a 400 on malformed input.
func Bind(c echo.Context, v interface{}) error {
	if err := c.Bind(v); err != nil {
		return BadRequest("malformed request body")
	}
	return nil
}

// Validator accumulates field errors for a 422 response.
type Validator struct{ fields map[string]string }

func NewValidator() *Validator { return &Validator{fields: map[string]string{}} }

func (v *Validator) Require(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.fields[field] = "is required"
	}
	return v
}

func (v *Validator) Check(cond bool, field, msg string) *Validator {
	if !cond {
		v.fields[field] = msg
	}
	return v
}

func (v *Validator) Err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return Validation(v.fields)
}
