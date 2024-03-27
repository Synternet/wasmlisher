package wasmlisher

import (
	"context"
	"errors"
	dlsdkOptions "github.com/syntropynet/data-layer-sdk/pkg/options"
	dlsdk "github.com/syntropynet/data-layer-sdk/pkg/service"
	"log"
	"sync"
	"time"
)

type Wasmlisher struct {
	Publisher   *dlsdk.Service
	config      string
	streams     []StreamConf
	msgChannels map[string]chan []byte
	active      bool
	mutex       sync.Mutex
}

func New(publisherOptions []dlsdkOptions.Option, config string) *Wasmlisher {
	ret := &Wasmlisher{
		Publisher:   &dlsdk.Service{},
		config:      config,
		msgChannels: make(map[string]chan []byte),
		active:      true,
	}

	ret.Publisher.Configure(publisherOptions...)

	return ret
}

func (w *Wasmlisher) loadAndApplyConfig() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	newStreams, err := LoadConfig(w.config)
	if err != nil {
		log.Printf("Error loading config: %v\n", err)
		return
	}

	newStreamMap := make(map[string]StreamConf)
	for _, stream := range newStreams {
		newStreamMap[stream.InputStream] = stream
	}

	for input, ch := range w.msgChannels {
		if _, exists := newStreamMap[input]; !exists {
			close(ch)
			delete(w.msgChannels, input)
		}
	}

	for _, stream := range newStreams {
		if _, exists := w.msgChannels[stream.InputStream]; !exists {
			w.subscribeToStream(stream.InputStream, stream.OutputStream, stream.File)
		}
	}

	w.streams = newStreams
}

func (w *Wasmlisher) reloadConfigPeriodically() {
	for w.active {
		w.loadAndApplyConfig()
		time.Sleep(5 * time.Minute)
	}
}

func (w *Wasmlisher) subscribeToStream(inputSubject, outputSubject, wasmFilePath string) {
	msgChannel := make(chan []byte)

	w.mutex.Lock()
	w.msgChannels[inputSubject] = msgChannel
	w.mutex.Unlock()

	_, err := w.Publisher.SubscribeTo(w.handlerInputStreamFactory(inputSubject), inputSubject)
	if err != nil {
		log.Printf("Error subscribing to stream: %v\n", err)
		return
	}

	go w.RunWasmStream(wasmFilePath, msgChannel, outputSubject)
}

// Factory function to create a handler function bound to a specific stream's channel
func (w *Wasmlisher) handlerInputStreamFactory(streamSubject string) func(dlsdk.Message) {
	return func(msg dlsdk.Message) {
		if msgChannel, ok := w.msgChannels[streamSubject]; ok {
			msgChannel <- msg.Data()
		}
	}
}

func (w *Wasmlisher) Start() context.Context {
	go w.reloadConfigPeriodically()

	go func() {
		w.loadAndApplyConfig() // Initial load
	}()

	return w.Publisher.Start()
}

func (w *Wasmlisher) Close() error {
	w.mutex.Lock()
	w.active = false
	for _, ch := range w.msgChannels {
		close(ch)
	}
	w.msgChannels = make(map[string]chan []byte) // Reset msgChannels to clean up
	w.mutex.Unlock()

	log.Println("Wasmlisher.Close")
	w.Publisher.Cancel(nil)

	var err []error

	log.Println("Waiting on Wasmlisher publisher group")
	errGr := w.Publisher.Group.Wait()
	if !errors.Is(errGr, context.Canceled) {
		err = append(err, errGr)
	}

	log.Println("Wasmlisher.Close DONE")
	return errors.Join(err...)
}
