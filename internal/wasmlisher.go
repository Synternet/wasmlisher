package wasmlisher

import (
	"context"
	"errors"
	"fmt"
	dlsdkOptions "github.com/synternet/data-layer-sdk/pkg/options"
	dlsdk "github.com/synternet/data-layer-sdk/pkg/service"
	"io"
	"log"
	"net"
	"os"
	"strconv"
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
	msgChannel := make(chan []byte, 100)
	w.msgChannels[stream.InputStream] = msgChannel

	switch stream.InputType {
	case "nats":
		_, err := w.Publisher.SubscribeTo(w.handlerInputStreamFactory(stream.InputStream), stream.InputStream)
		if err != nil {
			log.Printf("Error subscribing to NATS stream: %v\n", err)
			return
		}
	case "unix_socket":
		if err := w.createAndHandleUnixSocket(stream.InputStream, msgChannel); err != nil {
			log.Printf("Error setting up Unix socket %s: %v", stream.InputStream, err)
			return
		}
	default:
		log.Printf("Unsupported input type: %s\n", stream.InputType)
		return
	}

	go w.RunWasmStream(stream.LocalPath, msgChannel, stream.OutputStream, stream.Env)
}

func (w *Wasmlisher) createAndHandleUnixSocket(socketPath string, msgChannel chan []byte) error {
	// Remove existing socket if present
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing Unix socket %s: %w", socketPath, err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("error listening on Unix socket %s: %v", socketPath, err)
	}

	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Error accepting Unix socket connection: %v", err)
				continue
			}
			go w.handleUnixSocketConnection(conn, msgChannel)
		}
	}()
	return nil
}

func (w *Wasmlisher) handleUnixSocketConnection(conn net.Conn, msgChannel chan []byte) {
	defer conn.Close()
	const lengthPrefixSize = 10

	for {
		// Read length prefix
		lengthPrefix := make([]byte, lengthPrefixSize)
		_, err := io.ReadFull(conn, lengthPrefix)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading length prefix from Unix socket: %v", err)
			}
			break
		}

		// Parse the length
		messageLength, err := strconv.Atoi(string(lengthPrefix))
		if err != nil {
			log.Printf("Invalid length prefix: %v", err)
			break
		}

		// Read the actual message
		message := make([]byte, messageLength)
		_, err = io.ReadFull(conn, message)
		if err != nil {
			log.Printf("Error reading message from Unix socket: %v", err)
			break
		}

		msgChannel <- message
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
