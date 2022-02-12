package utils

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRandInt(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	min := 1
	max := 20

	for i := 0; i < 50; i++ {
		r := RandInt(min, max)

		assert.GreaterOrEqual(t, r, min)
		assert.LessOrEqual(t, r, max)
	}
}
