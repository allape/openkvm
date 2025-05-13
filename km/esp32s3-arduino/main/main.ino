// Credits
// https://github.com/espressif/arduino-esp32/tree/master/libraries/USB/examples
// https://github.com/hathach/tinyusb

#include "Arduino.h"

#include "USB.h"
#include "USBHIDMouse.h"
#include "USBHIDKeyboard.h"
#include "USBMSC.h"

#include "keys.h"

#define BufferLength 1024

#define MagicWord "open-kvm"
#define MagicWordLength 8

#define KeyEvent 4      // https://datatracker.ietf.org/doc/html/rfc6143#section-7.5.4
#define PointerEvent 5  // https://datatracker.ietf.org/doc/html/rfc6143#section-7.5.5

// Button / Switch Event
//
// CMD  TYPE PIN  VALUE
// 0xff 0x01 0x0b 0x01
//
// CMD: fixed value "0xff"
// PIN: pin number
// TYPE:
//   0x01 for initialing
//     VALUE: 0x00 for input
//     VALUE: 0x01 for output
//   0x02 for set pin
//     VALUE: 0x00 for LOW
//     VALUE: 0x01 for HIGH
#define ButtonEvent 0xff  // power button, rest button, etc

// Clipboard Event
//
// Bewared, the "Clipboard" here is not the clipboard of an OS,
//     all content of "clipboard" will be writen into a file which is wrapped by an USB MSC
//
// CMD     LENGTH    DATA
// 0xfe    0x0001    0x00 0x01 0x02 ...
//
// CMD: fixed value "0xfe"
// LENGTH: length of the data, max value is {@link BufferLength} - 3
// DATA: the data array
#define ClipboardEvent 0xfe

// tips: `ctrl+a` to enter command mode if screen, `k` to kill
// screen /dev/cu.wchusbserialxxx 921600 \n
// open-kvm\n

// test the builtin LED
// a1: lights up
// a0: lights off
// aN: N is on or off
#define LEDTestEvent 'a'

// test keyboard
// b049: 1
// b032: Space
// b027: Esc
// b013: Enter
// bNNN: NNN is the key code, 000~255
#define KeyboardTestEvent 'b'

// test mouse
// c1000001-00002: click left button at (0001, 0002)
// c2000000000000: right
// c4000000000000: middle
// c0010000-10000: moving cursor to (100, 100)
// cNXXXXXXYYYYYY: N is button, XXXXXX is x axis(int16), YYYYYY is y axis(int16)
#define MouseTestEvent 'c'

// test button
// d1121: set pin 12 to output
// d2121: set pin 12 to HIGH
// d2120: set pin 12 to LOW
// dTNNX: T is type, NN is pin number, X is output/input or high/low
#define ButtonTestEvent 'd'

// test clipboard
// e1234: put text "1234" into clipboard
// eNNNN: NNNN is the text, max 4 bytes
#define ClipboardTestEvent 'e'

// https://developer.mozilla.org/en-US/docs/Web/API/MouseEvent/button#value
//     0: Main button pressed, usually the left button or the un-initialized state
//     1: Auxiliary button pressed, usually the wheel button or the middle button (if present)
//     2: Secondary button pressed, usually the right button
//     3: Fourth button, typically the Browser Back button
//     4: Fifth button, typically the Browser Forward button
// VNC emits `1 << MouseEvent.button` as button mask
#define _MOUSE_LEFT 1 << 0
#define _MOUSE_MIDDLE 1 << 1
#define _MOUSE_RIGHT 1 << 2
#define _MOUSE_WHEEL_UP 1 << 3
#define _MOUSE_WHEEL_DOWN 1 << 4
#define _MOUSE_WHEEL_LEFT 1 << 5
#define _MOUSE_WHEEL_RIGHT 1 << 6

#if !ARDUINO_USB_CDC_ON_BOOT
USBCDC USBSerial;
#endif
USBHIDKeyboard Keyboard;
USBHIDAbsoluteMouse Mouse;
USBMSC MSC;

// region MSC

#define FAT_U8(v)          ((v) & 0xFF)
#define FAT_U16(v)         FAT_U8(v), FAT_U8((v) >> 8)
#define FAT_U32(v)         FAT_U8(v), FAT_U8((v) >> 8), FAT_U8((v) >> 16), FAT_U8((v) >> 24)
#define FAT_MS2B(s, ms)    FAT_U8(((((s) & 0x1) * 1000) + (ms)) / 10)
#define FAT_HMS2B(h, m, s) FAT_U8(((s) >> 1) | (((m) & 0x7) << 5)), FAT_U8((((m) >> 3) & 0x7) | ((h) << 3))
#define FAT_YMD2B(y, m, d) FAT_U8(((d) & 0x1F) | (((m) & 0x7) << 5)), FAT_U8((((m) >> 3) & 0x1) | ((((y) - 1980) & 0x7F) << 1))
#define FAT_TBL2B(l, h)    FAT_U8(l), FAT_U8(((l >> 8) & 0xF) | ((h << 4) & 0xF0)), FAT_U8(h >> 4)

