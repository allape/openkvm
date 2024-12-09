package main

import (
	_ "embed"
	"github.com/allape/gogger"
	"github.com/allape/openkvm/config"
	"github.com/allape/openkvm/factory"
	"github.com/allape/openkvm/kvm"
	"github.com/allape/openkvm/kvm/button"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"os/signal"
	"path"
	"slices"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

var l = gogger.New("main")

const (
	ButtonHTMLPath = "ui/button.html"
)

var ValidTypes = []button.Type{
	button.PowerButton,
	button.ResetButton,
	button.ExtraButton,
}

//go:embed ui/button.html
var ButtonHTML string

func main() {
	err := gogger.InitFromEnv()
	if err != nil {
		l.Error().Fatalln("init logger:", err)
	}

	conf, err := config.GetConfig()
	if err != nil {
		l.Error().Fatalln("get config:", err)
	}

	k, err := factory.KeyboardFromConfig(conf)
	if err != nil {
		l.Error().Fatalln("keyboard from config:", err)
	}
	defer func() {
		if k != nil {
			_ = k.Close()
		}
	}()

	v, err := factory.VideoFromConfig(conf)
	if err != nil {
		l.Error().Fatalln("video from config:", err)
	}
	defer func() {
		if v != nil {
			_ = v.Close()
		}
	}()

	m, err := factory.MouseFromConfigOrUseKeyboard(k, conf)
	if err != nil {
		l.Error().Fatalln("mouse from config or use keyboard:", err)
	}
	defer func() {
		if m != nil {
			_ = m.Close()
		}
	}()

	b, err := factory.ButtonFromConfig(conf, k, m)
	if err != nil {
		l.Error().Fatalln("button from config:", err)
	}
	defer func() {
		if b != nil {
			_ = b.Close()
		}
	}()

	videoCodec, err := factory.VideoCodecFromConfig(conf)
	if err != nil {
		l.Error().Fatalln("video codec from config:", err)
	}

	server, err := kvm.New(k, v, m, videoCodec, kvm.Options{
		Config: conf,
	})
	if err != nil {
		l.Error().Fatalln("new kvm:", err)
	}

	upgrader := websocket.Upgrader{}

	engine := gin.Default()

	if conf.Websocket.Cors {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		engine.Use(cors.Default())
	}

	clientCount := atomic.Int64{}

	handleWebsocket := func(context *gin.Context) {
		conn, err := upgrader.Upgrade(context.Writer, context.Request, nil)
		if err != nil {
			l.Error().Println("upgrade:", err)
			context.String(http.StatusInternalServerError, "Internal Server Error: upgrader")
			return
		}

		defer func() {
			_ = conn.Close()
			l.Debug().Println("client disconnected")
			if clientCount.Add(-1) == 0 {
				l.Info().Println("no client left, closing video")
				_ = v.Close()
			}
		}()

		l.Debug().Println("client connected")

		clientCount.Add(1)
		err = v.Open()
		if err != nil {
			l.Error().Println("open video:", err)
			return
		}
		err = server.HandleClient(Websockets2KVMClient(conn))
		if err != nil {
			l.Warn().Println("handle client:", err)
		}
	}

	basicAuth := gin.BasicAuth(gin.Accounts{
		conf.VNC.Username: conf.VNC.Password,
	})

	engine.GET(conf.Websocket.Path, handleWebsocket)

	apiGroup := engine.Group("/api", basicAuth)

	apiGroup.GET("/led", func(context *gin.Context) {
		state := context.Query("state")
		if state == "on" {
			_ = k.SendPointerEvent([]byte{'a', '1'})
		} else {
			_ = k.SendPointerEvent([]byte{'a', '0'})
		}

		context.String(http.StatusOK, "ok")
	})

	apiGroup.GET("/button", func(context *gin.Context) {
		if b == nil {
			context.String(http.StatusNotImplemented, "not implemented")
			return
		}

		t := context.Query("type")
		if !slices.Contains(ValidTypes, button.Type(t)) {
			context.String(http.StatusBadRequest, "button type not supported")
			return
		}

		msStr := context.Query("ms")
		if msStr == "" {
			context.String(http.StatusBadRequest, "missing duration")
			return
		}
		ms, err := strconv.Atoi(msStr)
		if err != nil {
			context.String(http.StatusBadRequest, "invalid duration")
			return
		}

		dur := time.Duration(ms) * time.Millisecond

		err = b.Press(button.Type(t))
		if err != nil {
			context.String(http.StatusInternalServerError, "press button: %s", err.Error())
			return
		}

		time.Sleep(dur)

		err = b.Release(button.Type(t))
		if err != nil {
			context.String(http.StatusInternalServerError, "release button: %s", err.Error())
			return
		}

		context.String(http.StatusOK, "ok")
	})

	if conf.VNC.Path != "" {
		handleVNCIndex := func(context *gin.Context) {
			context.Redirect(http.StatusMovedPermanently, "/vnc/vnc.html")
		}
		engine.GET("/", handleVNCIndex)
		engine.GET("/vnc.html", handleVNCIndex)

		handleButtonHTML := func(context *gin.Context) {
			if stat, err := os.Stat(ButtonHTMLPath); err == nil && !stat.IsDir() {
				context.File(ButtonHTMLPath)
			} else {
				context.Data(http.StatusOK, "text/html; charset=utf-8", []byte(ButtonHTML))
			}
		}

		vncGroup := engine.Group("/vnc", basicAuth)
		vncGroup.GET("/*filename", func(context *gin.Context) {
			filename := context.Param("filename")

			switch {
			case filename == "/button.html":
				handleButtonHTML(context)
				return
			case filename == conf.Websocket.Path:
				handleWebsocket(context)
			default:
				context.File(path.Join(conf.VNC.Path, filename))
			}
		})
	}

	go func() {
		l.Error().Fatalln(engine.Run(conf.Websocket.Addr))
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	l.Info().Println("started")

	sig := <-sigs
	l.Info().Println("exiting with", sig)
}
