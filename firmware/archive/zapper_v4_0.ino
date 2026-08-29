// ARCHIWALNY FIRMWARE ZAPPER v4.0
// Zachowany wyłącznie jako poprzednia wersja. Aktualny firmware: firmware/zapper_v5/zapper_v5.ino

#include <Wire.h>
#include <LiquidCrystal_I2C.h>

LiquidCrystal_I2C lcd(0x27, 20, 4);

#define CLK 2
#define DT 3
#define SW 4
#define ZAPPER_PIN 9

// --- USTAWIENIA ---
double frequency = 1.0;
double selectedFreq = 30000.0;
double step = 1000.0;
int stepIndex = 4;

// Czas
int selectedTimeMinutes = 7;
unsigned long therapyDuration = 0;
unsigned long startTime = 0;
unsigned long elapsedTime = 0;

// Opcje w menu dolnym
int subMenuOption = 0;

// Stany maszyny
enum State { MAIN_MENU, MANUAL_SETUP, MANUAL_OPTIONS, TIME_SELECT, RUNNING, FINISHED, SLEEP };
State currentState = MAIN_MENU;

// Obsługa wejścia
int lastCLKState;
unsigned long lastButtonPress = 0;
bool buttonActive = false;
bool longPressDetected = false;
bool isStopping = false;

// Cooldown po obudzeniu
unsigned long wakeUpTime = 0;

// Generator
unsigned long previousMicros = 0;
int outputState = LOW;

void setup() {
  pinMode(CLK, INPUT);
  pinMode(DT, INPUT);
  pinMode(SW, INPUT_PULLUP);
  pinMode(ZAPPER_PIN, OUTPUT);
  digitalWrite(ZAPPER_PIN, LOW);

  lcd.init();
  lcd.backlight();
  lcd.clear();

  showMainMenu();
  lastCLKState = digitalRead(CLK);
}

void loop() {
  if (currentState != SLEEP) {
    handleGenerator();
  }
  handleEncoder();
  handleButton();
}

void handleGenerator() {
  if (currentState == RUNNING) {
    if (selectedFreq < 1000.0) {
      unsigned long interval = (unsigned long)(1000000.0 / selectedFreq / 2.0);
      if (micros() - previousMicros >= interval) {
        previousMicros = micros();
        outputState = !outputState;
        digitalWrite(ZAPPER_PIN, outputState);
      }
    } else {
      tone(ZAPPER_PIN, (unsigned int)selectedFreq);
    }
  } else {
    noTone(ZAPPER_PIN);
    digitalWrite(ZAPPER_PIN, LOW);
  }
}

void handleButton() {
  int btnState = digitalRead(SW);
  if (btnState == LOW) {
    if (!buttonActive) {
      buttonActive = true;
      lastButtonPress = millis();
    }

    unsigned long pressDuration = millis() - lastButtonPress;
    if (currentState == SLEEP) return;

    if (currentState == RUNNING || currentState == MAIN_MENU) {
      if (pressDuration > 500) {
        isStopping = true;
        lcd.setCursor(0, 3);
        if (currentState == RUNNING) {
          if (pressDuration < 1500) lcd.print("ZATRZYMANIE ZA 3..  ");
          else if (pressDuration < 2500) lcd.print("ZATRZYMANIE ZA 2..  ");
          else if (pressDuration < 3500) lcd.print("ZATRZYMANIE ZA 1..  ");
          else {
            stopTherapy();
            buttonActive = false;
            longPressDetected = true;
          }
        } else if (currentState == MAIN_MENU) {
          if (pressDuration < 1500) lcd.print("WYLACZANIE ZA 3..   ");
          else if (pressDuration < 2500) lcd.print("WYLACZANIE ZA 2..   ");
          else if (pressDuration < 3500) lcd.print("WYLACZANIE ZA 1..   ");
          else {
            enterSleep();
            buttonActive = false;
            longPressDetected = true;
          }
        }
      }
    } else if (currentState == MANUAL_SETUP) {
      if (pressDuration > 1000 && !longPressDetected) {
        longPressDetected = true;
        currentState = MANUAL_OPTIONS;
        subMenuOption = 0;
        showManualOptions();
      }
    }
  } else {
    if (buttonActive) {
      if (currentState == SLEEP) {
        wakeUp();
        buttonActive = false;
        return;
      }
      if (isStopping) {
        isStopping = false;
        if (currentState == RUNNING) {
          lcd.setCursor(0, 3);
          lcd.print("                    ");
        } else if (currentState == MAIN_MENU) {
          showMainMenu();
        }
      }
      if (!longPressDetected && (millis() - lastButtonPress > 50)) {
        if (millis() - wakeUpTime > 1000) {
          if (currentState != RUNNING) handleClick();
        }
      }
      buttonActive = false;
      longPressDetected = false;
    }
  }
}

