# This file is written in TOML format
# See https://toml.io/en/

# `ext` argument is written in Golang struct tag format
# See https://go.dev/ref/spec#Tag

[websocket]
# `127.0.0.1:8080` listen for 127.0.0.1 only
# `:8080` can be accessed by any device
addr = ":8080"
# VNC websockets path, this is the default value for noVNC.
path = "/websockify"
# CORS only apply for websocket connection only, not all HTTP requests.
cors = false

[vnc]
# Path to a static served folder, noVNC is recommended.
#path = "../noVNC"

[keyboard]
# none, serialport
type = "serialport"
src = "/dev/ttyACM0"
ext = "baud:\"115200\""

[video]
# A command run before video capture.
#prelude_command = "shell:\"bash\" args:\"-c\" cmd:\"usbreset Hagibis\""
# `usb` only
type = "usb"
# Video device index when using `usb` as driver.
# Either the index of video device or the path to video device.
src = "0"
width = 1270
height = 720
frame_rate = 30.0
# min 0, max 100
quality = 50
# horizontal(0), vertical(1), both(-1), off(-2)
flip_code = -2
# `4` will slice frame into a 4x4 grid, total 16 pieces
slice_count = 4
ext = ""

[mouse]
# none, serialport
type = "serialport"
src = "/dev/ttyACM0"
ext = "baud:\"115200\""
# A factor to adjust the cursor move distance when the video is scaled.
# Suggested scale for (1920, 1080):
#   (1920, 1080):
#     cursor_x_scale = 17.00
#     cursor_x_scale = 28.78
#   (1270, 720):
#     cursor_x_scale = 25.6632
#     cursor_y_scale = 45.5754925372
cursor_x_scale = 25.6632
cursor_y_scale = 45.5754925372
