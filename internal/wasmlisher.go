package wasmlisher

import (
	"context"
	"errors"
	dlsdkOptions "github.com/syntropynet/data-layer-sdk/pkg/options"
	dlsdk "github.com/syntropynet/data-layer-sdk/pkg/service"
	"log"
)

type Wasmlisher struct {
	Publisher   *dlsdk.Service
	streams     []StreamConf
	msgChannels map[string]chan []byte
}

func New(publisherOptions []dlsdkOptions.Option, config string) *Wasmlisher {

	ret := &Wasmlisher{
		Publisher: &dlsdk.Service{},
	}

	ret.Publisher.Configure(publisherOptions...)

	ret.streams, _ = LoadConfig(config)

	return ret
}

func (w *Wasmlisher) handlerInputStream(msg dlsdk.Message) {
	println(msg.Data())
}

// Factory function to create a handler function bound to a specific stream's channel
func (w *Wasmlisher) handlerInputStreamFactory(streamSubject string) func(dlsdk.Message) {
	return func(msg dlsdk.Message) {
		if msgChannel, ok := w.msgChannels[streamSubject]; ok {
			msgChannel <- msg.Data()
		}
	}
}

func (w *Wasmlisher) SubscribeProcessEmit(inputSubject string, outputSubject string, wasmFilePath string) error {
	if w.msgChannels == nil {
		w.msgChannels = make(map[string]chan []byte)
	}

	// Create a new channel for this stream
	msgChannel := make(chan []byte)
	w.msgChannels[inputSubject] = msgChannel

	// Subscribe to the stream
	_, err := w.Publisher.SubscribeTo(w.handlerInputStreamFactory(inputSubject), inputSubject)
	if err != nil {
		return err
	}

	// Start processing messages for this stream in a separate goroutine
	go w.RunWasmStream(wasmFilePath, msgChannel, outputSubject)

	return nil
}

func (w *Wasmlisher) Start() context.Context {
	go func() {
		for _, stream := range w.streams {
			err := w.SubscribeProcessEmit(stream.InputStream, stream.OutputStream, stream.File)
			if err != nil {
				panic(err)
			}
		}
	}()
	return w.Publisher.Start()
}

func (w *Wasmlisher) Close() error {
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
