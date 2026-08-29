#include <Wire.h>
#include <LiquidCrystal_I2C.h>
#include <errno.h>
#include <stdio.h>

// Zapper v5: samodzielne sterowanie z enkodera oraz protokol USB 115200 N81.
// Wyjscie D9 jest generowane sprzetowo przez Timer1. Dla 0.1-0.119 Hz
// uzywany jest nieblokujacy generator programowy, poniewaz Timer1 ma 16 bitow.

const uint8_t LCD_COLS = 16;
const uint8_t LCD_ROWS = 2;
LiquidCrystal_I2C lcd(0x27, LCD_COLS, LCD_ROWS);

const uint8_t PIN_ENCODER_CLK = 2;
const uint8_t PIN_ENCODER_DT = 3;
const uint8_t PIN_BUTTON = 4;
const uint8_t PIN_OUTPUT = 9; // OC1A

const uint8_t MAX_STEPS = 12;
const uint32_t MIN_FREQUENCY_MILLIHZ = 100UL;
const uint32_t MAX_FREQUENCY_MILLIHZ = 4000000000UL;
const uint32_t MAX_DURATION_MS = 86400000UL;
const uint32_t LCD_REFRESH_MS = 250UL;
const uint32_t STATE_REFRESH_MS = 500UL;
const uint32_t LONG_PRESS_MS = 1000UL;

// BEGIN GENERATED LANGUAGE PACK
#define ZAPPER_FIRMWARE_VERSION "5.1.0"
#define ZAPPER_LANG_CODE "pl"
#define UI_PROGRAM_FMT "PROGRAM %u/3"
#define UI_MANUAL "> RECZNY"
#define UI_STEP_TIME_FMT "K:%s H:CZAS"
#define UI_TIME_FMT "CZAS: %u min"
#define UI_START_MENU "Klik=START H:MEN"
#define UI_USB_READY_FMT "USB GOTOWA %uK"
#define UI_START_CANCEL "Klik=GO H:ANUL"
#define UI_STOP "STOP"
#define UI_SESSION_DONE "KONIEC SESJI"
#define UI_CLICK_MENU "Klik = MENU"
// END GENERATED LANGUAGE PACK

struct SessionStep {
  uint32_t frequencyMilliHz;
  uint32_t durationMs;
};

enum UiState : uint8_t {
  UI_MENU,
  UI_MANUAL_FREQUENCY,
  UI_TIME_SELECT,
  UI_ARMED,
  UI_RUNNING,
  UI_FINISHED,
  UI_SLEEP
};

SessionStep sessionSteps[MAX_STEPS];
uint8_t sessionStepCount = 0;
uint8_t activeStepIndex = 0;
uint32_t activeStepStartedAt = 0;
uint32_t armedAt = 0;
uint32_t lastStateSentAt = 0;
bool remoteSession = false;

UiState uiState = UI_MENU;
uint8_t menuSelection = 0;
uint32_t selectedFrequencyMilliHz = 30000000UL;
uint8_t selectedMinutes = 7;
const uint32_t frequencySteps[] = {100UL, 1000UL, 10000UL, 100000UL, 1000000UL, 10000000UL};
uint8_t frequencyStepIndex = 4;

bool softwareGenerator = false;
bool softwareOutputHigh = false;
uint32_t softwareHalfPeriodMicros = 0;
uint32_t softwareLastToggleAt = 0;

int lastEncoderClock = HIGH;
uint32_t lastEncoderEventAt = 0;
bool buttonPressed = false;
bool lastRawButton = false;
bool wakeButtonGuard = false;
uint32_t rawButtonChangedAt = 0;
uint32_t buttonPressedAt = 0;
uint32_t lastLcdRefreshAt = 0;
bool lcdDirty = true;
char lcdCache[LCD_ROWS][LCD_COLS + 1];
bool lcdCacheValid[LCD_ROWS] = {false, false};

char serialLine[64];
uint8_t serialLineLength = 0;
bool serialLineOverflow = false;

void stopGenerator();
void emitState();
void renderScreen();
void invalidateLcdCache();
void markLcdDirty();
void clearLcdAndInvalidate();

