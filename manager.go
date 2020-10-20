/*
Copyright Medium Corp. 2020 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mediumpk

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

var (
	fm   *mbpuManager = nil
	lock              = &sync.Mutex{}
	// loggerInfo               = log.New(&buf, "[MBPU][INFO] : ", log.Lshortfile|log.Ldate|log.Ltime|log.LUTC)
	// loggerError              = log.New(&buf, "[MBPU][ERRO] : ", log.Lshortfile)
)

type requestWrapper struct {
	env      RequestEnvelop
	respChan chan ResponseEnvelop
}

type mbpuManager struct {
	chanRequest chan requestWrapper
	wg          *sync.WaitGroup
}

// InitMBPUManager opens MBPU device and runs goroutine each for request/response to/from MBPU
func InitMBPUManager(mbpuCount int, maxPending int, metricSocketPath string) (err error) {
	if fm != nil {
		return fmt.Errorf("mbpu manager is already initialized")
	}

	if mbpuCount < 1 {
		return fmt.Errorf("mbpuCount must larger than or equal to 1")
	}

	lock.Lock()
	defer lock.Unlock()

	var wg sync.WaitGroup
	fm = &mbpuManager{
		make(chan requestWrapper),
		&wg,
	}

	for i := 0; i < mbpuCount; i++ {
		mpk, err := newMediumpk(i, maxPending, metricSocketPath)
		if err != nil {
			return err
		}
		var available int32 = int32(maxPending)
		chPoll := make(chan bool, maxPending)
		chPendable := make(chan bool)

		wg.Add(1)
		chEmergency := runPushing(mpk, chPoll, chPendable, &available)
		runPolling(mpk, chPoll, chPendable, chEmergency, &available)
	}

	log.Println("MBPUManager Initialized...")
	log.Printf("MBPUCount: %d  MAXPENDING : %d \n", mbpuCount, maxPending)

	return err
}

// CloseMBPUManager closes MBPU Device and stops goroutines for request/response to/from MBPU
func CloseMBPUManager() error {
	lock.Lock()
	defer lock.Unlock()

	close(fm.chanRequest)
	log.Println("MBPUManager request channel closed")

	fm.wg.Wait()
	fm = nil
	log.Println("MBPUManager Closed")
	return nil
}

// Request send RequestEnvelop to push-goroutine with channel for receive response
func Request(env RequestEnvelop) (int, []byte, []byte) {
	respChan := make(chan ResponseEnvelop, 1)
	req := requestWrapper{
		env,
		respChan,
	}

	fm.chanRequest <- req
	respEnv, ok := <-respChan
	if !ok {
		return -1, []byte(nil), []byte(nil)
	}

	close(respChan)
	r, s := respEnv.Signature()
	return respEnv.Result(), r, s
}

func runPushing(mpk *Mediumpk, chPoll chan bool, chPendable chan bool, available *int32) chan bool {
	stop := false

	chEmergency := make(chan bool)
	mpk.startMetric()
	go func() {
		for !stop {
			select {
			case <-chEmergency:
				// mbpu is down
				mpk.clearChanStore()
				go runEmergency()
				stop = true
				continue
			case req, ok := <-fm.chanRequest:
				if !ok {
					// terminate this loop by CloseMBPUManager
					stop = true
					continue
				}
				if atomic.LoadInt32(available) == 0 {
					chPendable <- true
					<-chPendable
				}
				chPoll <- true

				for {
					idx, err := mpk.request(&req.respChan, req.env)
					if err == nil { // good to go
						atomic.AddInt32(available, -1)
						break
					}
					// check error type
					if idx == -1 { // maxPending refuse error... try again
						log.Println(err.Error() + ", try again..")
						continue
					} else { // something has gone wrong
						chEmergency <- true
						fmt.Println(err)
						break
					}
				}
			}
		}
		close(chPoll)
		close(chEmergency)
		err := mpk.stopMetric()
		if err != nil {
			log.Println(err.Error())
		}

		err = mpk.close()
		if err != nil {
			log.Println(err.Error())
		}

		fm.wg.Done()
	}()
	return chEmergency
}

func runPolling(mpk *Mediumpk, chPoll <-chan bool, chPendable chan bool, chEmergency chan bool, available *int32) {
	go func() {
		stop := false

		for !stop {
			_, ok := <-chPoll
			if !ok { // terminate this loop by CloseMBPUManager
				stop = true
				continue
			}
			err := mpk.getResponseAndNotify()
			if err != nil {
				log.Printf("emergency from polling %d\n ", *available)
				chEmergency <- true
				stop = true
				log.Println(err.Error())
				continue
			}

			select {
			case <-chPendable:
				atomic.AddInt32(available, 1)
				chPendable <- true
			default:
				atomic.AddInt32(available, 1)
			}
		}
		close(chPendable)
	}()
}

func runEmergency() {
	stop := false
	log.Println("emergency...!!")
	for !stop {
		req, ok := <-fm.chanRequest
		if !ok { // terminate this loop by CloseMBPUManager
			stop = true
			continue
		}
		close(req.respChan)
	}
}
