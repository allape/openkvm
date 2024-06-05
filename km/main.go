package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"github.com/allape/openkvm/logger"
	"go.bug.st/serial"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// https://datatracker.ietf.org/doc/html/rfc6143#section-7.5.4
// left ctrl + c
// 0x0401 0000 0000 ffe3
// 0x0401 0000 0000 ff63
// 0x0400 0000 0000 ffe3
// 0x0400 0000 0000 ff63

// https://datatracker.ietf.org/doc/html/rfc6143#section-7.5.5
// mouse left click at 120, 30
// 0b00000101 10000000 00000000 01100100 00000000 00011110

var log = logger.New("[serial]")

const Name = "/dev/ttyACM0"

func main() {
	mode := &serial.Mode{
		BaudRate: 9600,
	}
	port, err := serial.Open(Name, mode)
	if err != nil {
		log.Fatalln("unable to open port:", err)
	}

	go func(port serial.Port) {
		buf := make([]byte, 1024)
		for {
			n, err := port.Read(buf)
			if err != nil {
				log.Fatalln("read error:", err)
			}
			if n == 0 {
				log.Println("EOF")
				return
			}
			print(string(buf[:n]))
		}
	}(port)

	go func(s serial.Port) {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalln("fail to read from stdin:", err)
			}

			text = strings.TrimSpace(text)
			log.Println(">", text)

			var raw []byte

			if strings.HasPrefix(text, "0x") {
				text = strings.ReplaceAll(text, " ", "")
				raw, err = hex.DecodeString(text[2:])
				if err != nil {
					log.Println("invalid hex string:", text)
					continue
				}
			} else if strings.HasPrefix(text, "0b") {
				text = strings.ReplaceAll(text, " ", "")
				raw, err = BitsString2Bytes(text[2:])
				if err != nil {
					log.Println(err, text)
					continue
				}
			} else {
				raw = []byte(strings.TrimSpace(text))
			}

			log.Println("> 0x", hex.EncodeToString(raw))

			_, err = s.Write(raw)
			if err != nil {
				log.Fatalln("write error:", err)
			}
			err = s.Drain()
			if err != nil {
				log.Fatalln("flush error:", err)
			}
		}
	}(port)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Println("awaiting signal")
	sig := <-sigs
	log.Println("exiting with", sig)

	_ = port.Close()
}

func BitsString2Bytes(bitsStr string) ([]byte, error) {
	bits := []byte(bitsStr)
	if len(bits)%8 != 0 {
		return nil, errors.New("invalid binary string")
	}
	bs := make([]byte, len(bits)/8)
	for i := 0; i < len(bits); i++ {
		byteIndex := i / 8
		if bits[i] == '1' {
			bs[byteIndex] = bs[byteIndex]<<1 | 1
		} else {
			bs[byteIndex] = bs[byteIndex] << 1
		}
	}
	return bs, nil
}
