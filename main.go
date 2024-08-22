package main

import (
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/factory"
	"github.com/allape/openkvm/kvm"
	"github.com/allape/openkvm/kvm/button"
	"github.com/allape/openkvm/logger"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"syscall"
	"time"
)

var log = logger.New("[main]")

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatalln("get config:", err)
	}

	k, err := factory.KeyboardFromConfig(conf)
	if err != nil {
		log.Fatalln("keyboard from config:", err)
	}
	defer func() {
		if k != nil {
			_ = k.Close()
		}
	}()

	v, err := factory.VideoFromConfig(conf)
	if err != nil {
		log.Fatalln("video from config:", err)
	}
	defer func() {
		if v != nil {
			_ = v.Close()
		}
	}()

	m, err := factory.MouseFromConfigOrUseKeyboard(k, conf)
	if err != nil {
		log.Fatalln("mouse from config or use keyboard:", err)
	}
	defer func() {
		if m != nil {
			_ = m.Close()
		}
	}()

	b, err := factory.ButtonFromConfig(conf, k, m)
	if err != nil {
		log.Fatalln("button from config:", err)
	}
	defer func() {
		if b != nil {
			_ = b.Close()
		}
	}()

	videoCodec, err := factory.VideoCodecFromConfig(conf)
	if err != nil {
		log.Fatalln("video codec from config:", err)
	}

	server, err := kvm.New(k, v, m, videoCodec, kvm.Options{
		Config: conf,
	})
	if err != nil {
		log.Fatalln("new kvm:", err)
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
			log.Println("upgrade:", err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		err = server.HandleClient(Websockets2KVMClient(conn))
		if err != nil {
			log.Println("handle client:", err)
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

	http.HandleFunc("/button", func(writer http.ResponseWriter, request *http.Request) {
		if b == nil {
			writer.WriteHeader(http.StatusNotImplemented)
			_, _ = writer.Write([]byte("not implemented"))
			return
		}

		validTypes := []button.Type{button.PowerButton, button.ResetButton, button.ExtraButton}

		query := request.URL.Query()

		t := query.Get("type")
		if !slices.Contains(validTypes, button.Type(t)) {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("button type not supported"))
			return
		}

		msStr := query.Get("ms")
		ms, err := strconv.Atoi(msStr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("invalid duration"))
			return
		}
		dur := time.Duration(ms) * time.Millisecond

		err = b.Press(button.Type(t))
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("press button: " + err.Error()))
			return
		}

		time.Sleep(dur)

		err = b.Release(button.Type(t))
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("release button: " + err.Error()))
			return
		}

		writer.Header().Add("Content-Type", "text/plain")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})

	if conf.VNC.Path != "" {
		http.HandleFunc("/", http.FileServer(http.Dir(conf.VNC.Path)).ServeHTTP)
	}

	go func() {
		log.Fatalln(http.ListenAndServe(conf.Websocket.Addr, nil))
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Println("started")
	sig := <-sigs
	log.Println("exiting with", sig)
}
