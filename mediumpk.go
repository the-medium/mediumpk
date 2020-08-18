package mediumpk

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/the-medium/mediumpk/internal"
)

// Mediumpk is a structure to interact with FPGA
type Mediumpk struct{
	index		int
	dev 		*internal.FPGADevice
	chanStore 	[]*chan ResponseEnvelop
	chanEnd		chan bool
	socketAddr	string
	count 		int32
}

// New creates and returns Mediumpk instance
func New(index int, maxPending int) (*Mediumpk, error){
	dev, err := internal.NewFPGADevice(index)
	if(err != nil){
		return nil, err
	}
	
	socketAddr := "/var/run/mbpu" + strconv.Itoa(index) + ".sock"
	
	return &Mediumpk{index, dev, make([]*chan ResponseEnvelop, maxPending), make(chan bool, 1), socketAddr, 0}, nil
}

// Close releases Mediumpk instance
func(m *Mediumpk) Close() error{
	return m.dev.Close()
}

// Request send sign/verify request to FPGA
func(m *Mediumpk) Request(pchan *chan ResponseEnvelop, env RequestEnvelop) error {
	idx, err := m.putChannel(pchan)
	if(err != nil){
		return err
	}

	atomic.AddInt32(&m.count, 1)

	return m.dev.Request(env.Bytes(serializer{}, idx))
}


// GetResponseAndNotify get response from FPGA and send it to channel
func(m *Mediumpk) GetResponseAndNotify() (err error){
	buffer, err := m.dev.Poll()
	if(err != nil){
		return  
	}

	var resEnv ResponseEnvelop 
	idx, err := resEnv.Deserialize(deserializer{}, buffer)
	if(err != nil){
		return
	}
	
	ch, err := m.getChannel(idx)
	if(err != nil){
		return
	}

	atomic.AddInt32(&m.count, -1)
	
	*ch <- resEnv

	return
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

// StartMetric starts unix socket server to export metrics
func (m *Mediumpk) StartMetric(){
	go func(){
		if err := os.RemoveAll(m.socketAddr); err != nil {
			log.Fatal(err)
		}

		l, err := net.Listen("unix", m.socketAddr)
		if err != nil {
			log.Fatal("listen error:", err)
		}
		defer l.Close()
	
		for {
			select{
			case <- m.chanEnd:
				break
			default:
				// Accept new connections, dispatching them to echoServer
				// in a goroutine.
				conn, err := l.Accept()
				if err != nil {
					log.Fatal("accept error:", err)
				}
		
				go m.echoServer(conn)	
			}
		}
		m.chanEnd <- true
	}()
}

// StopMetric stops unix socket server
func (m *Mediumpk) StopMetric() (err error) {
	m.chanEnd <- true
	ticker := time.NewTicker(time.Duration(1) * time.Second)
	leftCount := 10
	for leftCount > 0{
		select{
		case <- m.chanEnd:
			leftCount = -1
		case <- ticker.C:
			fmt.Printf("metric server goroutine is not responding. check count left : %d\n", leftCount)
			leftCount--
		}
	}

	if leftCount == 0{
		err = fmt.Errorf("metric server goroutine is not stopped properly")
	}else{
		log.Println("metric server goroutine is stopped")
	}
	
	return nil
}

func (m *Mediumpk) echoServer(c net.Conn) {
	var resEnv MetricEnvelop
	buffer, err := m.dev.GetMetrics()
	if err != nil {
		log.Println(err)
	}
	
	err = resEnv.Deserialize(deserializer{}, buffer)
	if err != nil {
		log.Println(err)
	}

	vccint, vccaux, vccbram := resEnv.Voltages()
	// msg := []byte(fmt.Sprintf("Temperature:%s vccint:%s vccaux:%s vccbram:%s count:%d\n", resEnv.Temperature(), vccint, vccaux, vccbram, resEnv.Count()))
	msg := []byte(fmt.Sprintf("Temperature:%s vccint:%s vccaux:%s vccbram:%s count:%d\n", resEnv.Temperature(), vccint, vccaux, vccbram, m.count))
	c.Write(msg)
	c.Close()
}

func (m *Mediumpk) GetVersion() (string, error) {
	return m.dev.Version()
}