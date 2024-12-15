#include "Arduino.h"

#include "USB.h"
#include "USBHIDMouse.h"
#include "USBHIDKeyboard.h"

#include "keys.h"

#define BufferLength 256

#define MagicWord "open-kvm"
#define MagicWordLength 8

#define KeyEvent 4      // https://datatracker.ietf.org/doc/html/rfc6143#section-7.5.4
#define PointerEvent 5  // https://datatracker.ietf.org/doc/html/rfc6143#section-7.5.5

// CMD  TYPE PIN  VALUE
// 0xff 0x01 0x0b 0x01
//
// CMD: fixed value 0xff
// PIN: pin number
// TYPE:
//   0x01 for initialing
//     VALUE: 0x00 for input
//     VALUE: 0x01 for output
//   0x02 for set pin
//     VALUE: 0x00 for LOW
//     VALUE: 0x01 for HIGH
#define ButtonEvent 0xff  // power button, rest button, etc

// screen /dev/cu.wchusbserialxxx 460800 \n
// open-kvm\n

// test the builtin LED
// "a1" lights up, "a0" lights off
#define LEDTestEvent 'a'
// test keyboard
// "b049" for 1
// "b032" for Space
// "b027" for Esc
// "b013" for Enter
// "bNNN", NNN is the key code, 000~255
#define KeyboardTestEvent 'b'
// test mouse
// "c1000001-00002" for click left button at (0001, 0002)
// "c2000000000000" for right
// "c4000000000000" for middle
// "c0010000-10000" for moving cursor to (100, 100)
// "cNXXXXXXYYYYYY", N is button, XXXXXX is x axis(int16), YYYYYY is y axis(int16)
#define MouseTestEvent 'c'

// test button
// "d1121" for set pin 12 to output
// "d2121" for set pin 12 to HIGH
// "d2120" for set pin 12 to LOW
#define ButtonTestEvent 'd'

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
      default:
        Serial.println("[warn] unknown event");
    }

    this->_target_len = 0;
    this->_index = 0;
  }
};

//void on_spi(void *)
//{
//  while (1)
//  {
//    delay(1500);
//    // Serial.print("_");
//
//    // TODO start a SPI to listen data from a GPIO device
//  }
//}

USBHIDKeyboard Keyboard;
USBHIDAbsoluteMouse Mouse;

SerialReader *serialport = new SerialReader(Keyboard, Mouse);

void setup() {
  Serial.begin(460800);

  Serial.println("[000%] starting...");

  pinMode(LED_BUILTIN, OUTPUT);
  digitalWrite(LED_BUILTIN, LOW);

  Mouse.begin();
  Keyboard.begin();
  USB.begin();

  Serial.println("[100%] ready");

  //  // start a new thread
  //  xTaskCreatePinnedToCore(
  //      on_spi,   // the task
  //      "on_spi", // the name of the task
  //      10000,    // stack size
  //      NULL,     // parameters
  //      1,        // priority
  //      NULL,     // task handle
  //      0         // core
  //  );
}

void loop() {
  while (Serial.available() > 0) {
    serialport->push(Serial.read());
  }
  delay(500);
}
