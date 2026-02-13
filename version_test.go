package kuniumi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrameworkVersionString(t *testing.T) {
	result := frameworkVersionString()
	// frameworkVersionString always returns a string.
	// During go test, ReadBuildInfo is available,
	// so the result should contain "based on kuniumi".
	assert.Contains(t, result, "based on kuniumi")
	// Should contain either a version number or "dev"
	assert.Regexp(t, `based on kuniumi (v[\d.]+|dev)`, result)
}
