package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	webview2 "github.com/jchv/go-webview2"
)

//go:embed web/* instrukcja.html assets/*.svg assets/*.json assets/*.png
var applicationAssets embed.FS

func main() {
	runtime.LockOSThread()
	appDirectory, err := executableDirectory()
	if err != nil {
		log.Fatal(err)
	}
	application, err := NewApplication(appDirectory)
	if err != nil {
		log.Fatal(err)
	}
	errorSink := newBackgroundErrorSink(filepath.Join(appDirectory, "data", "errors.log"))
	// Ostrzeżenia z wczytywania powstały, zanim istniało okno — sink trzyma je
	// w kolejce i wypycha do banera zaraz po attach.
	for _, warning := range application.LoadWarnings() {
		errorSink.report("ostrzeżenie konfiguracji", errors.New(warning))
	}
	device := NewDeviceManager(appDirectory, func(sessionID string) {
		// Sesja ręczna ma syntetyczne ID (manual_…) — jej ślad zapisano przy
		// starcie (RecordManualRun), planu do ukończenia nie ma.
		if strings.HasPrefix(sessionID, "manual_") {
			return
		}
		// Zabieg FIZYCZNIE się odbył — płytka zgłosiła DONE. Błędu zapisu nie wolno
		// połknąć: bez wpisu w dzienniku zabieg znika, a odstęp po nim nie zaczyna biec.
		if _, err := application.CompleteDeviceSession(sessionID); err != nil {
			errorSink.report(fmt.Sprintf("NIE ZAPISANO ukończonego zabiegu (sesja %s) — zabieg się odbył, ale nie ma go w dzienniku", sessionID), err)
		}
	})
	device.onSessionCancelled = func(sessionID string) {
		if strings.HasPrefix(sessionID, "manual_") {
			return
		}
		application.DiscardPausedSession(sessionID)
	}
	defer device.Disconnect()

	server, address, err := startAssetServer(appDirectory)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	savedWindow := loadWindowState(appDirectory)
	window := webview2.NewWithOptions(webview2.WebViewOptions{
		// Debug: true wlacza w WebView2 narzedzia deweloperskie (F12) oraz
		// domyslne menu kontekstowe (prawy przycisk myszy -> Inspect).
		// Bez tego kazdy blad JS w oknie aplikacji jest calkowicie niewidoczny.
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Zapper — plan i dziennik",
			Width:  uint(savedWindow.Width),
			Height: uint(savedWindow.Height),
			Center: true,
		},
	})
	if window == nil {
		log.Fatal("nie udało się uruchomić WebView2")
	}
	defer window.Destroy()

	// Jedno natywne okno przez cały start. Zamiast chować/pokazywać HWND
	// (co na Windows potrafi zepchnąć aplikację niżej w stosie okien), WebView
	// od razu pokazuje lekki ekran startowy o tych samych kolorach co aplikacja.
	window.SetHtml(`<!doctype html><html><head><meta charset="utf-8"><meta name="color-scheme" content="light"><style>
		html,body{width:100%;height:100%;margin:0;background:#17231f;color:#fff;font-family:"Segoe UI Variable","Segoe UI",system-ui,sans-serif;overflow:hidden}
		body{display:grid;place-items:center}.boot{display:grid;justify-items:center;gap:16px}.mark{display:grid;place-items:center;width:54px;height:54px;border-radius:11px;background:#20815f;font-size:24px;font-weight:800}.text{color:#afbeb8;font-size:15px}
	</style></head><body><div class="boot"><div class="mark">Z</div><div class="text">Otwieranie dziennika…</div></div></body></html>`)

	errorSink.attach(window)
	window.SetSize(1060, 680, webview2.HintMin)
	setNativeWindowIcon(window.Window(), filepath.Join(appDirectory, "Zapper.ico"))
	setNativeWindowMaximized(window.Window(), savedWindow.Maximized)
	windowStateStop := make(chan struct{})
	go rememberWindowState(appDirectory, window.Window(), windowStateStop)

	mustBind(window, "apiLoad", func() Snapshot {
		return application.Snapshot()
	})
	mustBind(window, "apiLoadSettings", func() (AppSettings, error) {
		return loadAppSettings(appDirectory)
	})
	mustBind(window, "apiSetLanguage", func(language string, source string) (AppSettings, error) {
		return saveAppSettings(appDirectory, AppSettings{Language: language, LanguageSource: source})
	})
	mustBind(window, "apiCheckAppUpdate", func() (AppUpdateInfo, error) {
		return checkAppUpdate(appDirectory)
	})
	mustBind(window, "apiInstallAppUpdate", func() (AppUpdateInstallResult, error) {
		status := device.Status()
		if status.ActiveSessionID != "" || status.State == "starting" || status.State == "armed" || status.State == "running" || status.State == "stopping" {
			return AppUpdateInstallResult{}, errors.New("najpierw zakończ lub zatrzymaj bieżącą sesję przed aktualizacją aplikacji")
		}
		result, updateErr := installLatestAppUpdate(appDirectory)
		if updateErr != nil {
			return result, updateErr
		}
		go func() {
			time.Sleep(900 * time.Millisecond)
			// WebView2 Terminate() wysyła WM_QUIT do kolejki WĄTKU,
			// z którego zostało wywołane. Wywołanie go bezpośrednio z tej
			// gorutyny zostawiało główną pętlę okna uruchomioną, a instalator
			// czekał bez końca na zamknięcie Zapper.exe. Dispatch wykonuje
			// Terminate na właściwym wątku UI.
			terminateWindowOnUIThread(window)
		}()
		return result, nil
	})
	mustBind(window, "apiFirmwareFlashInfo", func(language string) FirmwareFlashInfo {
		return firmwareFlashInfo(appDirectory, language)
	})
	mustBind(window, "apiFlashFirmware", func(request FirmwareFlashRequest) (FirmwareFlashResult, error) {
		return flashFirmware(appDirectory, request)
	})
	mustBind(window, "apiSaveConfig", application.SaveConfig)
	mustBind(window, "apiSetDone", application.SetDone)
	mustBind(window, "apiSetSessionDone", application.SetSessionDone)
	mustBind(window, "apiFinishProfile", application.FinishProfile)
	mustBind(window, "apiDeleteProfile", application.DeleteProfile)
	mustBind(window, "apiSetStartDate", application.SetStartDate)
	mustBind(window, "apiResetProgress", application.ResetProgress)
	mustBind(window, "apiAddPerson", application.AddPerson)
	mustBind(window, "apiUpdatePerson", application.UpdatePerson)
	mustBind(window, "apiDeletePerson", application.DeletePerson)
	mustBind(window, "apiCreateProfileForPerson", application.CreateProfileForPerson)
	mustBind(window, "apiGenerateAIContext", application.GenerateAIContext)
	mustBind(window, "apiPreviewAIProfile", application.PreviewAIProfile)
	mustBind(window, "apiApplyAIProfile", application.ApplyAIProfile)
	mustBind(window, "apiCancelPausedSession", application.CancelPausedSession)
	mustBind(window, "apiDismissOverdueSessions", application.DismissOverdueSessions)
	mustBind(window, "apiDismissSessionGroup", application.DismissSessionGroup)
	mustBind(window, "apiDevicePorts", device.ListPorts)
	mustBind(window, "apiDeviceConnect", device.Connect)
	mustBind(window, "apiDeviceDisconnect", func() (DeviceStatus, error) {
		err := device.Disconnect()
		return device.Status(), err
	})
	mustBind(window, "apiDeviceStatus", device.Status)
	mustBind(window, "apiDeviceStartSession", func(sessionID string) (DeviceStatus, error) {
		plan, planErr := application.TodayPlanBySession(sessionID)
		if planErr != nil {
			return device.Status(), planErr
		}
		if plan.Done {
			return device.Status(), fmt.Errorf("ta sesja jest już wykonana")
		}
		if !plan.Available {
			return device.Status(), fmt.Errorf("sesja jest jeszcze niedostępna: %s", plan.BlockedReason)
		}
		return device.Start(plan.SessionID, plan.ProfileName, plan.DeviceSteps)
	})
	mustBind(window, "apiDeviceStartManual", func(frequencyMilliHz uint64, durationSeconds uint32) (DeviceStatus, error) {
		status, startErr := device.Start("", "", []DeviceStep{{FrequencyMilliHz: frequencyMilliHz, DurationSeconds: durationSeconds}})
		if startErr != nil {
			return status, startErr
		}
		// Tryb ręczny zostaje bez kontroli odstępów (świadoma furtka użytkownika),
		// ale musi zostawić ślad — inaczej przebiegu nie widać w historii, a odstępów
		// od niego nie da się policzyć uczciwie.
		if recordErr := application.RecordManualRun(frequencyMilliHz, durationSeconds); recordErr != nil {
			errorSink.report("nie zapisano śladu ręcznego uruchomienia", recordErr)
		}
		return status, nil
	})
	// Zatrzymaj: sesja profilu → zapis postępu (ile zrobiono, ile zostało) i wpis
	// w historii, ale ekran sesji zamyka się. Tryb ręczny zatrzymuje się po prostu
	// (bez komunikatu "zapisano postęp") — pauza z wznowieniem zostaje porzucona.
	mustBind(window, "apiDeviceStop", func() (DevicePauseResult, error) {
		wasManual := strings.HasPrefix(device.Status().ActiveSessionID, "manual_")
		pause, status, pauseErr := device.Pause()
		if pauseErr != nil {
			stopped, stopErr := device.Stop()
			if stopErr != nil {
				return DevicePauseResult{Status: stopped}, stopErr
			}
			return DevicePauseResult{Status: stopped}, nil
		}
		if wasManual {
			device.ClearManualPause()
			return DevicePauseResult{Status: status}, nil
		}
		snapshot, saveErr := application.SavePausedSession(pause, true)
		if saveErr != nil {
			return DevicePauseResult{Status: status, Pause: &pause}, saveErr
		}
		return DevicePauseResult{Status: status, Snapshot: &snapshot, Pause: &pause}, nil
	})
	mustBind(window, "apiDevicePause", func() (DevicePauseResult, error) {
		wasManual := strings.HasPrefix(device.Status().ActiveSessionID, "manual_")
		pause, status, pauseErr := device.Pause()
		if pauseErr != nil {
			// Tryb ręczny nie ma identyfikatora sesji — pauza sprowadza się do
			// zatrzymania płytki: bez zapisu postępu, bez ekranu wznowienia.
			stopped, stopErr := device.Stop()
			if stopErr != nil {
				return DevicePauseResult{Status: stopped}, stopErr
			}
			return DevicePauseResult{Status: stopped}, nil
		}
		if wasManual {
			// Wstrzymana sesja ręczna: postęp trzyma menedżer urządzenia,
			// nie dziennik — wznowienie przez apiDeviceResumeManual.
			return DevicePauseResult{Status: status, Pause: &pause}, nil
		}
		snapshot, saveErr := application.SavePausedSession(pause, false)
		if saveErr != nil {
			return DevicePauseResult{Status: status, Pause: &pause}, saveErr
		}
		return DevicePauseResult{Status: status, Snapshot: &snapshot, Pause: &pause}, nil
	})
	mustBind(window, "apiDeviceResumeManual", func(restart bool) (DeviceStatus, error) {
		steps, ok := device.ManualResumeSteps(restart)
		if !ok {
			return device.Status(), errors.New("brak wstrzymanej sesji ręcznej")
		}
		return device.Start("", "", steps)
	})
	// Błędy JS (wyjątki renderowania, odrzucone promisy) lądują w data/errors.log
	// obok błędów Go — dotąd znaliśmy je tylko z banera, który użytkownik mógł przeoczyć.
	// UWAGA: tylko zapis do pliku, bez banera — errorSink.report wraca do JS przez
	// baner i generował nieskończoną pętlę raportów.
	mustBind(window, "apiReportClientError", func(context string, message string) {
		errorSink.appendToFile("błąd JS: " + context + ": " + message)
	})

	if startView := os.Getenv("ZAPPER_START_VIEW"); startView != "" {
		address += "?view=" + url.QueryEscape(startView)
	}
	window.Navigate(address)
	window.Run()
	close(windowStateStop)
}