void setup() {
  pinMode(PIN_ENCODER_CLK, INPUT);
  pinMode(PIN_ENCODER_DT, INPUT);
  pinMode(PIN_BUTTON, INPUT_PULLUP);
  pinMode(PIN_OUTPUT, OUTPUT);
  digitalWrite(PIN_OUTPUT, LOW);

  Serial.begin(115200);
  lcd.init();
  lcd.backlight();
  clearLcdAndInvalidate();
  lastEncoderClock = digitalRead(PIN_ENCODER_CLK);
  lastRawButton = digitalRead(PIN_BUTTON) == LOW;
  renderScreen();
  Serial.print(F("READY ZAPPER_V5 "));
  Serial.print(F(ZAPPER_FIRMWARE_VERSION));
  Serial.print(' ');
  Serial.println(F(ZAPPER_LANG_CODE));
}

void loop() {
  handleSerial();
  handleEncoder();
  handleButton();
  updateSoftwareGenerator();
  updateSession();

  uint32_t now = millis();
  if (uiState != UI_SLEEP && (lcdDirty || (uint32_t)(now - lastLcdRefreshAt) >= LCD_REFRESH_MS)) {
    lastLcdRefreshAt = now;
    renderScreen();
    lcdDirty = false;
  }
}

// ---- Generator -----------------------------------------------------------

bool startHardwareGenerator(uint32_t frequencyMilliHz) {
  const uint16_t divisors[] = {1, 8, 64, 256, 1024};
  const uint8_t clockBits[] = {
    _BV(CS10),
    _BV(CS11),
    (uint8_t)(_BV(CS11) | _BV(CS10)),
    _BV(CS12),
    (uint8_t)(_BV(CS12) | _BV(CS10))
  };

  for (uint8_t index = 0; index < 5; index++) {
    uint64_t denominator = 2ULL * divisors[index] * frequencyMilliHz;
    uint64_t ticks = ((uint64_t)F_CPU * 1000ULL + denominator / 2ULL) / denominator;
    if (ticks < 1ULL || ticks > 65536ULL) {
      continue;
    }

    noInterrupts();
    TCCR1A = 0;
    TCCR1B = 0;
    TCNT1 = 0;
    OCR1A = (uint16_t)(ticks - 1ULL);
    TCCR1A = _BV(COM1A0);
    TCCR1B = (uint8_t)(_BV(WGM12) | clockBits[index]);
    interrupts();
    return true;
  }
  return false;
}

void startGenerator(uint32_t frequencyMilliHz) {
  stopGenerator();
  if (startHardwareGenerator(frequencyMilliHz)) {
    return;
  }

  softwareGenerator = true;
  softwareOutputHigh = false;
  softwareHalfPeriodMicros = (uint32_t)(500000000ULL / frequencyMilliHz);
  softwareLastToggleAt = micros();
  digitalWrite(PIN_OUTPUT, LOW);
}

void stopGenerator() {
  noInterrupts();
  TCCR1A = 0;
  TCCR1B = 0;
  TCNT1 = 0;
  interrupts();
  softwareGenerator = false;
  softwareOutputHigh = false;
  digitalWrite(PIN_OUTPUT, LOW);
}

void updateSoftwareGenerator() {
  if (!softwareGenerator || uiState != UI_RUNNING) {
    return;
  }
  uint32_t now = micros();
  if ((uint32_t)(now - softwareLastToggleAt) < softwareHalfPeriodMicros) {
    return;
  }
  softwareLastToggleAt = now;
  softwareOutputHigh = !softwareOutputHigh;
  digitalWrite(PIN_OUTPUT, softwareOutputHigh ? HIGH : LOW);
}

// ---- Sesje ---------------------------------------------------------------

void beginActiveStep() {
  activeStepStartedAt = millis();
  lastStateSentAt = activeStepStartedAt;
  startGenerator(sessionSteps[activeStepIndex].frequencyMilliHz);
  markLcdDirty();
  emitState();
}

void launchSession(bool fromComputer) {
  if (sessionStepCount == 0) {
    Serial.println(F("ERR NO_STEPS"));
    return;
  }
  if (uiState == UI_SLEEP) {
    lcd.backlight();
  }
  remoteSession = fromComputer;
  activeStepIndex = 0;
  uiState = UI_RUNNING;
  markLcdDirty();
  beginActiveStep();
}

