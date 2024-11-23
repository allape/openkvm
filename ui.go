package main

import (
	_ "embed"
	"github.com/allape/openkvm/config"
	"net/http"
	"os"
)

const (
	ButtonHTMLPath = "./ui/button.html"
)

//go:embed ui/button.html
var ButtonHTML string

func SetupUI(conf *config.Config) {
	http.HandleFunc("/ui/button.html", func(writer http.ResponseWriter, request *http.Request) {
		if stat, err := os.Stat(ButtonHTMLPath); err == nil && !stat.IsDir() {
			http.ServeFile(writer, request, ButtonHTMLPath)
		} else {
			writer.Header().Add("Content-Type", "text/html, charset=utf-8")
			writer.WriteHeader(http.StatusOK)
			_, err := writer.Write([]byte(ButtonHTML))
			if err != nil {
				log.Println("response button.html error:", err)
			}
		}
	})

	if conf.VNC.Path != "" {
		http.HandleFunc("/", http.FileServer(http.Dir(conf.VNC.Path)).ServeHTTP)
	}
}
