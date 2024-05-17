package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccount(t *testing.T) {
	acc, err := NewAccount("a", "b", "hunter")

	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%+v\n", acc)

	assert.NotNil(t, acc)
}
