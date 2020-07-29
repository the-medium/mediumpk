package mediumpk

import (
	"errors"

	"github.com/the-medium/mediumpk/internal"
)

// Mediumpk is a structure to interact with FPGA
type Mediumpk struct{
	dev 		*internal.FPGADevice
	chanStore 	[]*chan ResponseEnvelop
}

// New creates and returns Mediumpk instance
func New(maxPending int) (*Mediumpk, error){
	dev, err := internal.NewFPGADevice(0)
	if(err != nil){
		return nil, err
	}

	return &Mediumpk{dev, make([]*chan ResponseEnvelop, maxPending)}, nil
}

// Close releases Mediumpk instance
func(m *Mediumpk) Close() error{
	return m.dev.Close()
}

// Request send sign/verify request to FPGA
func(m *Mediumpk) Request(pchan *chan ResponseEnvelop, env RequestEnvelop) (bool, error){
	idx, err := m.putChannel(pchan)
	if(err != nil){
		return false, err
	}

	return m.dev.Request(env.Bytes(serializer{}, idx))
}


// GetResponseAndNotify get response from FPGA and send it to channel
func(m *Mediumpk) GetResponseAndNotify() (bool, error){
	buffer, err := m.dev.Poll()
	if(err != nil){
		return false, err 
	}

	var resEnv ResponseEnvelop 
	idx, err := resEnv.Deserialize(deserializer{}, buffer)
	if(err != nil){
		return false, err
	}
	
	ch, err := m.getChannel(idx)
	if(err != nil){
		return false, err
	}
	
	*ch <- resEnv

	return true, nil
}

// must not be called concurrently
func(m *Mediumpk) putChannel(resChan *chan ResponseEnvelop) (int, error){
	for i, c := range m.chanStore {
		if c == nil{
			m.chanStore[i] = resChan
			return i, nil
		}
	}
	return -1, errors.New("no empty channel Store")
}

// must not be called concurrently
func(m *Mediumpk) getChannel(i int) (*chan ResponseEnvelop, error){
	if( i >= len(m.chanStore)){
		return nil, errors.New("out of range")
	}
	if m.chanStore[i] == nil {
		return nil, errors.New("nil chanStore")
	}
	resChan :=  m.chanStore[i]
	m.chanStore[i] = nil
	return resChan, nil
}