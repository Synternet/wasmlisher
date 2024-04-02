package wasmlisher

import (
	"context"
	"errors"
	dlsdkOptions "github.com/syntropynet/data-layer-sdk/pkg/options"
	dlsdk "github.com/syntropynet/data-layer-sdk/pkg/service"
	"log"
	"time"
)

type Wasmlisher struct {
	Publisher   *dlsdk.Service
	config      string
	cfInterval  int
	streams     []StreamConf
	msgChannels map[string]chan []byte
	active      bool
}

func New(publisherOptions []dlsdkOptions.Option, config string, configInterval int) *Wasmlisher {
	ret := &Wasmlisher{
		Publisher:   &dlsdk.Service{},
		config:      config,
		msgChannels: make(map[string]chan []byte),
		cfInterval:  configInterval,
		active:      true,
	}

	ret.Publisher.Configure(publisherOptions...)

	return ret
}

func (w *Wasmlisher) loadAndApplyConfig() {

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
			w.subscribeToStream(stream)
		}
	}

	w.streams = newStreams
}

func (w *Wasmlisher) reloadConfigPeriodically() {
	for w.active {
		w.loadAndApplyConfig()
		time.Sleep(time.Duration(w.cfInterval) * time.Second)
	}
}

func (w *Wasmlisher) subscribeToStream(stream StreamConf) {
	msgChannel := make(chan []byte)

	w.msgChannels[stream.InputStream] = msgChannel
	_, err := w.Publisher.SubscribeTo(w.handlerInputStreamFactory(stream.InputStream), stream.InputStream)
	if err != nil {
		log.Printf("Error subscribing to stream: %v\n", err)
		return
	}
	go w.RunWasmStream(stream.File, msgChannel, stream.OutputStream, stream.Env)
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

	return w.Publisher.Start()
}

func (w *Wasmlisher) Close() error {
	w.active = false
	for _, ch := range w.msgChannels {
		close(ch)
	}
	w.msgChannels = make(map[string]chan []byte) // Reset msgChannels to clean up

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
