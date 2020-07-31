package mediumpk

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/the-medium/mediumpk/internal"
)

// Mediumpk is a structure to interact with FPGA
type Mediumpk struct{
	index		int
	dev 		*internal.FPGADevice
	chanStore 	[]*chan ResponseEnvelop
	chanStopMetric chan bool
	metricExportDir	string
}

// New creates and returns Mediumpk instance
func New(index int, maxPending int, metricExportDir string) (*Mediumpk, error){
	dev, err := internal.NewFPGADevice(index)
	if(err != nil){
		return nil, err
	}
	if metricExportDir == ""{
		metricExportDir = "./"
	}
	return &Mediumpk{index, dev, make([]*chan ResponseEnvelop, maxPending), make(chan bool, 1), metricExportDir}, nil
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

func (m *Mediumpk) startMetric(interval int){
	go func(interval int){
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		var resEnv MetricEnvelop
		fileName := fmt.Sprintf("%s/fpga%d.stat", m.metricExportDir, m.index)
		stop := false
		for !stop {
			select{
			case <- m.chanStopMetric:
				stop = true
			case <- ticker.C:
				buffer, err := m.dev.GetMetrics()
				err = resEnv.Deserialize(deserializer{}, buffer)
				if err != nil {
					fmt.Println(err)
				}

				vccint, vccaux, vccbram := resEnv.Voltages() 		
				fd, err := os.OpenFile(fileName, os.O_WRONLY | os.O_TRUNC | os.O_CREATE, os.FileMode(0644))
				if err != nil {
					fmt.Print(err)
				}else{
					fd.WriteString(fmt.Sprintf("Temperature:%s\nvccint:%s\nvccaux:%s\nvccbram:%s\ncount:%d", resEnv.Temperature(), vccint, vccaux, vccbram, resEnv.Count()))
				}
				fd.Close()
			}
		}
		fmt.Printf("stopping Metric goroutine\n")
		m.chanStopMetric <- true
	}(interval)
}
func (m *Mediumpk) stopMetric() (err error) {
	m.chanStopMetric <- true
	ticker := time.NewTicker(time.Duration(1) * time.Second)
	leftCount := 10
	for leftCount > 0{
		select{
		case <- m.chanStopMetric:
			leftCount = -1
		case <- ticker.C:
			fmt.Printf("Metric goroutine is not responding. check count left : %d\n", leftCount)
			leftCount--
		}
	}

	if leftCount == 0{
		err = fmt.Errorf("metric goroutine is not stopped properly")
	}
	return
}