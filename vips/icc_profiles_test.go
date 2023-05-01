package vips

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ICCProfileInitialisation(t *testing.T) {
	initializeICCProfiles()

	assertIccProfile(t, sRGBV2MicroICCProfile, SRGBV2MicroICCProfilePath)
	assertIccProfile(t, sGrayV2MicroICCProfile, SGrayV2MicroICCProfilePath)
	assertIccProfile(t, sRGBIEC6196621ICCProfile, SRGBIEC6196621ICCProfilePath)
	assertIccProfile(t, genericGrayGamma22ICCProfile, GenericGrayGamma22ICCProfilePath)
}

func assertIccProfile(t *testing.T, expectedProfile []byte, path string) {
	loadedProfile, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expectedProfile, loadedProfile)
}
