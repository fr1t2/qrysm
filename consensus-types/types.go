package consensus_types

import (
	"errors"
	"fmt"

	errors2 "github.com/pkg/errors"
	"github.com/theQRL/qrysm/v4/runtime/version"
)

var (
	// ErrNilObjectWrapped is returned in a constructor when the underlying object is nil.
	ErrNilObjectWrapped = errors.New("attempted to wrap nil object")
	// ErrUnsupportedField is returned when a getter/setter access is not supported.
	ErrUnsupportedField = errors.New("unsupported getter")
)

func ErrNotSupported(funcName string, ver int) error {
	return errors2.Wrap(ErrUnsupportedField, fmt.Sprintf("%s is not supported for %s", funcName, version.String(ver)))
}