void handleClick() {
  if (currentState == MAIN_MENU) {
    if (frequency == 1.0) {
      selectedFreq = 30000.0;
      currentState = MANUAL_SETUP;
      lcd.clear();
      showManualSetup();
    } else if (frequency == 2.0) {
      selectedFreq = 30000.0;
      currentState = TIME_SELECT;
      lcd.clear();
      showTimeSelect();
    } else if (frequency == 3.0) {
      selectedFreq = 7.83;
      currentState = TIME_SELECT;
      lcd.clear();
      showTimeSelect();
    }
  } else if (currentState == MANUAL_SETUP) {
    changeStep();
    showManualSetup();
  } else if (currentState == MANUAL_OPTIONS) {
    if (subMenuOption == 0) {
      currentState = TIME_SELECT;
      lcd.clear();
      showTimeSelect();
    } else if (subMenuOption == 1) {
      selectedFreq = 0.0;
      currentState = MANUAL_SETUP;
      lcd.clear();
      showManualSetup();
    } else if (subMenuOption == 2) {
      currentState = MAIN_MENU;
      lcd.clear();
      showMainMenu();
    }
  } else if (currentState == TIME_SELECT) {
    therapyDuration = (unsigned long)selectedTimeMinutes * 60 * 1000;
    currentState = RUNNING;
    startTherapy();
  } else if (currentState == FINISHED) {
    currentState = MAIN_MENU;
    showMainMenu();
  }
}

void enterSleep() {
  currentState = SLEEP;
  noTone(ZAPPER_PIN);
  digitalWrite(ZAPPER_PIN, LOW);
  lcd.clear();
  lcd.noBacklight();
  while (digitalRead(SW) == LOW) { delay(10); }
}

void wakeUp() {
  currentState = MAIN_MENU;
  wakeUpTime = millis();
  lcd.backlight();
  lcd.clear();
  showMainMenu();
}

void handleEncoder() {
  int currentStateCLK = digitalRead(CLK);
  if (currentStateCLK != lastCLKState && currentStateCLK == 1) {
    if (currentState == SLEEP) {
      wakeUp();
      lastCLKState = currentStateCLK;
      return;
    }

    if (digitalRead(DT) != currentStateCLK) {
      if (currentState == MAIN_MENU) {
        frequency += 1.0;
        if (frequency > 3.0) frequency = 3.0;
        showMainMenu();
      } else if (currentState == MANUAL_SETUP) {
        selectedFreq += step;
        showManualSetup();
      } else if (currentState == MANUAL_OPTIONS) {
        subMenuOption++;
        if (subMenuOption > 2) subMenuOption = 2;
        showManualOptions();
      } else if (currentState == TIME_SELECT) {
        selectedTimeMinutes++;
        if (selectedTimeMinutes > 90) selectedTimeMinutes = 90;
        showTimeSelect();
      }
    } else {
      if (currentState == MAIN_MENU) {
        frequency -= 1.0;
        if (frequency < 1.0) frequency = 1.0;
        showMainMenu();
      } else if (currentState == MANUAL_SETUP) {
        selectedFreq -= step;
        if (selectedFreq < 0.1) selectedFreq = 0.1;
        showManualSetup();
      } else if (currentState == MANUAL_OPTIONS) {
        subMenuOption--;
        if (subMenuOption < 0) subMenuOption = 0;
        showManualOptions();
      } else if (currentState == TIME_SELECT) {
        selectedTimeMinutes--;
        if (selectedTimeMinutes < 1) selectedTimeMinutes = 1;
        showTimeSelect();
      }
    }
  }
  lastCLKState = currentStateCLK;
  if (currentState == RUNNING && !isStopping) updateProgressBar();
}

