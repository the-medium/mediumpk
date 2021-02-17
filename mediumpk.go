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
type Mediumpk struct {
	index      int
	dev        *internal.FPGADevice
	chanStore  []*chan ResponseEnvelop
	chanEnd    chan bool
	socketAddr string
	count      int32
	emergency  int
	metricOn   bool
}

// New creates and returns Mediumpk instance
func newMediumpk(index int, maxPending int, socketPath string) (*Mediumpk, error) {
	if socketPath == "" {
		socketPath = "/var/run/"
	} else {
		_, err := os.Stat(socketPath)
		if err != nil {
			return nil, err
		}
	}
	socketAddr := fmt.Sprintf("%s%s%s%s", socketPath, "/mbpu", strconv.Itoa(index), ".sock")

	dev, err := internal.NewFPGADevice(index)
	if err != nil {
		return nil, err
	}

	return &Mediumpk{index, dev, make([]*chan ResponseEnvelop, maxPending), make(chan bool, 1), socketAddr, 0, 0, false}, nil
}

// Close releases Mediumpk instance
func (m *Mediumpk) close() error {
	m.stopMetric()
	return m.dev.Close()
}

// Request send sign/verify request to FPGA
func (m *Mediumpk) request(pchan *chan ResponseEnvelop, env RequestEnvelop) (int, error) {
	idx, err := m.putChannel(pchan)
	if err != nil {
		return idx, err
	}

	atomic.AddInt32(&m.count, 1)

	return idx, m.dev.Request(env.Bytes(serializer{}, idx))
}

// getResponseAndNotify get response from FPGA and send it to channel
func (m *Mediumpk) getResponseAndNotify() (err error) {
	buffer, err := m.dev.Poll()
	if err != nil {
		return
	}

	var resEnv ResponseEnvelop
	idx, err := resEnv.Deserialize(deserializer{}, buffer)
	if err != nil {
		return
	}

	ch, err := m.getChannel(idx)
	if err != nil {
		return
	}

	atomic.AddInt32(&m.count, -1)

	*ch <- resEnv

	return
}

// must not be called concurrently
func (m *Mediumpk) putChannel(resChan *chan ResponseEnvelop) (int, error) {
	for i, c := range m.chanStore {
		if c == nil {
			m.chanStore[i] = resChan
			return i, nil
		}
	}
	return -1, errors.New("no empty channel Store")
}

// must not be called concurrently
func (m *Mediumpk) getChannel(i int) (*chan ResponseEnvelop, error) {
	if i >= len(m.chanStore) {
		return nil, errors.New("out of range")
	}
	if m.chanStore[i] == nil {
		return nil, errors.New("nil chanStore")
	}
	resChan := m.chanStore[i]
	m.chanStore[i] = nil
	return resChan, nil
}

func (m *Mediumpk) clearChanStore() {
	resEnv := ResponseEnvelop{
		result: -1,
		r:      []byte(nil),
		s:      []byte(nil),
	}
	for i := 0; i < len(m.chanStore); i++ {
		if m.chanStore[i] != nil {
			*m.chanStore[i] <- resEnv
		}
	}
}

// startMetric starts unix socket server to export metrics
func (m *Mediumpk) startMetric() {
	if m.metricOn == true {
		log.Println("Metric is already started")
		return
	}

	m.metricOn = true
	go func() {
		if err := os.RemoveAll(m.socketAddr); err != nil {
			log.Fatal(err)
		}

		l, err := net.Listen("unix", m.socketAddr)
		if err != nil {
			log.Fatal("listen error:", err)
		}
		defer l.Close()

		for {
			select {
			case <-m.chanEnd:
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
	}()
}

// StopMetric stops unix socket server
func (m *Mediumpk) stopMetric() (err error) {
	if m.metricOn == false {
		return nil
	}

	m.chanEnd <- true
	ticker := time.NewTicker(time.Duration(1) * time.Second)
	leftCount := 10
	for leftCount > 0 {
		select {
		case <-m.chanEnd:
			leftCount = -1
		case <-ticker.C:
			fmt.Printf("[metric server] goroutine is not responding. check count left : %d\n", leftCount)
			leftCount--
		}
	}

	if leftCount == 0 {
		err = fmt.Errorf("[metric server] goroutine is not stopped properly")
	} else {
		log.Println("[metric server] goroutine is properly stopped")
	}

	m.metricOn = false

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
	signCount, verifyCount, errorCount := resEnv.Counter()
	msg := fmt.Sprintf(`{ "m_temperature":%s, "m_vccint":%s, "m_vccaux":%s, "m_vccbram":%s, "m_signCount":%d,"m_verifyCount":%d,"m_errorCount":%d, "m_emergency":%d }`, resEnv.Temperature(), vccint, vccaux, vccbram, signCount, verifyCount, errorCount, m.emergency)
	msgBytes := []byte(msg)
	c.Write(msgBytes)
	c.Close()
}

// GetVersion return mbpu version imformation
func (m *Mediumpk) getVersion() (string, error) {
	return m.dev.Version()
}