static const uint32_t DISK_SECTOR_COUNT = 2 * 8;   // 8KB is the smallest size that windows allow to mount
static const uint16_t DISK_SECTOR_SIZE = 512;      // Should be 512
static const uint16_t DISC_SECTORS_PER_TABLE = 1;  // Each table sector can fit 170KB (340 sectors)

static uint8_t msc_disk[DISK_SECTOR_COUNT][DISK_SECTOR_SIZE] = {
  //------------- Block0: Boot Sector -------------//
  {                                                        // Header (62 bytes)
   0xEB, 0x3C, 0x90,                                       // jump_instruction
   'M', 'S', 'D', 'O', 'S', '5', '.', '0',                 // oem_name
   FAT_U16(DISK_SECTOR_SIZE),                              // bytes_per_sector
   FAT_U8(1),                                              // sectors_per_cluster
   FAT_U16(1),                                             // reserved_sectors_count
   FAT_U8(1),                                              // file_alloc_tables_num
   FAT_U16(16),                                            // max_root_dir_entries
   FAT_U16(DISK_SECTOR_COUNT),                             // fat12_sector_num
   0xF8,                                                   // media_descriptor
   FAT_U16(DISC_SECTORS_PER_TABLE),                        // sectors_per_alloc_table; // FAT12 and FAT16
   FAT_U16(1),                                             // sectors_per_track; // A value of 0 may indicate LBA-only access
   FAT_U16(1),                                             // num_heads
   FAT_U32(0),                                             // hidden_sectors_count
   FAT_U32(0),                                             // total_sectors_32
   0x00,                                                   // physical_drive_number;0x00 for (first) removable media, 0x80 for (first) fixed disk
   0x00,                                                   // reserved
   0x29,                                                   // extended_boot_signature; // should be 0x29
   FAT_U32(0x1234),                                        // serial_number: 0x1234 => 1234
   'T', 'i', 'n', 'y', 'U', 'S', 'B', ' ', 'M', 'S', 'C',  // volume_label padded with spaces (0x20)
   'F', 'A', 'T', '1', '2', ' ', ' ', ' ',                 // file_system_type padded with spaces (0x20)

   // Zero up to 2 last bytes of FAT magic code (448 bytes)
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

   // boot signature (2 bytes)
   0x55, 0xAA
  },

  //------------- Block1: FAT12 Table -------------//
  {
    FAT_TBL2B(0xFF8, 0xFFF), FAT_TBL2B(0xFFF, 0x000)  // first 2 entries must be 0xFF8 0xFFF, third entry is cluster end of readme file
  },
};

static int32_t writeDataText(uint8_t *buffer, uint32_t bufsize) {
  uint8_t dataSector[DISK_SECTOR_SIZE] = {
    // first entry is volume label
    'o', 'p', 'e', 'n', 'k', 'v', 'm', ' ', ' ', ' ', ' ',
    0x08,                                                                                                                 // FILE_ATTR_VOLUME_LABEL
    0x00, FAT_MS2B(0, 0), FAT_HMS2B(0, 0, 0), FAT_YMD2B(0, 0, 0), FAT_YMD2B(0, 0, 0), FAT_U16(0), FAT_HMS2B(12, 0, 0),    // last_modified_hms
    FAT_YMD2B(2025, 5, 1),                                                                                                // last_modified_ymd
    FAT_U16(0), FAT_U32(0),

    // second entry is data.txt
    'd', 'a', 't', 'a', ' ', ' ', ' ', ' ',  // file_name[8]; padded with spaces (0x20)
    't', 'x', 't',                           // file_extension[3]; padded with spaces (0x20)
    0x20,                                    // file attributes: FILE_ATTR_ARCHIVE
    0x00,                                    // ignore
    FAT_MS2B(1, 980),                        // creation_time_10_ms (max 199x10 = 1s 990ms)
    FAT_HMS2B(12, 0, 0),                     // create_time_hms [5:6:5] => h:m:(s/2)
    FAT_YMD2B(2025, 5, 1),                   // create_time_ymd [7:4:5] => (y+1980):m:d
    FAT_YMD2B(2025, 5, 1),                   // last_access_ymd
    FAT_U16(0),                              // extended_attributes
    FAT_HMS2B(12, 0, 0),                     // last_modified_hms
    FAT_YMD2B(2025, 5, 1),                   // last_modified_ymd
    FAT_U16(2),                              // start of file in cluster
    FAT_U32(bufsize)                         // file size
  };
  memcpy(msc_disk[2], dataSector, DISK_SECTOR_SIZE);
  memcpy(msc_disk[3], buffer, bufsize);
  return bufsize;
}

