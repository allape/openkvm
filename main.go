package main

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/kvm"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const Tag = "[main]"

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatalln(Tag, "get config:", err)
	}

	k, err := config.KeyboardFromConfig(conf)
	if err != nil {
		log.Fatalln(Tag, "keyboard from config:", err)
	}
	defer func() {
		if k != nil {
			_ = k.Close()
		}
	}()

	v, err := config.VideoFromConfig(conf)
	if err != nil {
		log.Fatalln(Tag, "video from config:", err)
	}
	defer func() {
		if v != nil {
			_ = v.Close()
		}
	}()

	m, err := config.MouseFromConfigOrUseKeyboard(k, conf)
	if err != nil {
		log.Fatalln(Tag, "mouse from config or use keyboard:", err)
	}
	defer func() {
		if m != nil && m != k {
			_ = m.Close()
		}
	}()

	videoCodec, err := config.VideoCodecFromConfig(conf)
	if err != nil {
		log.Fatalln(Tag, "video codec from config:", err)
	}

	server, err := kvm.New(k, v, m, videoCodec, kvm.Options{
		SliceCount: conf.Video.SliceCount,
	})
	if err != nil {
		log.Fatalln(Tag, "new kvm:", err)
	}

	upgrader := websocket.Upgrader{}

	if conf.Websocket.Cors {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	}

	http.HandleFunc(conf.Websocket.Path, func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			log.Println(Tag, "upgrade:", err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		err = server.HandleClient(Websockets2KVMClient(conn))
		if err != nil {
			log.Println(Tag, "handle client:", err)
		}
	})

	http.HandleFunc("/led", func(writer http.ResponseWriter, request *http.Request) {
		state := request.URL.Query().Get("state")
		if state == "on" {
			_ = k.SendPointerEvent([]byte{'a', '1'})
		} else {
			_ = k.SendPointerEvent([]byte{'a', '0'})
		}
		writer.Header().Add("Content-Type", "text/plain")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})

	if conf.VNC.Path != "" {
		http.HandleFunc("/", http.FileServer(http.Dir(conf.VNC.Path)).ServeHTTP)
	}

	go func() {
		log.Fatalln(Tag, http.ListenAndServe(conf.Websocket.Addr, nil))
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Println(Tag, "started")
	sig := <-sigs
	log.Println(Tag, "exiting with", sig)
}
