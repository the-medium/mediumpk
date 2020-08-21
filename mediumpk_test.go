package mediumpk

import(
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)
var maxPending int = 100
var devIndex int = 0

func TestMediumpk_New(t *testing.T) {
	// new mediumpk
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	// close mediumpk
	err = mediumpk.Close()
	assert.NoError(t, err)
}

func TestMediumpk_New_Empty_SocketPath(t *testing.T) {
	// new mediumpk
	mediumpk, err := New(devIndex, maxPending, "")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	// close mediumpk
	err = mediumpk.Close()
	assert.NoError(t, err)
}

func TestMediumpk_New_Wrong_SocketPath(t *testing.T) {
	// new mediumpk
	mediumpk, err := New(devIndex, maxPending, "abcd")
	assert.Error(t, err)
	assert.Nil(t, mediumpk)
}

func TestMediumpk_Store_Channel(t *testing.T) {
	// new mediumpk
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
	chanStore :=	make([]*chan ResponseEnvelop, maxPending)

	//set slice of channel pointers
	for i := range(chanStore){
		resChan := make(chan ResponseEnvelop)
		
		assert.Nil(t, chanStore[i])
		chanStore[i] = &resChan
		assert.NotNil(t, chanStore[i])
	}
	
	// store channel pointers
	for i, v := range(chanStore){
		index, err := mediumpk.putChannel(v)
		assert.NoError(t, err)
		assert.Equal(t, i, index)
	}
	
	// full of channel pointers.. return error
	resChan := make(chan ResponseEnvelop)
	index, err := mediumpk.putChannel(&resChan)
	assert.Error(t, err)
	assert.Equal(t, index, -1)

	// get all stored channel pointers and check whether it is right one
	for i, v := range(chanStore){
		pchan, err := mediumpk.getChannel(i)
		assert.NoError(t, err)
		assert.Equal(t, v, pchan)
	}

	// close mediumpk
	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}

func TestCPU_Sign_CPU_Verify(t *testing.T){
	// generate private key
	privkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	// generate hashed
	msg := []byte("Hello World")	
	hasher := sha256.New()
	hasher.Write(msg)
	h := hasher.Sum(nil)

	// cpu sign
	r, s, err := ecdsa.Sign(rand.Reader, privkey, h)
	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.NotNil(t, s)

	// cpu verify
	result := ecdsa.Verify(&privkey.PublicKey, h, r, s)
	assert.True(t, result)
}

func TestMediumpk_Sign_Mediumpk_verify(t *testing.T){
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
	
	// generate private key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	// k
	k32 := make([]byte, 32)
    rand.Read(k32)
	
	// h
	msg := []byte("Hello World")	
	hasher := sha256.New()
	hasher.Write(msg)
	h32 := hasher.Sum(nil)

	// channel for result
	channel := make(chan ResponseEnvelop, 1)

	// mediumpk sign
	d := privKey.D.Bytes()
	d32 := make([]byte, 32)
	copy(d32[32-len(d):], d[:])
	
	var req RequestEnvelop = SignRequestEnvelop{d32, k32, h32}
	err = mediumpk.Request(&channel, req)
	assert.NoError(t, err)

	err = mediumpk.GetResponseAndNotify()
	assert.NoError(t, err)
	resp := <- channel
	assert.NotNil(t, resp)
	assert.Equal(t, resp.Result(), 0)

	// set r, s
	r, s := resp.Signature()

	// X coordinate of public key
	qx := privKey.PublicKey.X.Bytes()

	// Y coordinate of public key
	qy := privKey.PublicKey.Y.Bytes()

	// mediumpk verify
	qx32 := make([]byte, 32)
	qy32 := make([]byte, 32)
	r32 := make([]byte, 32)
	s32 := make([]byte, 32)

	copy(qx32[32-len(qx):], qx[:])
	copy(qy32[32-len(qy):], qy[:])
	copy(r32[32-len(r):], r[:])
	copy(s32[32-len(s):], s[:])
	req = VerifyRequestEnvelop{qx32, qy32, r32, s32, h32}

	err = mediumpk.Request(&channel, req)
	assert.NoError(t, err)

	err = mediumpk.GetResponseAndNotify()
	assert.NoError(t, err)

	resp = <- channel
	assert.NotNil(t, resp)
	assert.Equal(t, 0, resp.Result())

	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}

