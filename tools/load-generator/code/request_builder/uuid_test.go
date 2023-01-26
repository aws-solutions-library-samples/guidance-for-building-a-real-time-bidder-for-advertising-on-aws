package requestbuilder

import (
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestUuid(t *testing.T) {
	actual := string(newUUIDBuilder().uuid())
	_, err := uuid.FromString(actual)
	assert.NoError(t, err)
}