void armRemoteSession() {
  if (sessionStepCount == 0) {
    Serial.println(F("ERR NO_STEPS"));
    return;
  }
  stopGenerator();
  remoteSession = true;
  activeStepIndex = 0;
  armedAt = millis();
  uiState = UI_ARMED;
  lcd.backlight();
  markLcdDirty();
  Serial.print(F("ARMED "));
  Serial.println(sessionStepCount);
}

void startLocalSession() {
  sessionStepCount = 1;
  sessionSteps[0].frequencyMilliHz = selectedFrequencyMilliHz;
  sessionSteps[0].durationMs = (uint32_t)selectedMinutes * 60000UL;
  launchSession(false);
}

void stopSessionToMenu() {
  stopGenerator();
  remoteSession = false;
  uiState = UI_MENU;
  activeStepIndex = 0;
  markLcdDirty();
}

void finishSession() {
  stopGenerator();
  uiState = UI_FINISHED;
  remoteSession = false;
  markLcdDirty();
  Serial.println(F("DONE"));
}

void updateSession() {
  if (uiState == UI_ARMED) {
    // Sesja wysłana z komputera czeka bez limitu. Użytkownik może ją
    // uruchomić krótkim kliknięciem albo anulować długim przytrzymaniem.
    return;
  }
  if (uiState != UI_RUNNING) {
    return;
  }
  uint32_t now = millis();
  uint32_t elapsed = (uint32_t)(now - activeStepStartedAt);
  if (elapsed >= sessionSteps[activeStepIndex].durationMs) {
    if (activeStepIndex + 1 < sessionStepCount) {
      activeStepIndex++;
      beginActiveStep();
    } else {
      finishSession();
    }
    return;
  }
  if ((uint32_t)(now - lastStateSentAt) >= STATE_REFRESH_MS) {
    lastStateSentAt = now;
    emitState();
  }
}

uint32_t remainingMilliseconds() {
  if (uiState != UI_RUNNING) {
    return 0;
  }
  uint32_t elapsed = (uint32_t)(millis() - activeStepStartedAt);
  uint32_t duration = sessionSteps[activeStepIndex].durationMs;
  return elapsed >= duration ? 0 : duration - elapsed;
}

void printReport(const __FlashStringHelper *prefix) {
  Serial.print(prefix);
  if (uiState == UI_ARMED) {
    Serial.print(F(" ARMED 0 "));
    Serial.print(sessionStepCount);
    Serial.println(F(" 0 0"));
  } else if (uiState == UI_RUNNING) {
    Serial.print(F(" RUNNING "));
    Serial.print(activeStepIndex + 1);
    Serial.print(' ');
    Serial.print(sessionStepCount);
    Serial.print(' ');
    Serial.print(sessionSteps[activeStepIndex].frequencyMilliHz);
    Serial.print(' ');
    Serial.println(remainingMilliseconds());
  } else if (uiState == UI_FINISHED) {
    Serial.print(F(" DONE "));
    Serial.print(sessionStepCount);
    Serial.print(' ');
    Serial.print(sessionStepCount);
    Serial.println(F(" 0 0"));
  } else {
    Serial.print(F(" IDLE 0 "));
    Serial.print(sessionStepCount);
    Serial.println(F(" 0 0"));
  }
}

void emitState() {
  printReport(F("STATE"));
}

// ---- Protokol USB --------------------------------------------------------

bool parseUnsigned(const char *text, uint32_t &value) {
  if (text == NULL || *text == '\0' || *text == '-') {
    return false;
  }
  errno = 0;
  char *end = NULL;
  unsigned long parsed = strtoul(text, &end, 10);
  if (errno == ERANGE || end == text || *end != '\0') {
    return false;
  }
  value = (uint32_t)parsed;
  return true;
}

