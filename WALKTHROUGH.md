# Walk Through of my development

### Problem with HDMI Recorder

- ~~The common seen `HDMI Recorder` is a `USB 3.0` device.  
  There is only blank data from the `HDMI Recorder` on a `USB 2.0` port.  
  It seems that this is the problem of the `HDMI Recorder` itself,  
  the `HDMI Recorder` I bought can NOT work on Linux (Debian ARM or Ubuntu 24.04) even with a `USB 3.0` port.  
  Then, on the selection of SBC, beware of the support of `USB 3.0`.~~
- This should be the problem with HDMI recorder itself of the driver in Linux,  
  I am lack of the knowledge of the driver in Linux, but I might find a solution for it:
    - TL;DR: Reset the USB device before capturing the frame.
      ```shell
      # Display all USB device, make sure the HDMI recorder is in 480M or above
      lsusb -t
      # /:  Bus 01.Port 1: Dev 1, Class=root_hub, Driver=xhci-hcd/1p, 480M
      #     |__ Port 1: Dev 3, If 0, Class=Video, Driver=uvcvideo, 480M
      #     |__ Port 1: Dev 3, If 1, Class=Video, Driver=uvcvideo, 480M
      #     |__ Port 1: Dev 3, If 2, Class=Audio, Driver=snd-usb-audio, 480M
      #     |__ Port 1: Dev 3, If 3, Class=Audio, Driver=snd-usb-audio, 480M
      
      # Get the id of the USB device
      lsusb
      # Bus 001 Device 003: ID 1de1:f105 Actions Microelectronics Co. Hagibis
      
      # Reset the USB device
      # Replace this with the id or the name of your USB device. 
      # For me, this is `Hagibis` or `1de1:f105`
      usbreset Hagibis
      
      # Capture one frame from device
      v4l2-ctl --verbose \
      --device=/dev/video0 \
      --stream-mmap \
      --stream-count=1 \
      --stream-to=frame.jpg \
      --set-fmt-video="width=640,height=480,pixelformat=MJPG"
      
      # BUT! But, in size of 1920x1080, frame will be corrupted
      
      # Without resetting the USB device, there will be some error message in `dmesg` command
      # Non-zero status (-71) in video completion handler.
      
      # Here are some helpful commands
      # List all video devices
      v4l2-ctl --list-devices
      # List all supported formats for a video device
      v4l2-ctl --list-formats -d [device name or path or index] # v4l2-ctl --list-formats -d 0
      # List all supported frame sizes
      v4l2-ctl --list-framesizes [pixel format] -d [device] # v4l2-ctl --list-framesizes MJPEG -d 0
      ```

### PIO of ESP32

- [PIO](https://platformio.org/): Espressif has dropped the support for PIO in ESP-IDF v5.x. See git log to
  retrieve the code.
    - Open folder [km/esp32s3](./km/esp32s3) of this project in [VSCode](https://code.visualstudio.com/)
    - After installing [PlatformIO](https://marketplace.visualstudio.com/items?itemName=platformio.platformio-ide)
      extension
        - Go to `PlatformIO` Tab
        - `PROJECT TASKS` -> `Default` -> `General` -> `Upload`
        - Or open `command palette` and type `PlatformIO: Upload`
            - `⌘ + shift + p` on macOS to open command palette
            - `ctrl + shift + p` on Windows or Ubuntu

### OpenCV

OpenCV has been removed from this project, because it requires cgo to compile the entire project.
Using clt will be a better choice.
