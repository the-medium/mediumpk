package internal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
)

const (
	// SignRequestSize is buffer size of sign request
	SignRequestSize = 128
	// VerifyRequestSize is buffer size of verify request
	VerifyRequestSize = 192
	// ResponseSize is buffer size of response
	ResponseSize = 96
	// MetricSetSize is buffer size of MetricSet
	MetricSetSize = 28
	rwUnitBytes   = 4
)

// FPGADevice is a structue to store device file descriptors
type FPGADevice struct {
	h2c  *os.File
	c2h  *os.File
	ctrl *os.File
	user *os.File
}

// NewFPGADevice returns FPGADevice instance
func NewFPGADevice(index int) (*FPGADevice, error) {
	prefix := "/dev/mdlx" + strconv.Itoa(index)

	h2c, err := os.OpenFile(prefix+"_h2c_0", os.O_WRONLY|os.O_EXCL, os.ModeDevice)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	c2h, err := os.OpenFile(prefix+"_c2h_0", os.O_RDONLY|os.O_EXCL, os.ModeDevice)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	ctrl, err := os.OpenFile(prefix+"_control", os.O_RDONLY|os.O_EXCL, os.ModeDevice)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	user, err := os.OpenFile(prefix+"_user", os.O_RDWR|os.O_EXCL, os.ModeDevice)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	dev := FPGADevice{
		h2c,
		c2h,
		ctrl,
		user,
	}

	err = dev.Reset()
	if err != nil {
		return nil, err
	}

	return &dev, nil
}

// Close closes device descriptors
func (d *FPGADevice) Close() (err error) {
	err = d.h2c.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = d.c2h.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = d.ctrl.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = d.user.Close()
	if err != nil {
		log.Fatal(err)
	}

	return
}

// Request send request into FPGA
func (d *FPGADevice) Request(buffer []byte) (err error) {
	writeSize, err := d.h2c.Write(buffer)
	if err != nil {
		return
	}

	if writeSize != len(buffer) {
		err = errors.New("write size not match.." + strconv.Itoa(writeSize))
		return
	}

	return nil
}

// Poll brings result from FPGA
func (d *FPGADevice) Poll() ([]byte, error) {
	buffer := make([]byte, ResponseSize)

	readSize, err := d.c2h.Read(buffer)
	if err != nil {
		return nil, err
	}

	if readSize != ResponseSize {
		err = errors.New("read size not match.." + strconv.Itoa(readSize))
		return nil, err
	}

	return buffer, nil
}

// CheckAvailable checks if h2c/c2h channel is available
func (d *FPGADevice) CheckAvailable() error {
	buffer := make([]byte, rwUnitBytes)

	detail := []string{"H2C", "C2H"}
	value := [][]byte{{0x06, 0x80, 0xc0, 0x1f}, {0x06, 0x80, 0xc1, 0x1f}}
	pos := []int64{0x0000, 0x1000}
	for i, v := range pos {
		readSize, err := d.ctrl.ReadAt(buffer, v)
		if err != nil {
			return err
		}
		if readSize != rwUnitBytes {
			return fmt.Errorf("[control] readSize %d not match with %d at 0x%x... %s", readSize, rwUnitBytes, v, detail[i])
		}
		if bytes.Compare(buffer, value[i]) != 0 {
			return fmt.Errorf("[control] %s Channel Unavailable", detail[i])
		}
	}

	return nil
}

// GetMetrics returns device metric information
func (d *FPGADevice) GetMetrics() ([]byte, error) {
	buffer := make([]byte, MetricSetSize)

	idx := 0
	detail := []string{"Temperature", "VCCINT", "VCCAUX", "VCCBRAM", "Total", "Success", "Error"}
	pos := []int64{0x2400, 0x2404, 0x2408, 0x2418, 0x18010, 0x18014, 0x18018}
	for i, v := range pos {
		readSize, err := d.user.ReadAt(buffer[idx:idx+4], v)
		if err != nil {
			return nil, err
		}
		if readSize != rwUnitBytes {
			return nil, fmt.Errorf("[user] readSize %d not match with %d at 0x%x... %s", readSize, rwUnitBytes, v, detail[i])
		}
		idx += readSize
	}

	return buffer, nil
}

// Reset resets device
func (d *FPGADevice) Reset() error {
	buffer := [][]byte{{0x00, 0x00, 0x00, 0x00}, {0xff, 0xff, 0xff, 0xff}, {0x00, 0x00, 0x00, 0x00}}

	for _, v := range buffer {
		writeSize, err := d.user.WriteAt(v, 0x1800c)
		if err != nil {
			return err
		}
		if writeSize != rwUnitBytes {
			return fmt.Errorf("[user] writeSize %d not match with %d at 0x%x... %s", writeSize, rwUnitBytes, 0x1800c, "ecc_reset")
		}
	}

	return nil
}

// Version read mbpu version information from mbpu
func (d *FPGADevice) Version() (string, error) {
	buffer := make([]byte, rwUnitBytes)

	idx := 0
	detail := []string{"FPGA_INFO"}
	pos := []int64{0x18000}
	for i, v := range pos {
		readSize, err := d.user.ReadAt(buffer[idx:idx+4], v)
		if err != nil {
			return "", err
		}
		if readSize != rwUnitBytes {
			return "", fmt.Errorf("[user] readSize %d not match with %d at 0x%x... %s", readSize, rwUnitBytes, v, detail[i])
		}
		idx += readSize
	}

	u := binary.LittleEndian.Uint32(buffer[0:4])
	return fmt.Sprintf("%x\n", u), nil
}