func TestMediumpk_Sign_CPU_Verify(t *testing.T){
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
	
	// generate private key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	// k
	k := make([]byte, 32)
    rand.Read(k)
	
	// h
	msg := []byte("Hello World")	
	hasher := sha256.New()
	hasher.Write(msg)
	h := hasher.Sum(nil)

	// channel for result
	channel := make(chan ResponseEnvelop, 1)

	// mediumpk sign
	d := privKey.D.Bytes()
	d32 := make([]byte, 32)
	copy(d32[32-len(d):], d[:])
	var req RequestEnvelop = SignRequestEnvelop{d32, k, h}
	err = mediumpk.Request(&channel,req)
	assert.NoError(t, err)

	err = mediumpk.GetResponseAndNotify()
	assert.NoError(t, err)
	resp := <- channel
	assert.NotNil(t, resp)
	assert.Equal(t, resp.Result(), 0)

	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	_r, _s := resp.Signature()
	r := new(big.Int);r.SetBytes(_r)
	s := new(big.Int);s.SetBytes(_s)

	// cpu verify
	result := ecdsa.Verify(&privKey.PublicKey, h, r, s)
	assert.True(t, result)
	if !result{
		fmt.Println("wwwwwwww")
		fmt.Printf("resp \t\t: %v\n", resp)
		fmt.Printf("r.Bytes() \t: %x\n", r.Bytes())    	
	}
}

func TestCPU_Sign_Mediumpk_Verify(t *testing.T){
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	channel := make(chan ResponseEnvelop, 1)

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Nil(t, err)

	// X coordinate of public key
	qx := privKey.PublicKey.X.Bytes()

	// Y coordinate of public key
	qy := privKey.PublicKey.Y.Bytes()

	// message digest
	message := "Hello World!"
	hasher := sha256.New()
	hasher.Write([]byte(message))
	h := hasher.Sum(nil)

	// cpu sign
	_r, _s, err := ecdsa.Sign(rand.Reader, privKey, h)
	assert.NoError(t, err)

	r := _r.Bytes()
	s := _s.Bytes()
	
	// mediumpk verify
	qx32 := make([]byte, 32)
	qy32 := make([]byte, 32)
	r32 := make([]byte, 32)
	s32 := make([]byte, 32)
	h32 := make([]byte, 32)
	copy(qx32[32-len(qx):], qx[:])
	copy(qy32[32-len(qy):], qy[:])
	copy(r32[32-len(r):], r[:])
	copy(s32[32-len(s):], s[:])
	copy(h32[32-len(h):], h[:])
	var req RequestEnvelop = VerifyRequestEnvelop{qx32, qy32, r32, s32, h32}
	err = mediumpk.Request(&channel, req)
	assert.NoError(t, err)

	err = mediumpk.GetResponseAndNotify()
	assert.NoError(t, err)
	resp := <- channel
	assert.NotNil(t, resp)
	assert.Equal(t, 0, resp.Result())

	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}

var strD = "519b423d715f8b581f4fa8ee59f4771a5b44c8130b4e3eacca54a56dda72b464"
var strK = "94a1bbb14b906a61a280f245f9e93c7f3b4a6247824f5d33b9670787642a68de"
var strH1 = "ea5cd45052849c4ae816bbc44ed833e832af8a619ba47268aabca2744c4c6268"
var strRExpected = "f3ac8061b514795b8843e3d6629527ed2afd6b1f6a555a7acabb5e6f79c8c2ac"
var strSExpected = "6e9a1aee9981cc4a102aa7033fdf633b39be438527865373edfe90f2ea9e29ac"

func Test_Sign_FPGA_Multi(t *testing.T){
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	// D
	D := new(big.Int); D.SetString(strD, 16)
	assert.NoError(t, err)

	// K
	K := new(big.Int); K.SetString(strK, 16)
	assert.NoError(t, err)
	
	// hash
	H := new(big.Int); H.SetString(strH1, 16)
	assert.NoError(t, err)
	
	// mediumpk sign
	d32 := make([]byte, 32)
	k32 := make([]byte, 32)
	h32 := make([]byte, 32)
	copy(d32[32-len(D.Bytes()):], D.Bytes()[:])
	copy(k32[32-len(K.Bytes()):], K.Bytes()[:])
	copy(h32[32-len(H.Bytes()):], H.Bytes()[:])
		
	chList := [](*chan ResponseEnvelop){}
	for i := 0; i < maxPending; i++ {
		channel := make(chan ResponseEnvelop, 1)
		var req RequestEnvelop = SignRequestEnvelop{d32, k32, h32}
		err = mediumpk.Request(&channel, req)
		assert.NoError(t, err)
		chList = append(chList, &channel)
	}	
	for i := 0; i < maxPending; i++{
		err = mediumpk.GetResponseAndNotify()
		assert.NoError(t, err)
	}
	
	count := 0
	var resp ResponseEnvelop
	for{
		for _, v := range chList{
			select {
			case resp = <- *v:
				assert.NotNil(t, resp)
				assert.Equal(t, 0, resp.Result())
				count = count + 1
			default:
			}
		}
		if count == maxPending{
			break
		}
	}

	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}


var strX = "e305d41ab27b39c84230ab2faf34fb15e9d0543f4ac19d2520b94d71df9be5bf"
var strY = "0b97c506c163237d6e9264f7148336e524d32174754198066995a252b1a51f4e"
var strR = "5806c2774086b61c97afd87585215c09fe57233f232278c0e8976d35f0570641"
var strS = "6d8a758eb8edfeecbdab2e413bee8bc73a88a887f97a54c2a967de0afcb8b0af"
var strH2 = "00000000000000000000000000000000000000000048656c6c6f20576f726c64"

