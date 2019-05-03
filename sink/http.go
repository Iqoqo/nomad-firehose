package sink

import (
	"strconv"
	"time"
	"bytes"

	"net/http"

	"os"

	"fmt"

	log "github.com/sirupsen/logrus"
)

// HttpSink ...
type HttpSink struct {
	address     string
	workerCount int
	stopCh      chan interface{}
	putCh       chan []byte
}

// NewHttp ...
func NewHttp() (*HttpSink, error) {
	address := os.Getenv("SINK_HTTP_ADDRESS")
	if address == "" {
		return nil, fmt.Errorf("[sink/http] Missing SINK_HTTP_ADDRESS (example: http://miau.com:8080/biau)")
	}

	workerCountStr := os.Getenv("SINK_WORKER_COUNT")
	if workerCountStr == "" {
		workerCountStr = "1"
	}
	workerCount, err := strconv.Atoi(workerCountStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid SINK_WORKER_COUNT, must be an integer")
	}

	return &HttpSink{
		address:     address,
		workerCount: workerCount,
		stopCh:      make(chan interface{}),
		putCh:       make(chan []byte, 10000),
	}, nil
}

// Start ...
func (s *HttpSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	for i := 0; i < s.workerCount; i++ {
		go s.send(i)
	}

	// wait forever for a stop signal to happen
	for {
		select {
		case <-s.stopCh:
			break
		}
		break
	}

	return nil
}

// Stop ...
func (s *HttpSink) Stop() {
	log.Infof("[sink/http] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Info("[sink/http] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

// Put ..
func (s *HttpSink) Put(data []byte) error {
	s.putCh <- data

	return nil
}

func (s *HttpSink) send(id int) {
	log.Infof("[sink/http/%d] Starting writer", id)

	for {
		select {
		case data := <-s.putCh:
			buffer := bytes.NewBuffer(data[:])
			log.Debugf("[sink/http/%d] publishing to %s %v", id, s.address, buffer)
			_, err := http.Post(s.address, "application/json; charset=utf-8", buffer)
			if err != nil {
				log.Errorf("[sink/http/%d] %s", id, err)
			} else {
				log.Debugf("[sink/http/%d] publish ok(%d messages left)", id, len(s.putCh))
			}
		}
	}
}

// Name ..
func (s *HttpSink) Name() string {
	return "http"
}
