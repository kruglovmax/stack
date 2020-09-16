package out

import (
	"fmt"
	"io"
	"sync"

	"github.com/kruglovmax/stack/pkg/consts"
)

const (
	bufSize = 100
)

// Output type
type Output struct {
	enabled bool
	mux     *sync.Mutex
	output  io.Writer
}

// SendStringForObject func
func (output *Output) SendStringForObject(messagesChannel chan string, str string) {
	messagesChannel <- str
}

// StartOutputForObject func
func (output *Output) StartOutputForObject() (messagesChannel chan string, listenerChannel chan int) {
	messagesChannel = make(chan string, bufSize)
	listenerChannel = make(chan int)
	go output.runListener(messagesChannel, listenerChannel)
	return
}

// FinishOutputForObject func
func (output *Output) FinishOutputForObject(messagesChannel chan string, listenerChannel chan int) {
	defer output.mux.Unlock()
	close(messagesChannel)
	<-listenerChannel
	close(listenerChannel)
}

func (output *Output) runListener(messagesChannel chan string, listenerChannel chan int) {
	output.mux.Lock()
	for msg := range messagesChannel {
		fmt.Fprintln(output.output, msg)
	}
	listenerChannel <- consts.ExitCodeOK
}

// New Output
func New(ioWriter io.Writer) *Output {
	result := new(Output)
	result.output = ioWriter
	result.mux = new(sync.Mutex)
	return result
}