void processCommand(char *line) {
  char *command = strtok(line, " ");
  if (command == NULL) {
    return;
  }

  if (strcmp(command, "PING") == 0) {
    Serial.print(F("OK PONG ZAPPER_V5 "));
    Serial.print(F(ZAPPER_FIRMWARE_VERSION));
    Serial.print(' ');
    Serial.println(F(ZAPPER_LANG_CODE));
    return;
  }

  if (strcmp(command, "STATUS") == 0) {
    printReport(F("STATUS"));
    return;
  }

  if (strcmp(command, "STOP") == 0) {
    stopSessionToMenu();
    Serial.println(F("OK STOP"));
    emitState();
    return;
  }

  if (strcmp(command, "CLEAR") == 0) {
    if (uiState == UI_RUNNING) {
      Serial.println(F("ERR BUSY"));
      return;
    }
    if (uiState == UI_ARMED) stopSessionToMenu();
    sessionStepCount = 0;
    activeStepIndex = 0;
    Serial.println(F("OK CLEAR"));
    return;
  }

  if (strcmp(command, "STEP") == 0) {
    if (uiState == UI_RUNNING || uiState == UI_ARMED) {
      Serial.println(F("ERR BUSY"));
      return;
    }
    if (sessionStepCount >= MAX_STEPS) {
      Serial.println(F("ERR TOO_MANY_STEPS"));
      return;
    }
    char *frequencyText = strtok(NULL, " ");
    char *durationText = strtok(NULL, " ");
    char *extra = strtok(NULL, " ");
    uint32_t frequency = 0;
    uint32_t duration = 0;
    if (extra != NULL || !parseUnsigned(frequencyText, frequency) || !parseUnsigned(durationText, duration)) {
      Serial.println(F("ERR BAD_STEP"));
      return;
    }
    if (frequency < MIN_FREQUENCY_MILLIHZ || frequency > MAX_FREQUENCY_MILLIHZ) {
      Serial.println(F("ERR FREQUENCY_RANGE"));
      return;
    }
    if (duration < 1000UL || duration > MAX_DURATION_MS) {
      Serial.println(F("ERR DURATION_RANGE"));
      return;
    }
    sessionSteps[sessionStepCount].frequencyMilliHz = frequency;
    sessionSteps[sessionStepCount].durationMs = duration;
    sessionStepCount++;
    Serial.print(F("OK STEP "));
    Serial.println(sessionStepCount);
    return;
  }

  if (strcmp(command, "START") == 0) {
    if (uiState == UI_RUNNING || uiState == UI_ARMED) {
      Serial.println(F("ERR BUSY"));
      return;
    }
    if (sessionStepCount == 0) {
      Serial.println(F("ERR NO_STEPS"));
      return;
    }
    Serial.println(F("OK START"));
    armRemoteSession();
    return;
  }

  Serial.println(F("ERR UNKNOWN_COMMAND"));
}

void handleSerial() {
  while (Serial.available() > 0) {
    char character = (char)Serial.read();
    if (character == '\r') {
      continue;
    }
    if (character == '\n') {
      if (serialLineOverflow) {
        Serial.println(F("ERR LINE_TOO_LONG"));
      } else if (serialLineLength > 0) {
        serialLine[serialLineLength] = '\0';
        processCommand(serialLine);
      }
      serialLineLength = 0;
      serialLineOverflow = false;
      continue;
    }
    if (serialLineLength < sizeof(serialLine) - 1) {
      serialLine[serialLineLength++] = character;
    } else {
      serialLineOverflow = true;
    }
  }
}

// ---- Enkoder i przycisk --------------------------------------------------

void wakeDisplay() {
  lcd.backlight();
  clearLcdAndInvalidate();
  uiState = UI_MENU;
  markLcdDirty();
}

void enterSleep() {
  stopGenerator();
  uiState = UI_SLEEP;
  clearLcdAndInvalidate();
  lcd.noBacklight();
}

void adjustManualFrequency(int direction) {
  uint32_t amount = frequencySteps[frequencyStepIndex];
  uint64_t next = selectedFrequencyMilliHz;
  if (direction > 0) {
    next += amount;
  } else if (next > amount) {
    next -= amount;
  } else {
    next = MIN_FREQUENCY_MILLIHZ;
  }
  if (next < MIN_FREQUENCY_MILLIHZ) next = MIN_FREQUENCY_MILLIHZ;
  if (next > MAX_FREQUENCY_MILLIHZ) next = MAX_FREQUENCY_MILLIHZ;
  selectedFrequencyMilliHz = (uint32_t)next;
}

