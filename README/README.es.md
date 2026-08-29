**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

La nueva versión de la aplicación funciona en una sola ventana y no requiere Python, Node.js ni Wails. Puede utilizarse como planificador y registro sin una placa conectada, o controlar un Arduino Nano mediante USB.

## Licencia y responsabilidad

El código, el firmware, los esquemas y la documentación están disponibles públicamente para uso no comercial bajo la licencia **PolyForm Noncommercial 1.0.0**. Pueden utilizarse, estudiarse, modificarse y distribuirse para los fines permitidos por esa licencia, pero el proyecto no puede utilizarse comercialmente sin un permiso independiente del autor. Consulta el archivo `LICENSE` para obtener más detalles.

El proyecto se proporciona para experimentos independientes y usos DIY sin garantía. El usuario es responsable del montaje correcto, las modificaciones y la forma de utilizar el dispositivo. El autor no se responsabiliza de daños al hardware, otras pérdidas o consecuencias derivadas de un montaje o uso incorrectos, ni garantiza efectos concretos sobre la salud.

## Ejecutar la aplicación

Ejecuta `Zapper.exe` desde la carpeta de la versión portable. Las personas persistentes y sus identificadores se guardan en `data/persons.json`, los perfiles activos en `data/profiles.json` y cada ejecución tiene su propio archivo en `data/progress/`. Las ejecuciones terminadas se mueven a carpetas `data/archive/<id>/` que contienen `profile.json` y `progress.json`. La configuración de la placa se guarda en `data/device.json`, mientras que la configuración de la aplicación, incluido el idioma detectado o seleccionado, se guarda en `data/settings.json`. Las copias de seguridad permanecen en subcarpetas locales `backups/`. Todo se mantiene junto al EXE; no se escribe nada en AppData, Documentos ni en el Registro de Windows.

En la vista **Perfiles** puedes añadir personas, generar texto de contexto para IA listo para copiar y pegar JSON simplificado devuelto por un modelo de IA. Las frecuencias en este formato se indican como `frequency_hz`; la aplicación valida el perfil, muestra una vista previa y crea un nuevo `run_id` solo después de la confirmación. La ejecución activa anterior de esa persona se archiva primero.

Durante una sesión de perfil, el botón **Pausa** guarda la parte restante del paso actual y todos los pasos siguientes en el progreso local. Al reanudar se envía una secuencia abreviada al firmware sin modificar y vuelve a requerirse una confirmación física en la placa. **Detener** cancela el progreso parcial y deja la sesión completa disponible para ejecutarse de nuevo.

Las sesiones omitidas permanecen en la cola como atrasadas. Las reglas del programa definen el número de partes, las pausas dentro de una serie, el intervalo entre sesiones completas, el tiempo de espera posterior a una sesión y la compatibilidad con otros programas el mismo día. Un perfil sin sesiones atrasadas se archiva automáticamente al completar el plan, mientras que **Finalizar programa** permite cerrarlo antes.

## Idioma de la aplicación

Al iniciar, la aplicación lee el idioma de Windows/WebView2 y lo asigna a uno de los 30 idiomas compatibles. Mientras la configuración permanezca en el modo **Automático (Windows)**, la detección se realiza en cada inicio. La selección manual del idioma en el panel izquierdo se guarda en `data/settings.json` y desactiva los cambios automáticos hasta que se vuelva a seleccionar el modo automático.

El idioma de la aplicación también es el idioma predeterminado de la variante de firmware. Para sistemas de escritura que un LCD1602/HD44780 estándar no puede mostrar de forma portátil, la aplicación selecciona la variante de firmware correspondiente con texto LCD en inglés; la interfaz de escritorio sigue utilizando el idioma elegido.

## Arduino y USB

El firmware actual se encuentra en `firmware/zapper_v5/zapper_v5.ino`, y su descripción en `firmware/zapper_v5/README.md`. Después de flashear el firmware:

1. Abre la vista **Dispositivo**.
2. Selecciona el puerto COM y haz clic en **Conectar**.
3. Espera al estado **Listo**.
4. Envía la sesión de hoy o inicia un valor único en modo manual.
5. Comprueba las conexiones de la placa y después pulsa su botón físico; solo entonces se iniciará la salida.

El puerto seleccionado se recuerda en el archivo local `data/device.json`. Las sesiones de perfil guardan `device_steps` separados y exactos; una descripción como «30 kHz» sigue siendo texto legible para una persona, mientras que la placa recibe `30000000` milihercios y la duración en milisegundos.

### Idiomas del firmware LCD

El firmware 5.1.0 tiene 30 variantes de idioma independientes generadas a partir de una única base de código. Cada sketch de Arduino contiene solo un conjunto de textos LCD. Los idiomas que utilizan el alfabeto latino tienen sus propios textos breves almacenados como ASCII seguro. Para cirílico y otros sistemas de escritura que un LCD1602/HD44780 típico no puede mostrar de forma portátil, la variante correspondiente utiliza una interfaz LCD en inglés. La lista completa está disponible en `firmware/LANGUAGES.md`.

El comando `go run ./tools/firmware_i18n` genera todos los sketches en `build/generated/firmware/`. El proceso normal `build.ps1` lo hace automáticamente e incluye las variantes en la versión portable.

### Flashear firmware desde la aplicación

La sección **Dispositivo → Firmware** muestra la versión detectada, la última versión, el idioma de la variante de firmware y el idioma LCD. El usuario elige el bootloader nuevo o antiguo del Arduino Nano y hace clic explícitamente en **Flashear firmware**; la aplicación nunca flashea la placa automáticamente al iniciarse.

La compilación y la carga se realizan con `arduino-cli`. Zapper lo busca en `tools/arduino-cli/`, junto al EXE, en `PATH` y en ubicaciones habituales de Arduino IDE. Si la herramienta no está disponible, la aplicación lo indica claramente y el botón de flasheo permanece desactivado. La compilación también requiere que el core `arduino:avr` y la biblioteca `LiquidCrystal_I2C` estén disponibles para la instalación de `arduino-cli` utilizada.

### Detección de idioma y selección de firmware

Al iniciar, la aplicación lee el idioma del entorno WebView2/Windows (`navigator.languages`) y lo asigna a uno de los 30 códigos compatibles. Si el idioma del sistema no está soportado, se selecciona inglés. En el modo **Automático (Windows)** el idioma se comprueba en cada inicio; una selección manual se guarda en `data/settings.json` hasta que se vuelva a activar el modo automático.

El mismo código de idioma es la opción predeterminada en la pantalla de flasheo de firmware. Para los idiomas no compatibles con LCD1602, la aplicación sigue seleccionando la variante identificada por el idioma del usuario, pero informa de que el texto del LCD estará en inglés. El firmware nunca se flashea automáticamente al iniciar la aplicación; el flasheo requiere un clic explícito del usuario para evitar sobrescribir accidentalmente otro programa ya guardado en el Arduino.

## Compilación

Se requiere Go. La opción más sencilla es ejecutar esto en la raíz del proyecto:

```text
build.bat
```

Como alternativa, en PowerShell:

```powershell
.\build.ps1
```

El script ejecuta pruebas y análisis de código, compila `build/generated/Zapper-dev.exe` y prepara el portable `build/Zapper/Zapper.exe` sin ventana de consola.

## Estructura del proyecto

- `app/` — código Go, interfaz HTML/CSS/JS, guía y base de datos de frecuencias.
- `firmware/zapper_v5/` — firmware Arduino actual.
- `data/` — perfiles activos, progreso, archivo, configuración del dispositivo y copias de seguridad automáticas.
- `locales/` — traducciones versionadas de la interfaz y la guía, usadas en desarrollo y copiadas a las versiones publicadas.
- `build/Zapper/` — versión portable lista para copiar a otro ordenador.