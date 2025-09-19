package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	err := New(ErrNotFound, "not found")
	require.Equal(t, "not found", err.Error())
	require.Equal(t, ErrNotFound, err.Code)

	err = Newf(ErrBadRequest, "bad request %d", 400)
	require.Equal(t, "bad request 400", err.Error())
	require.Equal(t, ErrBadRequest, err.Code)
}