static int32_t onWrite(uint32_t lba, uint32_t offset, uint8_t *buffer, uint32_t bufsize) {
  // FIXME: prevent to write file besides data.txt, especially ".DS_Store"
  Serial.printf("MSC WRITE: lba: %lu, offset: %lu, bufsize: %lu\n", lba, offset, bufsize);
  memcpy(msc_disk[lba] + offset, buffer, bufsize);
  return bufsize;
}

static int32_t onRead(uint32_t lba, uint32_t offset, void *buffer, uint32_t bufsize) {
  Serial.printf("MSC READ: lba: %lu, offset: %lu, bufsize: %lu\n", lba, offset, bufsize);
  memcpy(buffer, msc_disk[lba] + offset, bufsize);
  return bufsize;
}

static bool onStartStop(uint8_t power_condition, bool start, bool load_eject) {
  Serial.printf("MSC START/STOP: power: %u, start: %u, eject: %u\n", power_condition, start, load_eject);
  return true;
}

static void usbEventCallback(void *arg, esp_event_base_t event_base, int32_t event_id, void *event_data) {
  if (event_base == ARDUINO_USB_EVENTS) {
    arduino_usb_event_data_t *data = (arduino_usb_event_data_t *)event_data;
    switch (event_id) {
      case ARDUINO_USB_STARTED_EVENT: Serial.println("USB PLUGGED"); break;
      case ARDUINO_USB_STOPPED_EVENT: Serial.println("USB UNPLUGGED"); break;
      case ARDUINO_USB_SUSPEND_EVENT: Serial.printf("USB SUSPENDED: remote_wakeup_en: %u\n", data->suspend.remote_wakeup_en); break;
      case ARDUINO_USB_RESUME_EVENT:  Serial.println("USB RESUMED"); break;
      default: break;
    }
  }
}

// endregion

class SerialReader {
private:
  USBHIDKeyboard _keyboard;
  USBHIDAbsoluteMouse _mouse;

  char _pressed_mouse_buttons = 0;

  char _buf[BufferLength] = {};
  int _index = 0;
  bool _acceptable = false;
  int _target_len = 0;

  void handle_key_event(char *buf) {
    bool is_down = buf[1];
    Serial.print("[debug] keydown: ");
    Serial.println(is_down ? "true" : "false");

    int key_code = buf[4] << 24 | buf[5] << 16 | buf[6] << 8 | buf[7];
    Serial.print("[debug] keyCode: ");
    Serial.println(key_code);

    // Serial.print("[debug] code_map_x11_to_usb.size: ");
    // Serial.println(code_map_x11_to_usb.size());

    if (key_code >= code_map_x11_to_usb.size()) {
      Serial.print("[warn] unknown key code: ");
      Serial.println(key_code);
      return;
    }

    char usb_key_code = code_map_x11_to_usb.at(key_code);
    if (is_down) {
      this->_keyboard.pressRaw(usb_key_code);
    } else {
      this->_keyboard.releaseRaw(usb_key_code);
    }
  }