func Test_Verify_CPU(t *testing.T){
	
	// qx, qy
	X := new(big.Int); X.SetString(strX, 16)
	Y := new(big.Int); Y.SetString(strY, 16)

	pubkey := &ecdsa.PublicKey{
		elliptic.P256(),
		X,Y,
	}

	// hash
	H := new(big.Int); H.SetString(strH2, 16)
	
	// r, s
	R := new(big.Int); R.SetString(strR, 16)
	S := new(big.Int); S.SetString(strS, 16)

	result := ecdsa.Verify(pubkey, H.Bytes(), R, S)
	assert.True(t, result)
}

func Test_Verify_FPGA(t *testing.T){
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	// qx, qy
	X := new(big.Int); X.SetString(strX, 16)
	assert.NoError(t, err)
	Y := new(big.Int); Y.SetString(strY, 16)
	assert.NoError(t, err)

	// hash
	H := new(big.Int); H.SetString(strH2, 16)
	assert.NoError(t, err)
	
	// r, s
	R := new(big.Int); R.SetString(strR, 16)
	assert.NoError(t, err)
	S := new(big.Int); S.SetString(strS, 16)
	assert.NoError(t, err)

	// mediumpk verify
	qx32 := make([]byte, 32)
	qy32 := make([]byte, 32)
	r32 := make([]byte, 32)
	s32 := make([]byte, 32)
	h32 := make([]byte, 32)
	copy(qx32[32-len(X.Bytes()):], X.Bytes()[:])
	copy(qy32[32-len(Y.Bytes()):], Y.Bytes()[:])
	copy(r32[32-len(R.Bytes()):], R.Bytes()[:])
	copy(s32[32-len(S.Bytes()):], S.Bytes()[:])
	copy(h32[32-len(H.Bytes()):], H.Bytes()[:])

	channel := make(chan ResponseEnvelop, 1)
	var req RequestEnvelop = VerifyRequestEnvelop{qx32, qy32, r32, s32, h32}
	err = mediumpk.Request(&channel, req)
	assert.NoError(t, err)

	err = mediumpk.GetResponseAndNotify()
	assert.NoError(t, err)
	resp := <- channel
	assert.NotNil(t, resp)
	assert.Equal(t, 0, resp.Result())

	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}

func Test_Verify_FPGA_Multi(t *testing.T){
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	// qx, qy
	X := new(big.Int); X.SetString(strX, 16)
	assert.NoError(t, err)
	Y := new(big.Int); Y.SetString(strY, 16)
	assert.NoError(t, err)

	// hash
	H := new(big.Int); H.SetString(strH2, 16)
	assert.NoError(t, err)
	
	// r, s
	R := new(big.Int); R.SetString(strR, 16)
	assert.NoError(t, err)
	S := new(big.Int); S.SetString(strS, 16)
	assert.NoError(t, err)

	// mediumpk verify
	qx32 := make([]byte, 32)
	qy32 := make([]byte, 32)
	r32 := make([]byte, 32)
	s32 := make([]byte, 32)
	h32 := make([]byte, 32)
	copy(qx32[32-len(X.Bytes()):], X.Bytes()[:])
	copy(qy32[32-len(Y.Bytes()):], Y.Bytes()[:])
	copy(r32[32-len(R.Bytes()):], R.Bytes()[:])
	copy(s32[32-len(S.Bytes()):], S.Bytes()[:])
	copy(h32[32-len(H.Bytes()):], H.Bytes()[:])
		
	chList := [](*chan ResponseEnvelop){}
	
	for i := 0; i < maxPending; i++ {
		channel := make(chan ResponseEnvelop, 1)
		var req RequestEnvelop = VerifyRequestEnvelop{qx32, qy32, r32, s32, h32}
		err = mediumpk.Request(&channel, req)
		assert.NoError(t, err)
		chList = append(chList, &channel)
	}	
	for i := 0; i < maxPending; i++{
		err := mediumpk.GetResponseAndNotify()
		assert.NoError(t, err)
	}
	
	count := 1
	var resp ResponseEnvelop
	
	for _, v := range chList{
		select {
		case resp = <- *v:
			fmt.Printf("%03d : %d\n", count, resp.Result())
			assert.NotNil(t, resp)
			assert.Equal(t, 0, resp.Result())
	
		default:
		}
		count++
	}
		
	

	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}

func Test_startLogging(t *testing.T){
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	mediumpk.StartMetric()

	time.Sleep(1 * time.Second)

	err = mediumpk.StopMetric()
	assert.NoError(t, err)

	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}

func Test_GetVersion(t *testing.T){
	mediumpk, err := New(devIndex, maxPending, "/tmp/")
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	v, err := mediumpk.GetVersion()
	assert.NoError(t, err)
	assert.NotEqual(t, v, "")
	fmt.Println(v)
	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}