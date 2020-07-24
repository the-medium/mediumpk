package internal

import (
	"errors"
	"log"
	"os"
	"strconv"
)

const (
	// SignRequestSize is buffer size of sign request
	SignRequestSize		= 128
	// VerifyRequestSize is buffer size of verify request
	VerifyRequestSize	= 192
	// ResponseSize is buffer size of response
	ResponseSize		= 96
)

// FPGADevice is a structue to store device file descriptors
type FPGADevice struct {
	h2c *os.File
	c2h *os.File
	ctrl *os.File
	user *os.File
}

// NewFPGADevice returns FPGADevice instance
func NewFPGADevice(index int) (*FPGADevice, error) {
	prefix := "/dev/mdlx" + strconv.Itoa(index)
	
	h2c, err := os.OpenFile(prefix + "_h2c_0", os.O_WRONLY | os.O_EXCL, os.ModeDevice)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	c2h, err := os.OpenFile(prefix + "_c2h_0", os.O_RDONLY | os.O_EXCL, os.ModeDevice)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	ctrl, err := os.OpenFile(prefix + "_control", os.O_RDONLY | os.O_EXCL, os.ModeDevice)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	user, err := os.OpenFile(prefix + "_user", os.O_RDONLY | os.O_EXCL, os.ModeDevice)
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

	return &dev, nil
}

// Close closes device descriptors
func (d *FPGADevice) Close() (err error){
	err = d.h2c.Close()
	if err != nil{
		log.Fatal(err)
	}
	err = d.c2h.Close()
	if err != nil{
		log.Fatal(err)
	}
	err = d.ctrl.Close()
	if err != nil{
		log.Fatal(err)
	}
	err = d.user.Close()
	if err != nil{
		log.Fatal(err)
	}

	return
}

// Request send request into FPGA
func (d *FPGADevice) Request(buffer []byte) (bool, error) {
	writeSize, err := d.h2c.Write(buffer)
	if err != nil {
		return false, err
	}

	if writeSize != len(buffer) {
		err = errors.New("write size not match.." + strconv.Itoa(writeSize))
		return false, err
	}

	return true, nil
}

// Poll brings result from FPGA
func (d *FPGADevice) Poll() ([]byte, error){
	buffer := make([]byte, ResponseSize)

	readSize, err := d.c2h.Read(buffer)
	if err != nil {
		return nil, err
	}

	if readSize != ResponseSize{
		err = errors.New("read size not match.." + strconv.Itoa(readSize))
		return nil, err
	}

	return buffer, nil
}