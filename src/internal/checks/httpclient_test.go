package checks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateClientForInterfaceInvalid(t *testing.T) {
	client, err := createClientForInterface("nonexistent0", "4", 5*time.Second)
	assert.Error(t, err)
	assert.Nil(t, client)
}