type dispatchingWindow interface {
	Dispatch(func())
	Terminate()
}

func terminateWindowOnUIThread(window dispatchingWindow) {
	window.Dispatch(window.Terminate)
}

func executableDirectory() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executable), nil
}

func startAssetServer(appDirectory string) (*http.Server, string, error) {
	webRoot, err := fs.Sub(applicationAssets, "web")
	if err != nil {
		return nil, "", err
	}
	assetRoot, err := fs.Sub(applicationAssets, "assets")
	if err != nil {
		return nil, "", err
	}
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetRoot))))
	serveLocale := func(kind string) http.HandlerFunc {
		return func(response http.ResponseWriter, request *http.Request) {
			code := strings.TrimPrefix(request.URL.Path, "/locale/"+kind+"/")
			if code == "" || strings.ContainsAny(code, `/\\.`) {
				http.Error(response, "nieprawidłowy kod języka", http.StatusBadRequest)
				return
			}
			for _, char := range code {
				if char < 'a' || char > 'z' {
					http.Error(response, "nieprawidłowy kod języka", http.StatusBadRequest)
					return
				}
			}
			fileName := kind + "." + code + ".json"
			candidates := []string{
				filepath.Join(appDirectory, "locales", fileName),
			}
			for _, candidate := range candidates {
				data, readErr := os.ReadFile(candidate)
				if readErr != nil {
					if os.IsNotExist(readErr) {
						continue
					}
					http.Error(response, readErr.Error(), http.StatusInternalServerError)
					return
				}
				response.Header().Set("Content-Type", "application/json; charset=utf-8")
				response.Header().Set("Cache-Control", "no-store")
				_, _ = response.Write(data)
				return
			}
			http.Error(response, "brak pakietu językowego: "+fileName, http.StatusNotFound)
		}
	}
	mux.HandleFunc("/locale/ui/", serveLocale("ui"))
	mux.HandleFunc("/locale/guide/", serveLocale("guide"))
	mux.HandleFunc("/firmware/current", func(response http.ResponseWriter, request *http.Request) {
		firmwarePath := filepath.Join(appDirectory, "firmware", "zapper_v5", "zapper_v5.ino")
		data, readErr := os.ReadFile(firmwarePath)
		if readErr != nil {
			http.Error(response, "nie znaleziono aktualnego firmware: "+readErr.Error(), http.StatusNotFound)
			return
		}
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = response.Write(data)
	})
	mux.Handle("/", http.FileServer(http.FS(webRoot)))
	mux.HandleFunc("/guide", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, readErr := applicationAssets.ReadFile("instrukcja.html")
		if readErr != nil {
			http.Error(response, readErr.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = response.Write(data)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			log.Printf("serwer interfejsu: %v", serveErr)
		}
	}()
	return server, fmt.Sprintf("http://%s/", listener.Addr().String()), nil
}

func mustBind(window webview2.WebView, name string, function any) {
	if err := window.Bind(name, function); err != nil {
		log.Fatalf("nie można podłączyć funkcji %s: %v", name, err)
	}
}
