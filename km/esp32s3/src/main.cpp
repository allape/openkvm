#include "Arduino.h"

#include "USB.h"
#include "USBHIDMouse.h"
#include "USBHIDKeyboard.h"

#include "keys.h"

#define BufferLength 256

#define MagicWord "open-kvm"
#define MagicWordLength 8

#define KeyEvent 4       // https://datatracker.ietf.org/doc/html/rfc6143#section-7.5.4
#define PointerEvent 5   // https://datatracker.ietf.org/doc/html/rfc6143#section-7.5.5
#define ButtonEvent 0xff // power button, rest button, etc

// screen /dev/cu.wchusbserialxxx 9600 \n
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
// "c10001-002" for left button with offset (00001, -0002)
// "c200000000" for right
// "c400000000" for middle
// "c00100-100" for moving cursor with (100, -100)
// "cNXXXXYYYY", N is button, XXXX is x offset(-128~127), YYYY is y offset(-128~127)
#define MouseTestEvent 'c'

class SerialReader
{
private:
  USBHIDKeyboard _keyboard;
  USBHIDMouse _mouse;

  char _pressed_mouse_buttons = 0;
  int _last_mouse_x = 0;
  int _last_mouse_y = 0;

  char _buf[BufferLength] = {};
  int _index = 0;
  bool _acceptable = false;
  int _target_len = 0;

  void handle_key_event(char *buf)
  {
    bool is_down = buf[1];
    Serial.print("[debug] keydown: ");
    Serial.println(is_down ? "true" : "false");

    int key_code = buf[4] << 24 | buf[5] << 16 | buf[6] << 8 | buf[7];
    Serial.print("[debug] keyCode: ");
    Serial.println(key_code);

    // Serial.print("[debug] code_map_x11_to_usb.size: ");
    // Serial.println(code_map_x11_to_usb.size());

    if (key_code >= code_map_x11_to_usb.size())
    {
      Serial.print("[warn] unknown key code: ");
      Serial.println(key_code);
      return;
    }

    char usb_key_code = code_map_x11_to_usb.at(key_code);
    if (is_down)
    {
      this->_keyboard.pressRaw(usb_key_code);
    }
    else
    {
      this->_keyboard.releaseRaw(usb_key_code);
    }
  }

  void handle_pointer_event(char *buf)
  {
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

    // vnc pointer event to USB button
    if (button_mask & 0b1 == 0b1)
    { // left
      button |= MOUSE_LEFT;
    }
    if (button_mask & 0b10 == 0b10)
    { // middle
      button |= MOUSE_MIDDLE;
    }
    if (button_mask & 0b100 == 0b100)
    { // right
      button |= MOUSE_RIGHT;
    }
    if (button_mask & 0b1000 == 0b1000)
    { // wheel up
      wheel = 1;
    }
    if (button_mask & 0b10000 == 0b10000)
    { // wheel down
      wheel = -1;
    }
    if (button_mask && 0b100000 == 0b100000)
    { // backward
      button |= MOUSE_BACKWARD;
    }
    if (button_mask && 0b1000000 == 0b1000000)
    { // forward
      button |= MOUSE_FORWARD;
    }

    char released_buttons = this->_pressed_mouse_buttons & ~button_mask;
    this->_pressed_mouse_buttons = this->_pressed_mouse_buttons & ~released_buttons | button_mask;

    // this->_mouse.buttons(button_mask);
    Serial.print("[debug] released: ");
    Serial.print(released_buttons, BIN);
    Serial.print(", pressed: ");
    Serial.println(this->_pressed_mouse_buttons, BIN);

    if (released_buttons != 0)
    {
      this->_mouse.release(released_buttons);
    }
    if (this->_pressed_mouse_buttons != 0)
    {
      this->_mouse.press(this->_pressed_mouse_buttons);
    }

    this->_mouse.move(x - this->_last_mouse_x, y - this->_last_mouse_y, wheel, 0);
    this->_last_mouse_x = x;
    this->_last_mouse_y = y;
  }

public:
  SerialReader(USBHIDKeyboard keyboard, USBHIDMouse mouse)
  {
    this->_keyboard = keyboard;
    this->_mouse = mouse;
  }

  void push(char b)
  {
    this->_buf[this->_index] = b;

    // Serial.print(b);

    if (this->_index >= BufferLength)
    { // overflowed
      this->_index = 0;
      return;
    }

    if (!this->_acceptable)
    {
      if (this->_buf[this->_index] == MagicWord[this->_index])
      {
        this->_index++;

        // magic word ok
        if (this->_index == MagicWordLength)
        {
          Serial.println("[debug] magic word accepted");
          this->_acceptable = true;
          this->_index = 0;
        }
      }
      else
      {
        this->_index = 0;
      }
      return;
    }

    this->_index++;

    if (this->_target_len > 0)
    {
      if (this->_index < this->_target_len)
      {
        return;
      }
    }
    else
    {
      switch (b)
      {
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
        Serial.println("[debug] wait for led event");
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
        this->_target_len = 10;
        Serial.println("[debug] wait for mouse test event");
        break;
      default:
        this->_target_len = 0;
        this->_index = 0;
        Serial.println("[debug] unknown event type, reset buffered index");
      }
      return;
    }

    switch (this->_buf[0])
    {
    case KeyEvent:
      this->handle_key_event(this->_buf);
      break;
    case PointerEvent:
      this->handle_pointer_event(this->_buf);
      break;
    case ButtonEvent:
      // TODO
      // 0: 0xff: button event
      // 1: 0x00: padding
      // 2: 0x??: power button, this byte represents the seconds to be hold down, min 1, max 255
      // 3: 0x??: reset button, this byte represents the seconds to be hold down, min 1, max 255
    case LEDTestEvent:
    {
      bool on = this->_buf[1] == '1';
      Serial.print("[debug] led test: ");
      Serial.println(on ? "on" : "off");
      digitalWrite(LED_BUILTIN, on ? HIGH : LOW);
    }
    break;
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

      char x_str[6] = {};
      memcpy(x_str, this->_buf + 2, 4);
      char y_str[6] = {};
      memcpy(y_str, this->_buf + 6, 4);
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
    default:
      Serial.println("[warn] unknown event");
    }

    this->_target_len = 0;
    this->_index = 0;
  }
};

void on_spi(void *)
{
  while (1)
  {
    delay(1500);
    // Serial.print("_");

    // TODO start a SPI to listen data from a GPIO device
  }
}

USBHIDKeyboard Keyboard;
USBHIDMouse Mouse;

SerialReader *serialport = new SerialReader(Keyboard, Mouse);

void setup()
{
  Serial.begin(115200);

  Serial.println("[000%] starting...");

  pinMode(LED_BUILTIN, OUTPUT);
  digitalWrite(LED_BUILTIN, LOW);

  Mouse.begin();
  Keyboard.begin();
  USB.begin();

  Serial.println("[100%] ready");

  // start a new thread
  xTaskCreatePinnedToCore(
      on_spi,   // the task
      "on_spi", // the name of the task
      10000,    // stack size
      NULL,     // parameters
      1,        // priority
      NULL,     // task handle
      0         // core
  );
}

void loop()
{
  while (Serial.available() > 0)
  {
    serialport->push(Serial.read());
  }
  delay(500);
}
