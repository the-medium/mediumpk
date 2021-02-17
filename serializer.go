package mediumpk

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	"github.com/the-medium/mediumpk/internal"
)

type serializer struct{}

func (s *serializer) serializeSignRequest(env SignRequestEnvelop, userctx int) []byte {
	tmp := make([]byte, internal.SignRequestSize)
	binary.BigEndian.PutUint64(tmp[0:], 12297829379609722880)

	var i int = 16
	binary.BigEndian.PutUint64(tmp[8:], uint64(userctx))
	i += copy(tmp[i:], env.D)
	i += copy(tmp[i:], env.K)
	i += copy(tmp[i:], env.H)

	return tmp
}

func (s *serializer) serializeVerifyRequest(env VerifyRequestEnvelop, userctx int) []byte {
	tmp := make([]byte, internal.VerifyRequestSize)

	binary.BigEndian.PutUint64(tmp[0:8], 13527612317570695168)
	binary.BigEndian.PutUint64(tmp[8:16], uint64(userctx))

	var i int = 16
	i += copy(tmp[i:], env.Qx)
	i += copy(tmp[i:], env.Qy)
	i += copy(tmp[i:], env.R)
	i += copy(tmp[i:], env.S)
	i += copy(tmp[i:], env.H)

	return tmp
}

type deserializer struct{}

func (s *deserializer) deserializeResponse(env *ResponseEnvelop, buffer []byte) (int, error) {
	if len(buffer) != internal.ResponseSize {
		return 0, errors.New("wrong responseEnvelopSize : " + strconv.Itoa(len(buffer)))
	}

	env.result = int(binary.BigEndian.Uint32(buffer[4:8]))
	env.r = make([]byte, 32)
	env.s = make([]byte, 32)
	copy(env.r, buffer[16:48])
	copy(env.s, buffer[48:80])

	return int(binary.BigEndian.Uint64(buffer[8:16])), nil
}

func (s *deserializer) deserializeMetric(env *MetricEnvelop, buffer []byte) error {
	if len(buffer) != internal.MetricSetSize {
		return errors.New("wrong MetricSetSize : " + strconv.Itoa(len(buffer)))
	}

	tempFloat32 := (float32(binary.LittleEndian.Uint32(buffer[0:4])) * 501.3743 / 65536) - 273.6777
	temperature := fmt.Sprintf("%f", tempFloat32)
	env.temperature = temperature

	vccIntFloat32 := (float32(binary.LittleEndian.Uint32(buffer[4:8])) / 65536) * 3
	vccint := fmt.Sprintf("%f", vccIntFloat32)
	env.vccint = vccint

	vccAuxFloat32 := (float32(binary.LittleEndian.Uint32(buffer[8:12])) / 65536) * 3
	vccaux := fmt.Sprintf("%f", vccAuxFloat32)
	env.vccaux = vccaux

	vccBramFloat32 := (float32(binary.LittleEndian.Uint32(buffer[12:16])) / 65536) * 3
	vccbram := fmt.Sprintf("%f", vccBramFloat32)
	env.vccbram = vccbram

	env.signCount = int(binary.LittleEndian.Uint32(buffer[16:20]))
	env.verifyCount = int(binary.LittleEndian.Uint32(buffer[20:24]))
	env.errorCount = int(binary.LittleEndian.Uint32(buffer[24:28]))

	return nil
}