void handleEncoder() {
  int clock = digitalRead(PIN_ENCODER_CLK);
  uint32_t now = millis();
  if (clock != lastEncoderClock && clock == HIGH && (uint32_t)(now - lastEncoderEventAt) >= 2UL) {
    lastEncoderEventAt = now;
    int direction = digitalRead(PIN_ENCODER_DT) != clock ? 1 : -1;
    if (uiState == UI_SLEEP) {
      wakeDisplay();
    } else if (uiState == UI_MENU) {
      int next = (int)menuSelection + direction;
      if (next < 0) next = 2;
      if (next > 2) next = 0;
      menuSelection = (uint8_t)next;
    } else if (uiState == UI_MANUAL_FREQUENCY) {
      adjustManualFrequency(direction);
    } else if (uiState == UI_TIME_SELECT) {
      int next = (int)selectedMinutes + direction;
      if (next < 1) next = 1;
      if (next > 90) next = 90;
      selectedMinutes = (uint8_t)next;
    }
    markLcdDirty();
  }
  lastEncoderClock = clock;
}

void handleButtonRelease(uint32_t heldFor) {
  bool longPress = heldFor >= LONG_PRESS_MS;
  if (uiState == UI_ARMED) {
    if (longPress) {
      stopSessionToMenu();
      Serial.println(F("CANCELLED USER"));
      emitState();
    } else {
      Serial.println(F("OK CONFIRMED"));
      launchSession(true);
    }
    return;
  }
  if (uiState == UI_RUNNING) {
    stopSessionToMenu();
    emitState();
    return;
  }
  if (uiState == UI_FINISHED) {
    uiState = UI_MENU;
    markLcdDirty();
    return;
  }
  if (uiState == UI_MENU) {
    if (longPress) {
      enterSleep();
      return;
    }
    if (menuSelection == 0) {
      selectedFrequencyMilliHz = 30000000UL;
      uiState = UI_MANUAL_FREQUENCY;
    } else {
      selectedFrequencyMilliHz = menuSelection == 1 ? 30000000UL : 7830UL;
      uiState = UI_TIME_SELECT;
    }
  } else if (uiState == UI_MANUAL_FREQUENCY) {
    if (longPress) {
      uiState = UI_TIME_SELECT;
    } else {
      frequencyStepIndex = (frequencyStepIndex + 1) % 6;
    }
  } else if (uiState == UI_TIME_SELECT) {
    if (longPress) {
      uiState = UI_MENU;
    } else {
      startLocalSession();
    }
  }
  markLcdDirty();
}

void handleButton() {
  bool rawPressed = digitalRead(PIN_BUTTON) == LOW;
  uint32_t now = millis();
  if (rawPressed != lastRawButton) {
    lastRawButton = rawPressed;
    rawButtonChangedAt = now;
  }
  if ((uint32_t)(now - rawButtonChangedAt) < 25UL || rawPressed == buttonPressed) {
    return;
  }
  buttonPressed = rawPressed;
  if (buttonPressed) {
    buttonPressedAt = now;
    if (uiState == UI_SLEEP) {
      wakeDisplay();
      wakeButtonGuard = true;
    }
  } else {
    if (wakeButtonGuard) {
      wakeButtonGuard = false;
      return;
    }
    handleButtonRelease((uint32_t)(now - buttonPressedAt));
  }
}

// ---- LCD -----------------------------------------------------------------

void invalidateLcdCache() {
  for (uint8_t row = 0; row < LCD_ROWS; row++) {
    lcdCacheValid[row] = false;
  }
}

void markLcdDirty() {
  lcdDirty = true;
}

void clearLcdAndInvalidate() {
  lcd.clear();
  invalidateLcdCache();
  markLcdDirty();
}

