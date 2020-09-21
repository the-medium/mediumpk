/*
Copyright Medium Corp. 2020 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mediumpk

import (
	"bytes"
	"log"
	"sync"
	"sync/atomic"
)

var (
	buf         bytes.Buffer
	fm          *mbpuManager = nil
	lock                     = &sync.Mutex{}
	loggerInfo               = log.New(&buf, "[MBPU][INFO] : ", log.Lshortfile|log.Ldate|log.Ltime|log.LUTC)
	loggerError              = log.New(&buf, "[MBPU][ERRO] : ", log.Lshortfile)
)

type requestWrapper struct {
	env      RequestEnvelop
	respChan chan ResponseEnvelop
}

type mbpuManager struct {
	mpk             *Mediumpk
	available       int32
	chanAvailable   chan bool
	chanIsAvailable chan bool
	chanRequest     chan requestWrapper
	chanPoll        chan bool
	chanEmergency   chan bool
}

// InitMBPUManager opens MBPU device and runs goroutine each for request/response to/from MBPU
func InitMBPUManager(index int, maxPending int, metricSocketPath string) (err error) {
	if fm != nil {
		loggerError.Println("MBPUManager is already initialized ...")
		return
	}

	lock.Lock()
	defer lock.Unlock()

	mpk, err := newMediumpk(index, maxPending, metricSocketPath)
	if err != nil {
		return
	}
	var available int32 = int32(maxPending)
	fm = &mbpuManager{
		mpk,
		available,
		make(chan bool, 1),
		make(chan bool),
		make(chan requestWrapper),
		make(chan bool, maxPending),
		make(chan bool, 1),
	}
	fm.mpk.startMetric()
	go fm.runPushing()
	go fm.runPolling()

	loggerInfo.Println("MBPUManager Initialized... MAX_PENDING : ", maxPending)
	return
}

// CloseMBPUManager closes MBPU Device and stops goroutines for request/response to/from MBPU
func CloseMBPUManager() error {
	lock.Lock()
	defer lock.Unlock()

	close(fm.chanRequest)
	close(fm.chanPoll)
	loggerInfo.Println("Close MBPUManager Request channel")

	err := fm.mpk.stopMetric()
	if err != nil {
		return err
	}

	err = fm.mpk.close()
	if err != nil {
		return err
	}

	fm = nil
	loggerInfo.Println("Close MBPUManager")
	return nil
}

// Request send RequestEnvelop to push-goroutine with channel for receive response
func Request(env RequestEnvelop) (bool, []byte, []byte) {
	respChan := make(chan ResponseEnvelop, 1)
	req := requestWrapper{
		env,
		respChan,
	}

	fm.chanRequest <- req
	respEnv := <-respChan
	close(respChan)

	r, s := respEnv.Signature()
	result := false
	if respEnv.Result() == 0 {
		result = true
	}
	return result, r, s
}

func (fm *mbpuManager) runPushing() {
	stop := false

	for !stop {
		req, ok := <-fm.chanRequest
		if !ok {
			// break pushing
			stop = true
			continue
		}

		if atomic.LoadInt32(&fm.available) == 0 {
			fm.chanIsAvailable <- true
			<-fm.chanAvailable
		}

		fm.chanPoll <- true
		for {
			idx, err := fm.mpk.request(&req.respChan, req.env)
			if err == nil { // good to go
				atomic.AddInt32(&fm.available, -1)
				break
			}

			// check error type
			if idx == -1 { // maxPending refuse error... try again
				loggerError.Println(err.Error() + ", try again..")
				continue
			} else { // something has gone wrong
				stop = true
				loggerError.Println(err)
			}
		}

	}
	fm.chanEmergency <- false
}

func (fm *mbpuManager) runPolling() {
	stop := false

	for !stop {
		_, ok := <-fm.chanPoll
		if !ok {
			// break polling
			stop = true
			continue
		}

		err := fm.mpk.getResponseAndNotify()
		if err != nil {
			stop = true
			loggerError.Println(err)
		}

		select {
		case <-fm.chanIsAvailable:
			atomic.AddInt32(&fm.available, 1)
			fm.chanAvailable <- true
		default:
			atomic.AddInt32(&fm.available, 1)
		}
	}
}