  void handle_pointer_event(char *buf) {
    char button_mask = buf[1];
    Serial.print("[debug] point mask: ");
    Serial.println(button_mask, BIN);

    int x = buf[2] << 8 | buf[3];
    int y = buf[4] << 8 | buf[5];

    Serial.print("[debug] pointer: ");
    Serial.print(x);
    Serial.print(", ");
    Serial.println(y);

    char button = 0;
    char wheel = 0;
    char pan = 0;

    // VNC pointer event to USB button

    if ((button_mask & _MOUSE_LEFT) == _MOUSE_LEFT) {  // left
      button |= MOUSE_LEFT;
    }
    if ((button_mask & _MOUSE_MIDDLE) == _MOUSE_MIDDLE) {  // middle
      button |= MOUSE_MIDDLE;
    }
    if ((button_mask & _MOUSE_RIGHT) == _MOUSE_RIGHT) {  // right
      button |= MOUSE_RIGHT;
    }

    // FIXME
    // This is conflict with scroll event
    // if ((button_mask & MOUSE_BACKWARD) == MOUSE_BACKWARD) {  // backward
    //   button |= MOUSE_BACKWARD;
    // }
    // if ((button_mask & MOUSE_FORWARD) == MOUSE_FORWARD) {  // forward
    //   button |= MOUSE_FORWARD;
    // }

    if ((button_mask & _MOUSE_WHEEL_UP) == _MOUSE_WHEEL_UP) {  // wheel up
      wheel = -50;
    }
    if ((button_mask & _MOUSE_WHEEL_DOWN) == _MOUSE_WHEEL_DOWN) {  // wheel down
      wheel = 50;
    }
    if ((button_mask & _MOUSE_WHEEL_LEFT) == _MOUSE_WHEEL_LEFT) {  // wheel left
      pan = -50;
    }
    if ((button_mask & _MOUSE_WHEEL_RIGHT) == _MOUSE_WHEEL_RIGHT) {  // wheel right
      pan = 50;
    }

    char released_buttons = this->_pressed_mouse_buttons & ~button;
    this->_pressed_mouse_buttons = this->_pressed_mouse_buttons & ~released_buttons | button;

    // this->_mouse.buttons(button_mask);
    Serial.print("[debug] released: ");
    Serial.print(released_buttons, BIN);
    Serial.print(", pressed: ");
    Serial.println(this->_pressed_mouse_buttons, BIN);

    if (released_buttons != 0) {
      this->_mouse.release(released_buttons);
    }
    if (this->_pressed_mouse_buttons != 0) {
      this->_mouse.press(this->_pressed_mouse_buttons);
    }

    this->_mouse.move(x, y, wheel, pan);
  }

public:
  SerialReader(USBHIDKeyboard keyboard, USBHIDAbsoluteMouse mouse) {
    this->_keyboard = keyboard;
    this->_mouse = mouse;
  }

