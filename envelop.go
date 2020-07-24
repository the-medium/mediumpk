package mediumpk

// RequestEnvelop is the interface for sending requst to FPGA
type RequestEnvelop interface {
	Bytes(serializer, int) []byte
}

// SignRequestEnvelop is a structure for Sign Generation Request
type SignRequestEnvelop struct {
	D           []byte
	K           []byte
	H           []byte
}


// Bytes copies value of SignRequestEnvelop into aligned memory
func (req SignRequestEnvelop) Bytes(s serializer, userctx int) []byte {
	return s.serializeSignRequest(req, userctx)
}

// VerifyRequestEnvelop is a structure for Sign Verification Request
type VerifyRequestEnvelop struct {
	Qx          []byte
	Qy          []byte
	R           []byte
	S           []byte
	H           []byte
}

// Bytes copies value of VerifyRequestEnvelop into aligned memory
func (req VerifyRequestEnvelop) Bytes(s serializer,  userctx int) []byte {
	return s.serializeVerifyRequest(req,  userctx)
}

// ResponseEnvelop is the interface to receive respose from FPGA
type ResponseEnvelop struct {
	result 	int
	r 		[]byte
	s		[]byte
}

// Deserialize fill ResponseEnvelop with data from buffer and return userctx
func(res *ResponseEnvelop) Deserialize(ds deserializer, buffer []byte) (int, error) {

	return ds.deserializeResponse(res, buffer)
}

// Signature returns signature r, s
func(res ResponseEnvelop) Signature() ([]byte, []byte){
	return res.r, res.s
}

// Result returns result of VerifyRespEnvelop
func(res ResponseEnvelop) Result() int{
	return res.result
}

// MetricEnvelop is a structrue that stores device metric infomation
type MetricEnvelop struct {
	temperature string
	vccint string
	vccaux string
	vccbram string
	count int
}

// Deserialize fill MetricEnvelop with data from buffer
func(m *MetricEnvelop) Deserialize(ds deserializer, buffer []byte) error {
	return ds.deserializeMetric(m, buffer)
}

// Temperature returns temperature
func(m MetricEnvelop) Temperature() string {
	return m.temperature
}

// Voltages returns vccint, vccaux, vccbram
func(m MetricEnvelop) Voltages() (string, string, string) {
	return m.vccint, m.vccaux, m.vccbram
}

// Count returns the number of pending requests in fpga
func(m MetricEnvelop) Count() int {
	return m.count
}