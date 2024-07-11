package wasmlisher

import (
	"context"
	"errors"
	dlsdkOptions "github.com/synternet/data-layer-sdk/pkg/options"
	dlsdk "github.com/synternet/data-layer-sdk/pkg/service"
	"io"
	"log"
	"net"
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

	newStreams, err := LoadConfig(w.config, w.streams)
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

	switch stream.InputType {
	case "nats":
		_, err := w.Publisher.SubscribeTo(w.handlerInputStreamFactory(stream.InputStream), stream.InputStream)
		if err != nil {
			log.Printf("Error subscribing to NATS stream: %v\n", err)
			return
		}
	case "unix_socket":
		go w.handleUnixSocket(stream.InputStream, msgChannel)
	default:
		log.Printf("Unsupported input type: %s\n", stream.InputType)
		return
	}

	go w.RunWasmStream(stream.LocalPath, msgChannel, stream.OutputStream, stream.Env)
}

func (w *Wasmlisher) handleUnixSocket(socketPath string, msgChannel chan []byte) {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Printf("Error listening on Unix socket %s: %v", socketPath, err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting Unix socket connection: %v", err)
			continue
		}

		go w.handleUnixSocketConnection(conn, msgChannel)
	}
}

func (w *Wasmlisher) handleUnixSocketConnection(conn net.Conn, msgChannel chan []byte) {
	defer conn.Close()
	buffer := make([]byte, 16384)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from Unix socket: %v", err)
			}
			break
		}
		msgChannel <- buffer[:n]
	}
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
