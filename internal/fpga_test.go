package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFPGA_OpenNClose(t *testing.T){
	fpga, err := NewFPGADevice(0)
	assert.NoError(t, err)
	assert.NotNil(t, fpga)
	
	err = fpga.Close()
	assert.NoError(t, err)
}