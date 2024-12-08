package main

import (
	"github.com/allape/openkvm/kvm"
	"github.com/gorilla/websocket"
)

type WebsocketsKVMClient struct {
	Conn *websocket.Conn
}

func (w *WebsocketsKVMClient) Read(dst []byte) (int, error) {
	_, src, err := w.Conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	n := copy(dst, src)
	return n, nil
}

func (w *WebsocketsKVMClient) Write(src []byte) (int, error) {
	err := w.Conn.WriteMessage(websocket.BinaryMessage, src)
	if err != nil {
		return 0, err
	}
	return len(src), nil
}

func (w *WebsocketsKVMClient) Close() error {
	return w.Conn.Close()
}

func Websockets2KVMClient(conn *websocket.Conn) *kvm.Client {
	return kvm.NewClient(&WebsocketsKVMClient{Conn: conn})
}
