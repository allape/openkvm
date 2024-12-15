# OpenKVM

DIY KVM device for remote-controlling a computer. `KVM` stands for `Keyboard & Video & Mouse`.

A side system on a rackmount server,
[`IPMI`](https://en.wikipedia.org/wiki/Intelligent_Platform_Management_Interface) in `Dell Server` for example.

Unlike the [IP-KVM](https://github.com/tiny-pilot/tinypilot), this project
respects [VNC protocol](https://datatracker.ietf.org/doc/html/rfc6143).

Just like the [MIT license](./LICENSE) says, no warranty or guarantee.

My dev walkthrough is in [WALKTHROUGH.md](./WALKTHROUGH.md).

**And do _NOT_ use for any illegal purposes.**

### TODO

- [x] Remove OpenCV, see [WALKTHROUGH.md](WALKTHROUGH.md#opencv) for the reason of removal
- [ ] Installation script
    - [ ] Register as a system service
    - [ ] Start on boot
- [x] VNC authentication
    - [x] DES encryption in Golang can NOT directly apply to [`VNC Authentication`](https://datatracker.ietf.org/doc/html/rfc6143#section-7.1.2)
    - [x] Http Basic Auth in web page and API
- [ ] More effective to calculate the difference between frames
    - Balance between the power of SBC and the network efficiency
    - Or achieve more support for noVNC, beyond [rfc6143](https://datatracker.ietf.org/doc/html/rfc6143)
- [ ] OTG as keyboard and mouse, see
  Linux [USB Gadget API](https://www.kernel.org/doc/html/v4.16/driver-api/usb/gadget.html)
- [x] Using a single command to get the frame for `Video`, like
  ```shell
  v4l2-ctl --device=/dev/video0 --stream-mmap --stream-count=1 --stream-to=- --set-fmt-video="width=640,height=480,pixelformat=MJPG"
  ```

## Dev Environment

### Hardware

**_NONE_ of them is sponsored, use them at your own risk!**

Essential hardware are:

- A computer that can run [Golang](https://go.dev/)
    - An old computer
    - SBC
        - [RaspberryPi](https://www.raspberrypi.com/)
        - [OrangePi](https://www.orangepi.org/)
        - [BTT-Pi](https://bigtree-tech.com/blogs/news/new-release-bigtreetech-btt-pi)
        - etc...
    - Maybe an Android phone?
- `HDMI Recorder`, see [WALKTHROUGH.md](WALKTHROUGH.md#problem-with-hdmi-recorder) for more details.
    - Or a `webcam` with an always-on monitor.
        - I know...this is a stupid way -- pointing a camera to a screen.
- Keyboard & Mouse Emulator
    - ESP32-S3
    - ESP32-S2
    - Arduino?
    - Some other device that supports USB HID output.
        - HID over BLE is not recommended, because it may not work in BIOS.
- Relay and/or delayed relay

#### My Gears

- `SBC`: [OrangePi 3 LTS](http://www.orangepi.cn/html/hardWare/computerAndMicrocontrollers/details/Orange-Pi-3-LTS.html)
    - `¥241 RMB` ≈ `$34 USD`
- `HDMI Recorder`: [hagibis UHC07](https://cn.hagibis.com/products-p00222p1.html)
    - `¥69 RMB` ≈ `$10 USD`
    - ⚠️ This device **ONLY** supports 1920x1080 and 3840x2160.
- `Keyboard & Mouse`: [ESP32-S3](https://docs.espressif.com/projects/esp-idf/en/latest/esp32s3/hw-reference/esp32s3/user-guide-devkitc-1.html)
    - I bought a third-party one of [WeAct ESP32 S3 (A) DevKitC 1](https://github.com/WeActStudio).
    - `¥52 RMB` ≈ `$7.5 USD`
- 5V relay * 2, 5V delayed relay * 1
    - `¥15 RMB` ≈ `$2 USD`

#### Others

- Test Device:
    - [RaspberryPi 2B](https://www.raspberrypi.com/products/raspberry-pi-2-model-b/)
    - A Windows computer
    - Ubuntu 24.04 x86_64
- Some SD cards.
- Some `USB Type-C2C/C2A` cables.
- A `HDMI` cable.
- Some power supplies.

It costs about `¥300 RMB` ≈ `40 USD`.

_**Price is for reference only, the actual price may vary.**_

### Software

- [GO](https://go.dev/)
- [Arduino](https://www.arduino.cc/)
- [noVNC](https://github.com/novnc/noVNC)

## Diagram

[飞书文档, FeiShu Doc](https://qi58or3rjjg.feishu.cn/wiki/KTZewFOx9iRyzQkfdzTcu8linxc?from=from_copylink)

![diagram.png](./docs/diagram.png)

## Installation

### Debian ARM64

1. Install GO dev kit
   ```shell
   sudo apt-get update
   sudo apt-get install -y wget curl ffmpeg v4l-utils
   GO_ZIP="go1.23.3.linux-arm64.tar.gz"
   wget "https://go.dev/dl/$GO_ZIP"
   sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf "$GO_ZIP"
   ```
2. Pull this repo
   ```shell
   sudo apt-get update
   sudo apt-get install -y git
   git clone --depth 1 https://github.com/allape/openkvm.git
   ```
3. Get noNVC
   ```shell
   git clone --depth 1 https://github.com/novnc/noVNC.git
   
   # [Optional] You can run noVNC separately
   python3 -m http.server --directory noVNC/ 8081
   ```
4. Flash ESP32-S3
    - [PIO](https://platformio.org/): See [WALKTHROUGH.md](WALKTHROUGH.md#pio-of-esp32) for the reason of removal
    - [Arduino](https://www.arduino.cc/)
        - Open `Perferences` -> `Additional Board Manager URLs` ->
          Add `https://espressif.github.io/arduino-esp32/package_esp32_dev_index.json`
            - Click [HERE](https://docs.espressif.com/projects/arduino-esp32/en/latest/installing.html#installing-using-arduino-ide) for more details
        - Open [km/esp32s3-arduino/main/main.ino](./km/esp32s3-arduino/main/main.ino)
          with [Arduino IDE](https://github.com/arduino/arduino-ide)
        - Select board `ESP32S3 Dev Module` and corresponding port
        - Click `Upload`
            - For deployed device, use [Arduino CLI](https://arduino.github.io/arduino-cli/1.1/installation/) to compile and upload firmware
            - Here is an example on `Debian` with an `ESP32-S3` connected to `/dev/ttyACM0`
              ```shell
              cd ~
              # command below will install `arduino-cli` at ~/bin
              curl -fsSL https://raw.githubusercontent.com/arduino/arduino-cli/master/install.sh | sh
              echo "export PATH=\$PATH:$HOME/bin" >> ./.bashrc
              source ./.bashrc
              arduino-cli config init
              arduino-cli config add board_manager.additional_urls https://espressif.github.io/arduino-esp32/package_esp32_dev_index.json
              arduino-cli config set network.proxy "http://localhost:1080" # optional, because arduino-cli may NOT respect http_proxy or https_proxy environment variables
              arduino-cli core update-index
              arduino-cli core install esp32:esp32 # this will takes a while...
              cd openkvm # change to the directory where the project located
              cd ./km/esp32s3-arduino/main/
              arduino-cli compile -b esp32:esp32:esp32s3 .
              arduino-cli upload . --fqbn esp32:esp32:esp32s3 -p /dev/ttyACM0 # change to your port
              ``` 
5. Run or build repo
   ```shell
   cd openkvm
   
   go mod download
   
   cp kvm.new.toml kvm.toml
   
   # Find out serial port
   dmesg | grep tty
   
   # Edit this file to apply your settings
   vim kvm.toml
   
   # Should run with super user privilege
   sudo go run .
   
   #go build -o openkvm .
   #sudo ./openkvm
   ```
6. Open browser and go to http://ip:8080/vnc.html, then click `Connect`
    - Hostname and port may vary depending on your settings
7. Open http://ip:8080/ui/button.html to control the relay

# Credits

- [Listed in go.mod](./go.mod)
- [Roboto Font](https://fonts.google.com/specimen/Roboto/about)
- [noVNC](https://github.com/novnc/noVNC)
- [ESP-IDF](https://docs.espressif.com/projects/esp-idf/en/latest/esp32s3/get-started/index.html)
- [ESP32-Arduino](https://docs.espressif.com/projects/arduino-esp32/en/latest/getting_started.html)
- [Raspberry Pi Imager](https://www.raspberrypi.com/software/)
- [KeyMap](https://github.com/qemu/keycodemapdb)
