package internal

import (
	"log"
	"os"
	"strconv"
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