void changeStep() {
  stepIndex++;
  if (stepIndex > 5) stepIndex = 0;
  if (stepIndex == 0) step = 0.1;
  if (stepIndex == 1) step = 1.0;
  if (stepIndex == 2) step = 10.0;
  if (stepIndex == 3) step = 100.0;
  if (stepIndex == 4) step = 1000.0;
  if (stepIndex == 5) step = 10000.0;
}

void showManualSetup() {
  lcd.setCursor(0, 0);
  lcd.print("FREQ: ");
  lcd.print(selectedFreq, 1);
  lcd.print(" Hz      ");
  lcd.setCursor(0, 2);
  lcd.print("KROK: ");
  if (step >= 1000) {
    lcd.print(step / 1000, 0);
    lcd.print("kHz ");
  } else if (step < 1) {
    lcd.print("0.1 Hz");
  } else {
    lcd.print(step, 0);
    lcd.print(" Hz  ");
  }
  lcd.setCursor(0, 3);
  lcd.print("[PRZYTRZYMAJ=OPCJE]");
}

void showManualOptions() {
  lcd.setCursor(0, 3);
  lcd.print("                    ");
  lcd.setCursor(0, 3);
  if (subMenuOption == 0) lcd.print(" [START] ZERUJ MENU ");
  else if (subMenuOption == 1) lcd.print(" START [ZERUJ] MENU ");
  else if (subMenuOption == 2) lcd.print(" START ZERUJ [MENU] ");
}

void showMainMenu() {
  lcd.setCursor(0, 0);
  lcd.print("WYBIERZ PROGRAM:");
  lcd.setCursor(0, 1);
  if (frequency == 1.0) lcd.print("> RECZNY (Manual)   ");
  else lcd.print("  RECZNY (Manual)   ");
  lcd.setCursor(0, 2);
  if (frequency == 2.0) lcd.print("> CLARK (30kHz)     ");
  else lcd.print("  CLARK (30kHz)     ");
  lcd.setCursor(0, 3);
  if (frequency == 3.0) lcd.print("> SCHUMANN (7.83Hz) ");
  else lcd.print("  SCHUMANN (7.83Hz) ");
}

void showTimeSelect() {
  lcd.setCursor(0, 0);
  lcd.print("USTAW CZAS TERAPII:");
  lcd.setCursor(5, 2);
  lcd.print(selectedTimeMinutes);
  lcd.print(" MINUT   ");
  lcd.setCursor(0, 3);
  lcd.print("[KLIKNIJ = START]   ");
}

void startTherapy() {
  lcd.clear();
  lcd.setCursor(0, 0);
  lcd.print("TERAPIA TRWA...");
  lcd.setCursor(0, 1);
  lcd.print(selectedFreq, 1);
  lcd.print(" Hz");
  startTime = millis();
}

void stopTherapy() {
  currentState = MAIN_MENU;
  noTone(ZAPPER_PIN);
  digitalWrite(ZAPPER_PIN, LOW);
  lcd.clear();
  showMainMenu();
}

void updateProgressBar() {
  elapsedTime = millis() - startTime;
  if (elapsedTime >= therapyDuration) {
    currentState = FINISHED;
    noTone(ZAPPER_PIN);
    digitalWrite(ZAPPER_PIN, LOW);
    lcd.clear();
    lcd.setCursor(5, 1);
    lcd.print("KONIEC!");
    lcd.setCursor(0, 3);
    lcd.print("[KLIKNIJ ABY WROCIC]");
    for (int i = 0; i < 3; i++) {
      tone(ZAPPER_PIN, 1000);
      delay(200);
      noTone(ZAPPER_PIN);
      delay(200);
    }
    return;
  }
  if (elapsedTime % 500 < 20) {
    unsigned long remaining = (therapyDuration - elapsedTime) / 1000;
    lcd.setCursor(14, 0);
    if (remaining / 60 < 10) lcd.print("0");
    lcd.print(remaining / 60);
    lcd.print(":");
    if (remaining % 60 < 10) lcd.print("0");
    lcd.print(remaining % 60);

    if (!isStopping) {
      lcd.setCursor(0, 3);
      lcd.print("[");
      int progress = map(elapsedTime, 0, therapyDuration, 0, 18);
      for (int i = 0; i < 18; i++) {
        if (i < progress) lcd.print("#");
        else lcd.print(" ");
      }
      lcd.print("]");
    }
  }
}
