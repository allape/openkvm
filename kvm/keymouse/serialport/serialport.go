package serialport

import (
	"errors"
	"github.com/allape/gogger"
	"github.com/allape/openkvm/kvm/keymouse"
	"go.bug.st/serial"
	"strings"
	"sync"
	"time"
)

var l = gogger.New("kvm.keymouse.serialport")

const MagicWord = "open-kvm"

type KeyboardMouseDriver struct {
	keymouse.Driver

	openLocker  sync.Locker
	writeLocker sync.Locker

	Port serial.Port

	Name string
	Baud int
}

func (d *KeyboardMouseDriver) Open() error {
	d.openLocker.Lock()
	defer d.openLocker.Unlock()

	if d.Port != nil {
		return errors.New("port already open")
	}

	mode := &serial.Mode{
		BaudRate: d.Baud,
	}
	port, err := serial.Open(d.Name, mode)
	if err != nil {
		return err
	}
	d.Port = port

	go func(port serial.Port) {
		buf := make([]byte, 1024)
		unfinishedLine := ""
		for {
			n, err := port.Read(buf)
			if err != nil {
				l.Error().Println("read error:", err)
			}
			if n == 0 {
				l.Warn().Println("EOF")
				return
			}
			lines := strings.Split(unfinishedLine+string(buf[:n]), "\n")
			for i := 0; i < len(lines)-1; i++ {
				l.Verbose().Println(">", lines[i])
			}
			unfinishedLine = lines[len(lines)-1]
		}
	}(port)

	_, err = port.Write([]byte(MagicWord))
	if err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	return nil
}

func (d *KeyboardMouseDriver) Close() error {
	d.openLocker.Lock()
	defer d.openLocker.Unlock()

	if d.Port == nil {
		return nil
	}

	err := d.Port.Close()
	d.Port = nil
	return err
}

func (d *KeyboardMouseDriver) Write(data []byte) (int, error) {
	err := d.Open()

	if d.Port == nil {
		return 0, err
	}

	d.writeLocker.Lock()
	defer d.writeLocker.Unlock()

	n, err := d.Port.Write(data)
	if err != nil {
		_ = d.Close()
		return n, err
	}

	return n, nil
}

func (d *KeyboardMouseDriver) SendKeyEvent(e keymouse.KeyEvent) error {
	_, err := d.Write(e)
	return err
}

func (d *KeyboardMouseDriver) SendPointerEvent(e keymouse.PointerEvent) error {
	_, err := d.Write(e)
	return err
}

func New(name string, baud int) keymouse.Driver {
	return &KeyboardMouseDriver{
		openLocker:  &sync.Mutex{},
		writeLocker: &sync.Mutex{},
		Name:        name,
		Baud:        baud,
	}
}
