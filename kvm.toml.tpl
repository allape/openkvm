# This file is written in TOML format
# See https://toml.io/en/

# `ext` argument is written in Golang struct tag format
# See https://go.dev/ref/spec#Tag

[websocket]
addr = ":8080"
path = "/websockify"
cors = true

[vnc]
# path to a static served folder, noVNC is recommended
path = "../noVNC"

[keyboard]
# none, serialport
type = "serialport"
src = "/dev/ttyACM0"
ext = "baud:\"115200\""

[video]
# `usb` only
type = "usb"
# video device index when using `usb` as driver
src = "0"
frame_rate = 30.0
# min 0, max 100
quality = 75
# horizontal(0), vertical(1), both(-1), off(-2)
flip_code = -2
# `4` will slice frame into a 4x4 grid, total 16 pieces
slice_count = 4
# phw = placeholder width
# phh = placeholder height
# placeholder will be used when no output frame from usb device
ext = "phw:\"1920\" phh:\"1080\""

[mouse]
# none, serialport
type = "serialport"
src = "/dev/ttyACM0"
ext = "baud:\"115200\""
