package mediumpk

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	chars            []rune     = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖabcdefghijklmnopqrstuvwxyzåäö0123456789")
	data             []*dataset = make([]*dataset, dataCount)
	dataCount        int        = 10000
	testDataFileName string     = "dat"

	mbpuCount        int    = 1
	maxPending       int    = 64
	metricSocketPath string = "/tmp"
)

type dataset struct {
	d  []byte
	qx []byte
	qy []byte
	r  []byte
	s  []byte
	h  []byte
}

func TestMain(m *testing.M) {
	var err error
	err = setUp(testDataFileName, dataCount)
	if err != nil {
		fmt.Println("Failed setUp test", err.Error())
		os.Exit(-1)
	}

	ret := m.Run()
	tearDown(testDataFileName)
	if ret != 0 {
		fmt.Printf("Failed testing\n")
		os.Exit(-1)
	}

	os.Exit(0)
}

func BenchmarkSign(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sign()
	}
}

func BenchmarkSignParallel(b *testing.B) {
	parallel := 300 / runtime.GOMAXPROCS(0)
	b.SetParallelism(parallel)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sign()
		}
	})
}

func BenchmarkVerify(b *testing.B) {
	for i := 0; i < b.N; i++ {
		verify()
	}
}

func BenchmarkVerifyParallel(b *testing.B) {
	parallel := 300 / runtime.GOMAXPROCS(0)
	b.SetParallelism(parallel)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			verify()
		}
	})
}

func setUp(fileName string, dataCount int) error {
	// run fpgaManager

	f, err := os.Create("./" + fileName)
	if err != nil {
		fmt.Printf("Could not create data file [%s]", err)
		os.Exit(-1)
	}
	defer f.Close()

	for i := 0; i < dataCount; i++ {
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return fmt.Errorf("Failed generating ECDSA key for [%v]: [%s]", elliptic.P256(), err)
		}

		// private Key
		D := hex.EncodeToString(privKey.D.Bytes())

		// X coordinate of public key
		Qx := hex.EncodeToString(privKey.PublicKey.X.Bytes())

		// Y coordinate of public key
		Qy := hex.EncodeToString(privKey.PublicKey.Y.Bytes())

		// message digest
		randstr := randString()
		hasher := sha256.New()
		hasher.Write([]byte(randstr))
		h := hasher.Sum(nil)
		H := hex.EncodeToString(h)

		// signature
		// cpu sign
		r, s, err := ecdsa.Sign(rand.Reader, privKey, h)
		R := hex.EncodeToString(r.Bytes())
		S := hex.EncodeToString(s.Bytes())
		if err != nil {
			fmt.Printf(err.Error())
		}

		line := fmt.Sprintf("%s %s %s %s %s %s\n", D, Qx, Qy, R, S, H)
		_, _ = f.WriteString(line)
	}

	file, err := os.Open("./dat")
	if err != nil {
		fmt.Println("cannot open file")
	}

	reader := bufio.NewReader(file)

	for i := 0; i < dataCount; i++ {
		line, _, err := reader.ReadLine()

		if err == io.EOF {
			break
		}
		_data, err := parseData(string(line))
		if err != nil {
			return err
		}
		data[i] = _data
	}

	maxProcs := runtime.GOMAXPROCS(0)
	fmt.Println("GOMAXPROCS : ", strconv.Itoa(maxProcs))
	err = InitMBPUManager(mbpuCount, maxPending, metricSocketPath)

	return err
}

func tearDown(fileName string) {
	var err = os.Remove("./" + fileName)
	if err != nil {
		fmt.Printf("Could not remove data file [%s]", err)
		os.Exit(-1)
	}

	err = CloseMBPUManager()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}
}

func randString() string {
	mrand.Seed(time.Now().UnixNano())

	length := 10
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[mrand.Intn(len(chars))])
	}
	return b.String() // E.g. "ExcbsVQs"
}

func parseData(data string) (*dataset, error) {
	arr := strings.Split(data, " ")

	d, err := hex.DecodeString(arr[0])
	if err != nil {
		return nil, err
	}

	qx, err := hex.DecodeString(arr[1])
	if err != nil {
		return nil, err
	}

	qy, err := hex.DecodeString(arr[2])
	if err != nil {
		return nil, err
	}

	r, err := hex.DecodeString(arr[3])
	if err != nil {
		return nil, err
	}

	s, err := hex.DecodeString(arr[4])
	if err != nil {
		return nil, err
	}

	h, err := hex.DecodeString(arr[5])
	if err != nil {
		return nil, err
	}

	return &dataset{d, qx, qy, r, s, h}, nil
}

func sign() {
	workload := data[mrand.Int()%len(data)]
	d := workload.d
	h := workload.h
	k, err := CreateRandomK(d, h)
	if err != nil {
		fmt.Println("Failed to create random k")
	}

	d32 := make([]byte, 32)
	k32 := make([]byte, 32)
	h32 := make([]byte, 32)
	copy(d32[32-len(d):], d[:])
	copy(k32[32-len(k):], k[:])
	copy(h32[32-len(h):], h[:])
	var reqEnv RequestEnvelop
	reqEnv = SignRequestEnvelop{
		D: d32,
		K: k32,
		H: h32,
	}

	result, _, _ := Request(reqEnv)
	if result != 0 {
		if result == -1 {
			x := new(big.Int)
			x.SetBytes(workload.qx)
			y := new(big.Int)
			y.SetBytes(workload.qy)
			pub := ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     x,
				Y:     y,
			}
			d := new(big.Int)
			d.SetBytes(d32)
			priv := &ecdsa.PrivateKey{
				PublicKey: pub,
				D:         d,
			}
			_, _, err := ecdsa.Sign(rand.Reader, priv, h32)
			if err != nil {
				fmt.Printf("sign generation result : %t\n", false)
			}

		} else {
			fmt.Printf("sign generation result : %d\n", result)
		}
	}
}

func verify() {
	workload := data[mrand.Int()%len(data)]

	qx := workload.qx
	qy := workload.qy
	r := workload.r
	s := workload.s
	h := workload.h

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

	var reqEnv RequestEnvelop
	reqEnv = VerifyRequestEnvelop{
		Qx: qx32,
		Qy: qy32,
		R:  r32,
		S:  s32,
		H:  h32,
	}

	result, _, _ := Request(reqEnv)
	if result != 0 {
		if result == -1 {
			x := new(big.Int)
			x.SetBytes(qx32)
			y := new(big.Int)
			y.SetBytes(qy32)
			r := new(big.Int)
			r.SetBytes(r32)
			s := new(big.Int)
			s.SetBytes(s32)
			pub := &ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     x,
				Y:     y,
			}
			res := ecdsa.Verify(pub, h32, r, s)
			if !res {
				fmt.Printf("verification result : %d\n", result)
			}
		} else {
			fmt.Printf("verification result : %d\n", result)
		}
	}

}
