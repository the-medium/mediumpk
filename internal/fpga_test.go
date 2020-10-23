package internal

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFPGA_OpenNClose(t *testing.T) {
	fpga, err := NewFPGADevice(0)
	assert.NoError(t, err)
	assert.NotNil(t, fpga)

	err = fpga.Close()
	assert.NoError(t, err)
}

func TestFPGA_Sign_CPU_Verify(t *testing.T) {
	// generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	// get hashed message
	msg := []byte("Hello World")
	hasher := sha256.New()
	hasher.Write(msg)
	h32 := hasher.Sum(nil)

	// newfpgadevice
	dev, err := NewFPGADevice(0)
	assert.NoError(t, err)
	assert.NotNil(t, dev)

	// userctx
	userctx := 0xab
	reqBuf, err := genSignSet(privateKey, h32, userctx)
	assert.NoError(t, err)
	// send request
	err = dev.Request(reqBuf)
	assert.NoError(t, err)

	// poll result
	respBuf, err := dev.Poll()
	assert.NoError(t, err)
	assert.Equal(t, ResponseSize, len(respBuf))
	assert.Equal(t, userctx, int(binary.BigEndian.Uint64(respBuf[8:16])))
	assert.Equal(t, 0, int(binary.BigEndian.Uint32(respBuf[4:8])))

	// close fpga
	err = dev.Close()
	assert.NoError(t, err)

	// verify with cpu
	publicKey := privateKey.PublicKey
	rBig := new(big.Int)
	rBig.SetBytes(respBuf[16:48])
	sBig := new(big.Int)
	sBig.SetBytes(respBuf[48:80])
	ok := ecdsa.Verify(&publicKey, h32, rBig, sBig)
	assert.True(t, ok)
}

func TestCPU_Sign_FPGA_Verify(t *testing.T) {
	// newfpgadevice
	dev, err := NewFPGADevice(0)
	assert.NoError(t, err)
	assert.NotNil(t, dev)

	userctx := 0xabc
	reqBuf, err := genVerifySet(userctx)
	assert.NoError(t, err)

	// send verify request
	err = dev.Request(reqBuf)
	assert.NoError(t, err)

	// poll result
	respBuf, err := dev.Poll()
	assert.NoError(t, err)
	assert.Equal(t, ResponseSize, len(respBuf))
	assert.Equal(t, userctx, int(binary.BigEndian.Uint64(respBuf[8:16])))
	assert.Equal(t, 0, int(binary.BigEndian.Uint32(respBuf[4:8])))

	// close fpga
	err = dev.Close()
	assert.NoError(t, err)
}

func TestFPGADevice_CheckAvailable(t *testing.T) {
	dev, err := NewFPGADevice(0)
	assert.NoError(t, err)
	assert.NotNil(t, dev)

	err = dev.CheckAvailable()
	assert.NoError(t, err)

	err = dev.Close()
	assert.NoError(t, err)
}

func TestFPGADevice_GetMetrics(t *testing.T) {
	dev, err := NewFPGADevice(0)
	assert.NoError(t, err)
	assert.NotNil(t, dev)

	buffer, err := dev.GetMetrics()
	assert.NoError(t, err)
	assert.NotNil(t, buffer)

	err = dev.Close()
	assert.NoError(t, err)
}

func TestFPGADevice_Reset(t *testing.T) {
	dev, err := NewFPGADevice(0)
	assert.NoError(t, err)
	assert.NotNil(t, dev)

	userctx := 0xabc
	reqBuf, err := genVerifySet(userctx)
	assert.NoError(t, err)

	// send verify request (fill C2H FIFO)
	for i := 1; i < 10; i++ {
		err = dev.Request(reqBuf)
		assert.NoError(t, err)
	}

	// Clear C2H FIFO
	err = dev.Reset()
	assert.NoError(t, err)

	// Polling Empty C2H FIFO => Input/Output Error
	_, err = dev.Poll()
	assert.Error(t, err)

	err = dev.Close()
	assert.NoError(t, err)
}

func TestFPGADevice_Version(t *testing.T) {
	dev, err := NewFPGADevice(0)
	assert.NoError(t, err)
	assert.NotNil(t, dev)

	v, err := dev.Version()

	assert.NoError(t, err)
	assert.NotEqual(t, "", v)

	err = dev.Close()
	assert.NoError(t, err)
}

func genSignSet(privateKey *ecdsa.PrivateKey, h32 []byte, userctx int) ([]byte, error) {

	d := privateKey.D.Bytes()
	d32 := make([]byte, 32)
	copy(d32[32-len(d):], d[:])

	// generate random k
	k32 := make([]byte, 32)
	rand.Read(k32)

	// create buffer for sign request
	reqBuf := make([]byte, SignRequestSize)

	// fill parameters into buffer
	binary.BigEndian.PutUint64(reqBuf[0:8], 12297829379609722880)
	binary.BigEndian.PutUint64(reqBuf[8:16], uint64(userctx))
	i := 16
	i += copy(reqBuf[i:], d32)
	i += copy(reqBuf[i:], k32)
	i += copy(reqBuf[i:], h32)

	return reqBuf, nil
}

func genVerifySet(userctx int) ([]byte, error) {
	// generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	// get hashed message
	msg := []byte("Hello World")
	hasher := sha256.New()
	hasher.Write(msg)
	h32 := hasher.Sum(nil)

	// sign with cpu
	_r, _s, err := ecdsa.Sign(rand.Reader, privateKey, h32)
	if err != nil {
		return nil, err
	}

	// set qx32, qy32, r32, s32
	qx := privateKey.PublicKey.X.Bytes()
	qy := privateKey.PublicKey.Y.Bytes()
	r := _r.Bytes()
	s := _s.Bytes()

	qx32 := make([]byte, 32)
	qy32 := make([]byte, 32)
	r32 := make([]byte, 32)
	s32 := make([]byte, 32)

	copy(qx32[32-len(qx):], qx[:])
	copy(qy32[32-len(qy):], qy[:])
	copy(r32[32-len(r):], r[:])
	copy(s32[32-len(s):], s[:])

	// userctx

	// create buffer for verify request
	reqBuf := make([]byte, VerifyRequestSize)

	// fill parameters into buffer
	binary.BigEndian.PutUint64(reqBuf[0:8], 13527612317570695168)
	binary.BigEndian.PutUint64(reqBuf[8:16], uint64(userctx))
	i := 16
	i += copy(reqBuf[i:], qx32)
	i += copy(reqBuf[i:], qy32)
	i += copy(reqBuf[i:], r32)
	i += copy(reqBuf[i:], s32)
	i += copy(reqBuf[i:], h32)

	return reqBuf, nil
}
