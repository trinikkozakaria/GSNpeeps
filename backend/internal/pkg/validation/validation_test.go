package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructUsesJSONFieldNameAndIndonesianMessage(t *testing.T) {
	type input struct {
		WorkEmail string `json:"work_email" validate:"required,email"`
	}

	fields := New().Struct(input{})

	assert.Equal(t, "Wajib diisi", fields["work_email"])
}