  void push(char b) {
    this->_buf[this->_index] = b;

    // Serial.print(b);

    if (this->_index >= BufferLength) {  // overflowed
      this->_index = 0;
      return;
    }

    if (!this->_acceptable) {
      if (this->_buf[this->_index] == MagicWord[this->_index]) {
        this->_index++;

        // magic word ok
        if (this->_index == MagicWordLength) {
          Serial.println("[debug] magic word accepted");
          this->_acceptable = true;
          this->_index = 0;
        }
      } else {
        this->_index = 0;
      }
      return;
    }

    this->_index++;

    if (this->_target_len > 0) {
      if (this->_index < this->_target_len) {
        return;
      }
    } else {
      switch (b) {
        case KeyEvent:
          this->_target_len = 8;
          Serial.println("[debug] wait for key event");
          break;
        case PointerEvent:
          this->_target_len = 6;
          Serial.println("[debug] wait for pointer event");
          break;
        case ButtonEvent:
          this->_target_len = 4;
          Serial.println("[debug] wait for button event");
          break;
        case ClipboardEvent:
          this->_target_len = 3;
          Serial.println("[debug] wait for button event");
          break;

        case LEDTestEvent:
          this->_target_len = 2;
          Serial.println("[debug] wait for led event");
          break;
        case KeyboardTestEvent:
          this->_target_len = 4;
          Serial.println("[debug] wait for keyboard test event");
          break;
        case MouseTestEvent:
          this->_target_len = 14;
          Serial.println("[debug] wait for mouse test event");
          break;
        case ButtonTestEvent:
          this->_target_len = 5;
          Serial.println("[debug] wait for button test event");
          break;
        case ClipboardTestEvent:
          this->_target_len = 5;
          Serial.println("[debug] wait for clipboard test event");
          break;
        default:
          this->_target_len = 0;
          this->_index = 0;
          Serial.println("[debug] unknown event type, reset buffered index");
      }
      return;
    }

    switch (this->_buf[0]) {
      case KeyEvent:
        this->handle_key_event(this->_buf);
        break;
      case PointerEvent:
        this->handle_pointer_event(this->_buf);
        break;
      case ButtonEvent:
        {
          if (this->_buf[1] == 0x1) {
            bool is_output = this->_buf[3] == 0x1;
            Serial.print("[debug] button event: set pin ");
            Serial.print(int(this->_buf[2]));
            Serial.print(" to ");
            Serial.println(is_output ? "output" : "input");
            pinMode(this->_buf[2], is_output ? OUTPUT : INPUT);
          } else if (this->_buf[1] == 0x2) {
            bool is_high = this->_buf[3] == 0x1;
            Serial.print("[debug] button event: set pin ");
            Serial.print(int(this->_buf[2]));
            Serial.print(" to ");
            Serial.println(is_high ? "high" : "low");
            digitalWrite(this->_buf[2], is_high ? HIGH : LOW);
          } else {
            Serial.print("[debug] button event: unknown sub command: ");
            Serial.println(int(this->_buf[1]));
          }
          break;
        }
      case ClipboardEvent:
        {
          if (this->_target_len == 3) {
            this->_target_len += ((this->_buf[1] << 8) | this->_buf[2]);
            if (this->_target_len == 3) {
              this->_target_len = 0;
              this->_index = 0;
            }
            return;
          }

          char *data = this->_buf + 3;
          int length = this->_target_len - 3;

          writeDataText((uint8_t*) data, length);
          USBSerial.write(data, length);

          break;
        }

      case LEDTestEvent:
        {
          bool on = this->_buf[1] == '1';
          Serial.print("[debug] led test: ");
          Serial.println(on ? "on" : "off");
          digitalWrite(LED_BUILTIN, on ? HIGH : LOW);
          break;
        }
      case KeyboardTestEvent:
        {
          // if there use `3` as length,
          // something will overflow, I have no idea why
          char key_str[5] = {};
          memcpy(key_str, this->_buf + 1, 3);
          char key = atoi(key_str);
          this->_keyboard.write(key);
          Serial.print("[debug] keyboard test: ");
          Serial.print(key);
          Serial.print(" - ");
          Serial.println(int(key));
          break;
        }
      case MouseTestEvent:
        {
          char button = this->_buf[1] - '1' + 1;
          this->_mouse.click(button);

          char x_str[8] = {};
          memcpy(x_str, this->_buf + 2, 6);
          char y_str[8] = {};
          memcpy(y_str, this->_buf + 8, 6);
          int x = atoi(x_str);
          int y = atoi(y_str);
          this->_mouse.move(x, y, 0, 0);

          Serial.print("[debug] mouse test: ");
          Serial.print(int(button));
          Serial.print("@(");
          Serial.print(x);
          Serial.print(",");
          Serial.print(y);
          Serial.println(")");
          break;
        }
      case ButtonTestEvent:
        {
          char pin_str[4] = {};
          memcpy(pin_str, this->_buf + 2, 2);
          int pin = atoi(pin_str);
          if (this->_buf[1] == '1') {
            bool is_output = this->_buf[4] == '1';
            Serial.print("[debug] button test event: set pin ");
            Serial.print(pin);
            Serial.print(" to ");
            Serial.println(is_output ? "output" : "input");
            pinMode(pin, is_output ? OUTPUT : INPUT);
          } else if (this->_buf[1] == '2') {
            bool is_high = this->_buf[4] == '1';
            Serial.print("[debug] button test event: set pin ");
            Serial.print(pin);
            Serial.print(" to ");
            Serial.println(is_high ? "high" : "low");
            digitalWrite(pin, is_high ? HIGH : LOW);
          } else {
            Serial.print("[debug] button event: unknown sub command: ");
            Serial.println(this->_buf[1]);
          }
          break;
        }
      case ClipboardTestEvent:
        {
          char text[4] = {};
          memcpy(text, this->_buf + 1, 4);
          Serial.print("[debug] clipboard test: ");
          Serial.println(text);
          writeDataText((uint8_t*) text, 4);
          USBSerial.write(text, 4);
          break;
        }
      default:
        Serial.println("[warn] unknown event");
    }

    this->_target_len = 0;
    this->_index = 0;
  }
};

SerialReader *serialport = new SerialReader(Keyboard, Mouse);

void setup() {
  Serial.begin(921600);

  Serial.println("[000%] starting...");

  pinMode(LED_BUILTIN, OUTPUT);
  digitalWrite(LED_BUILTIN, LOW);

  writeDataText((uint8_t *)"", 4);

  USB.onEvent(usbEventCallback);

  MSC.vendorID("allape");
  MSC.productID("openkvm");
  MSC.productRevision("1.0");
  MSC.onStartStop(onStartStop);
  MSC.onRead(onRead);
  MSC.onWrite(onWrite);
  MSC.mediaPresent(true);
  MSC.isWritable(false);
  MSC.begin(DISK_SECTOR_COUNT, DISK_SECTOR_SIZE);

  USBSerial.begin();

  Mouse.begin();
  Keyboard.begin();

  USB.begin();

  Serial.println("[100%] ready");
}

void loop() {
  while (Serial.available()) {
    serialport->push(Serial.read());
  }
  delay(1);
}