void writeLcdLine(uint8_t row, const char *text) {
  if (row >= LCD_ROWS) return;

  char padded[LCD_COLS + 1];
  uint8_t length = 0;
  while (length < LCD_COLS && text[length] != '\0') {
    padded[length] = text[length];
    length++;
  }
  while (length < LCD_COLS) padded[length++] = ' ';
  padded[LCD_COLS] = '\0';

  uint8_t column = 0;
  while (column < LCD_COLS) {
    if (lcdCacheValid[row] && lcdCache[row][column] == padded[column]) {
      column++;
      continue;
    }
    uint8_t start = column;
    while (column < LCD_COLS && (!lcdCacheValid[row] || lcdCache[row][column] != padded[column])) {
      column++;
    }
    lcd.setCursor(start, row);
    for (uint8_t index = start; index < column; index++) {
      lcd.print(padded[index]);
      lcdCache[row][index] = padded[index];
    }
  }
  lcdCache[row][LCD_COLS] = '\0';
  lcdCacheValid[row] = true;
}

void formatFrequencyText(uint32_t frequencyMilliHz, char *output, size_t outputSize) {
  uint32_t scale;
  const char *unit;
  if (frequencyMilliHz >= 1000000000UL) {
    scale = 1000000000UL;
    unit = "MHz";
  } else if (frequencyMilliHz >= 1000000UL) {
    scale = 1000000UL;
    unit = "kHz";
  } else {
    scale = 1000UL;
    unit = "Hz";
  }
  uint32_t whole = frequencyMilliHz / scale;
  uint16_t fraction = (uint16_t)(((uint64_t)(frequencyMilliHz % scale) * 1000ULL) / scale);
  snprintf(output, outputSize, "%lu.%03u %s", (unsigned long)whole, (unsigned int)fraction, unit);
}

void formatClockText(uint32_t milliseconds, char *output, size_t outputSize) {
  uint32_t seconds = (milliseconds + 999UL) / 1000UL;
  uint16_t minutes = (uint16_t)(seconds / 60UL);
  uint8_t remainder = (uint8_t)(seconds % 60UL);
  snprintf(output, outputSize, "%02u:%02u", (unsigned int)minutes, (unsigned int)remainder);
}

void renderScreen() {
  if (uiState == UI_SLEEP) return;

  char line[LCD_COLS + 1];
  char frequency[16];
  char clockText[10];

  if (uiState == UI_MENU) {
    snprintf(line, sizeof(line), UI_PROGRAM_FMT, (unsigned int)(menuSelection + 1));
    writeLcdLine(0, line);
    if (menuSelection == 0) writeLcdLine(1, UI_MANUAL);
    else if (menuSelection == 1) writeLcdLine(1, "> CLARK 30 kHz");
    else writeLcdLine(1, "> SCHUMANN 7.83");
    return;
  }

  if (uiState == UI_MANUAL_FREQUENCY) {
    formatFrequencyText(selectedFrequencyMilliHz, frequency, sizeof(frequency));
    snprintf(line, sizeof(line), "F:%s", frequency);
    writeLcdLine(0, line);
    formatFrequencyText(frequencySteps[frequencyStepIndex], frequency, sizeof(frequency));
    snprintf(line, sizeof(line), UI_STEP_TIME_FMT, frequency);
    writeLcdLine(1, line);
    return;
  }

  if (uiState == UI_TIME_SELECT) {
    snprintf(line, sizeof(line), UI_TIME_FMT, (unsigned int)selectedMinutes);
    writeLcdLine(0, line);
    writeLcdLine(1, UI_START_MENU);
    return;
  }

  if (uiState == UI_ARMED) {
    snprintf(line, sizeof(line), UI_USB_READY_FMT, (unsigned int)sessionStepCount);
    writeLcdLine(0, line);
    writeLcdLine(1, UI_START_CANCEL);
    return;
  }

  if (uiState == UI_RUNNING) {
    formatFrequencyText(sessionSteps[activeStepIndex].frequencyMilliHz, frequency, sizeof(frequency));
    writeLcdLine(0, frequency);
    formatClockText(remainingMilliseconds(), clockText, sizeof(clockText));
    snprintf(line, sizeof(line), "%u/%u %s %s", (unsigned int)(activeStepIndex + 1), (unsigned int)sessionStepCount, clockText, UI_STOP);
    writeLcdLine(1, line);
    return;
  }

  writeLcdLine(0, UI_SESSION_DONE);
  writeLcdLine(1, UI_CLICK_MENU);
}
