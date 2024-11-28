package capigraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextDimension(t *testing.T) {
	var w, h float64
	w, h = getTextDimensions("", FontTypefaceArial, FontWeightNormal, 20, 0.0)
	assert.Equal(t, 0.0, w)
	assert.Equal(t, 0.0, h)
	w, h = getTextDimensions("A", FontTypefaceArial, FontWeightNormal, 20, 0.0)
	assert.Equal(t, 13.4, w)
	assert.Equal(t, 20.0, h)
	w, h = getTextDimensions("\nA", FontTypefaceArial, FontWeightNormal, 20, 0.0)
	assert.Equal(t, 13.4, w)
	assert.Equal(t, 40.0, h)
	w, h = getTextDimensions("\n", FontTypefaceArial, FontWeightNormal, 20, 0.0)
	assert.Equal(t, 0.0, w)
	assert.Equal(t, 0.0, h)
}

func TestRunWidth(t *testing.T) {
	var w float64
	w = getTextRunWidth("-A-------", 1, 2, FontTypefaceArial, FontWeightNormal, 20)
	assert.Equal(t, 13.4, w)
	w = getTextRunWidth("-AB------", 1, 3, FontTypefaceArial, FontWeightNormal, 20)
	assert.Equal(t, 26.8, w)
	w = getTextRunWidth("-\r-------", 1, 2, FontTypefaceArial, FontWeightNormal, 20)
	assert.Equal(t, 5.6, w)
}
