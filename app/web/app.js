// ---------------------------------------------------------------------------
// DIAGNOSTYKA — musi byc PIERWSZA rzecza w pliku.
//
// WebView2 nie ma widocznej konsoli, wiec kazdy blad JS byl dotad calkowicie
// niewidoczny: przycisk po prostu "nie dzialal". Ponizszy kod pokazuje kazdy
// nieobsluzony blad jako czerwony baner.
//
// WAZNE: baner musi trafic do TOP LAYER przegladarki. Zwykly element (nawet z
// z-index: 2147483647) jest rysowany POD oknem dialogowym otwartym przez
// showModal() i pod jego ::backdrop. Dlatego uzywamy Popover API, ktore
// umieszcza element w top layer nad modalem. Gdyby runtime byl za stary,
// schodzimy do position: fixed (lepsze to niz nic).
// ---------------------------------------------------------------------------
(function setupDiagnostics() {
  var panel = null;
  var list = null;

  function supportsPopover(element) {
    return typeof element.showPopover === "function" && typeof HTMLElement.prototype.togglePopover === "function";
  }

  function build() {
    if (panel && panel.isConnected) return true;
    if (!document.body) return false;
    panel = document.createElement("div");
    panel.id = "zapper-diagnostics";
    panel.style.cssText = [
      "position: fixed",
      "inset: 0 0 auto 0",
      "margin: 0",
      "max-height: 60vh",
      "overflow: auto",
      "padding: 0",
      "border: 0",
      "border-bottom: 3px solid #5a1a16",
      "background: #8a1f19",
      "color: #fff",
      "font: 13px/1.45 Consolas, 'Courier New', monospace",
      "z-index: 2147483647",
      "-webkit-user-select: text",
      "user-select: text",
      "box-shadow: 0 10px 30px rgba(0,0,0,.45)",
    ].join(";");

    var header = document.createElement("div");
    header.style.cssText = "display:flex;align-items:center;justify-content:space-between;gap:12px;padding:8px 12px;background:#5a1a16;font-weight:700";
    var title = document.createElement("span");
    title.textContent = "Blad aplikacji (skopiuj i przeslij)";
    var close = document.createElement("button");
    close.type = "button";
    close.textContent = "Zamknij";
    close.style.cssText = "border:1px solid #fff;border-radius:6px;background:transparent;color:#fff;cursor:pointer;padding:4px 12px;font:inherit";
    close.addEventListener("click", function () {
      if (supportsPopover(panel)) {
        try { panel.hidePopover(); } catch (ignored) { panel.style.display = "none"; }
      } else {
        panel.style.display = "none";
      }
    });
    header.appendChild(title);
    header.appendChild(close);

    list = document.createElement("div");
    list.style.cssText = "padding:10px 12px;white-space:pre-wrap;word-break:break-word;-webkit-user-select:text;user-select:text";

    panel.appendChild(header);
    panel.appendChild(list);

    if (supportsPopover(panel)) panel.setAttribute("popover", "manual");
    document.body.appendChild(panel);
    return true;
  }

  var pending = [];

  function flush() {
    if (!build()) return;
    while (pending.length) {
      var entry = document.createElement("div");
      entry.style.cssText = "padding:6px 0;border-top:1px solid rgba(255,255,255,.28)";
      entry.textContent = pending.shift();
      list.appendChild(entry);
    }
    panel.style.display = "";
    // Pokazujemy ponownie, zeby baner trafil na SAM WIERZCH top layer nawet gdy
    // modal <dialog> zostal otwarty pozniej.
    if (supportsPopover(panel)) {
      try { panel.hidePopover(); } catch (ignored) { /* nie byl otwarty */ }
      try { panel.showPopover(); } catch (ignored) { panel.style.display = "block"; }
    }
  }

  function describe(value) {
    if (value === null || value === undefined) return String(value);
    if (typeof value === "string") return value;
    if (value instanceof Error) {
      return (value.name || "Error") + ": " + (value.message || "") + (value.stack ? "\n" + value.stack : "");
    }
    try { return JSON.stringify(value); } catch (ignored) { return String(value); }
  }

  // Eksponowane globalnie, zeby try/catch w reszcie pliku mogl raportowac tu samo.
  window.zapperReportError = function (context, error, extra) {
    var text = "[" + new Date().toISOString() + "] " + context + "\n" + describe(error);
    if (extra) text += "\n" + describe(extra);
    pending.push(text);
    flush();
    try {
      if (typeof window.apiReportClientError === "function") {
        window.apiReportClientError(String(context), text);
      }
    } catch (ignored) {}
  };

  // Diagnostyka: w konsoli (F12) wpisz __dump() — pełny stan aplikacji trafi
  // do data/errors.log, zeby mozna bylo zobaczyc co dokladnie zniknelo.
  window.__dump = function () {
    try {
      if (typeof window.apiReportClientError !== "function") return "brak apiReportClientError";
      var dump = JSON.stringify({
        currentView: typeof currentView !== "undefined" ? currentView : null,
        deviceStatus: typeof deviceStatus !== "undefined" ? deviceStatus : null,
        snapshot: typeof snapshot !== "undefined" && snapshot ? {
          today: snapshot.today || [],
          schedule: snapshot.schedule || [],
          config: snapshot.config || null,
          progress: snapshot.progress || null,
          persons: snapshot.persons || [],
          manual_runs: snapshot.manual_runs || []
        } : null
      });
      window.apiReportClientError("DUMP stanu aplikacji", dump);
      return "zrzut zapisany do data/errors.log";
    } catch (error) {
      return String(error);
    }
  };

  window.onerror = function (message, source, lineNumber, columnNumber, error) {
    window.zapperReportError(
      "window.onerror @ " + (source || "?") + ":" + lineNumber + ":" + columnNumber,
      error || message
    );
    return false;
  };

  window.addEventListener("error", function (event) {
    // Bledy ladowania zasobow (np. brak app.css) nie trafiaja do window.onerror.
    if (event.target && event.target !== window && event.target.tagName) {
      window.zapperReportError("nie udalo sie zaladowac zasobu", event.target.outerHTML || event.target.tagName);
    }
  }, true);

  window.addEventListener("unhandledrejection", function (event) {
    window.zapperReportError("unhandledrejection", event.reason);
  });

  document.addEventListener("DOMContentLoaded", function () {
    if (pending.length) flush();
  });
})();

// Owija funkcje tak, by kazdy jej blad byl WIDOCZNY zamiast cicho zabijac dalszy kod.
function guarded(context, fn) {
  return function () {
    try {
      var result = fn.apply(this, arguments);
      if (result && typeof result.catch === "function") {
        return result.catch(function (error) {
          window.zapperReportError(context, error);
          throw error;
        });
      }
      return result;
    } catch (error) {
      window.zapperReportError(context, error);
      throw error;
    }
  };
}

const WEEKDAYS = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
const WEEKDAY_SAMPLE_DATES = {
  Monday: new Date(Date.UTC(2026, 7, 31)), Tuesday: new Date(Date.UTC(2026, 8, 1)),
  Wednesday: new Date(Date.UTC(2026, 8, 2)), Thursday: new Date(Date.UTC(2026, 8, 3)),
  Friday: new Date(Date.UTC(2026, 8, 4)), Saturday: new Date(Date.UTC(2026, 8, 5)),
  Sunday: new Date(Date.UTC(2026, 8, 6))
};

function localizedWeekdayName(key) {
  const locale = window.ZapperI18n?.locale || LANGUAGE_LOCALES[preferredLanguage] || "en-US";
  const date = WEEKDAY_SAMPLE_DATES[key] || WEEKDAY_SAMPLE_DATES.Monday;
  return new Intl.DateTimeFormat(locale, { weekday: "long", timeZone: "UTC" }).format(date);
}

function uiText(key, fallback = key) {
  const value = window.ZapperI18n?.t(key);
  return value && value !== key ? value : fallback;
}

function uiFormat(key, values = {}, fallback = key) {
  let value = uiText(key, fallback);
  for (const [name, replacement] of Object.entries(values)) {
    value = value.split(`{${name}}`).join(String(replacement ?? ""));
  }
  return value;
}

function localizedSource(source) {
  return window.ZapperI18n?.translateSource(source) || source;
}

function localizedBlockedReason(reason) {
  return localizedSource(String(reason || ""));
}

function localizedTodaySummary(total, completed, overdue, remainingSeconds) {
  const values = { total, completed, overdue, remaining: formatDuration(remainingSeconds) };
  return uiFormat(overdue ? "dynTodaySummaryOverdue" : "dynTodaySummary", values);
}

function localizedPlannedContext(plan) {
  const values = { date: formatShortDate(plan.planned_date), days: Number(plan.extension_days || 0) };
  return uiFormat(plan.extension_days ? "dynPlannedDateExtended" : "dynPlannedDate", values);
}

function localizedOlderWarning(count) {
  return count ? uiFormat("dynOlderAppointmentsRemain", { count }) : "";
}

function scheduleCountdownValues(availableAt, now = Date.now()) {
  const target = Date.parse(String(availableAt || ""));
  if (!Number.isFinite(target)) return null;
  const locale = window.ZapperI18n?.locale || LANGUAGE_LOCALES[preferredLanguage] || "en-US";
  return {
    target,
    seconds: Math.max(0, Math.ceil((target - now) / 1000)),
    time: new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit" }).format(new Date(target))
  };
}

function scheduleWaitingText(plan, now = Date.now()) {
  const reason = localizedBlockedReason(plan?.blocked_reason || uiText("notAvailableYet"));
  const countdown = scheduleCountdownValues(plan?.available_at, now);
  if (!countdown) return reason;
  return `${reason} · ${countdown.time} · ${uiText("remaining")} ${formatClock(countdown.seconds)}`;
}

function scheduleWaitingMarkup(plan) {
  const text = scheduleWaitingText(plan);
  if (!plan?.available_at) return escapeHTML(text);
  return `<span class="schedule-countdown" data-schedule-countdown data-available-at="${escapeAttribute(plan.available_at)}" data-blocked-reason="${escapeAttribute(plan.blocked_reason || uiText("notAvailableYet"))}">${escapeHTML(text)}</span>`;
}

async function tickScheduleCountdowns() {
  if (!snapshot) return;
  const now = Date.now();
  let expired = false;
  document.querySelectorAll("[data-schedule-countdown]").forEach(element => {
    const values = scheduleCountdownValues(element.dataset.availableAt, now);
    if (!values) return;
    const plan = { available_at: element.dataset.availableAt, blocked_reason: element.dataset.blockedReason };
    element.textContent = scheduleWaitingText(plan, now);
    if (values.seconds <= 0) expired = true;
  });
  if (!expired || scheduleRefreshInFlight || now < nextScheduleRefreshAt) return;

  scheduleRefreshInFlight = true;
  nextScheduleRefreshAt = now + 1500;
  try {
    // Dostępność jest wyliczana z zapisanej godziny ukończenia przez backend.
    // Ponowne pobranie usuwa licznik i odblokowuje przycisk dokładnie po przerwie.
    snapshot = await window.apiLoad();
    renderToday();
    renderSchedule();
    renderDeviceSessions();
    renderDeviceStatus();
  } catch (error) {
    window.zapperReportError("tickScheduleCountdowns", error);
  } finally {
    scheduleRefreshInFlight = false;
  }
}

const SUPPORTED_LANGUAGES = new Set([
  "en", "pl", "de", "fr", "es", "it", "pt", "nl", "sv", "no",
  "da", "fi", "cs", "sk", "hu", "ro", "tr", "id", "ms", "vi",
  "ru", "uk", "bg", "el", "ar", "he", "hi", "zh", "ja", "ko"
]);
const LANGUAGE_LOCALES = {
  en: "en-US", pl: "pl-PL", de: "de-DE", fr: "fr-FR", es: "es-ES", it: "it-IT",
  pt: "pt-PT", nl: "nl-NL", sv: "sv-SE", no: "nb-NO", da: "da-DK", fi: "fi-FI",
  cs: "cs-CZ", sk: "sk-SK", hu: "hu-HU", ro: "ro-RO", tr: "tr-TR", id: "id-ID",
  ms: "ms-MY", vi: "vi-VN", ru: "ru-RU", uk: "uk-UA", bg: "bg-BG", el: "el-GR",
  ar: "ar", he: "he-IL", hi: "hi-IN", zh: "zh-CN", ja: "ja-JP", ko: "ko-KR"
};
const FIRMWARE_LCD_ENGLISH_FALLBACK = new Set(["ru", "uk", "bg", "el", "ar", "he", "hi", "zh", "ja", "ko"]);

const UPDATE_TEXT = {
  en: { title: "New Zapper version", message: "Version {latest} is available. You are using {current}.", install: "Download and install", later: "Later", checking: "Checking for updates…", upToDate: "Zapper is up to date.", downloading: "Downloading and verifying the update…", restarting: "Update ready. Zapper will restart.", portableOnly: "A new version is available on GitHub. Automatic installation is available in the portable build." },
  pl: { title: "Nowa wersja Zapper", message: "Dostępna jest wersja {latest}. Używasz {current}.", install: "Pobierz i zainstaluj", later: "Później", checking: "Sprawdzanie aktualizacji…", upToDate: "Zapper jest aktualny.", downloading: "Pobieranie i sprawdzanie aktualizacji…", restarting: "Aktualizacja gotowa. Zapper uruchomi się ponownie.", portableOnly: "Nowa wersja jest dostępna na GitHubie. Automatyczna instalacja działa w wersji portable." },
  de: { title: "Neue Zapper-Version", message: "Version {latest} ist verfügbar. Installiert ist {current}.", install: "Herunterladen und installieren", later: "Später", checking: "Nach Updates suchen…", upToDate: "Zapper ist aktuell.", downloading: "Update wird heruntergeladen und geprüft…", restarting: "Update ist bereit. Zapper wird neu gestartet.", portableOnly: "Eine neue Version ist auf GitHub verfügbar. Die automatische Installation ist in der portablen Version verfügbar." },
  fr: { title: "Nouvelle version de Zapper", message: "La version {latest} est disponible. Vous utilisez {current}.", install: "Télécharger et installer", later: "Plus tard", checking: "Recherche de mises à jour…", upToDate: "Zapper est à jour.", downloading: "Téléchargement et vérification de la mise à jour…", restarting: "Mise à jour prête. Zapper va redémarrer.", portableOnly: "Une nouvelle version est disponible sur GitHub. L’installation automatique est disponible dans la version portable." },
  es: { title: "Nueva versión de Zapper", message: "Está disponible la versión {latest}. Estás usando {current}.", install: "Descargar e instalar", later: "Más tarde", checking: "Buscando actualizaciones…", upToDate: "Zapper está actualizado.", downloading: "Descargando y verificando la actualización…", restarting: "Actualización lista. Zapper se reiniciará.", portableOnly: "Hay una nueva versión en GitHub. La instalación automática está disponible en la versión portable." },
  it: { title: "Nuova versione di Zapper", message: "È disponibile la versione {latest}. Stai usando {current}.", install: "Scarica e installa", later: "Più tardi", checking: "Ricerca aggiornamenti…", upToDate: "Zapper è aggiornato.", downloading: "Download e verifica dell’aggiornamento…", restarting: "Aggiornamento pronto. Zapper verrà riavviato.", portableOnly: "Una nuova versione è disponibile su GitHub. L’installazione automatica è disponibile nella versione portable." },
  pt: { title: "Nova versão do Zapper", message: "A versão {latest} está disponível. Está a usar {current}.", install: "Transferir e instalar", later: "Mais tarde", checking: "A procurar atualizações…", upToDate: "O Zapper está atualizado.", downloading: "A transferir e verificar a atualização…", restarting: "Atualização pronta. O Zapper será reiniciado.", portableOnly: "Está disponível uma nova versão no GitHub. A instalação automática está disponível na versão portable." },
  nl: { title: "Nieuwe Zapper-versie", message: "Versie {latest} is beschikbaar. Je gebruikt {current}.", install: "Downloaden en installeren", later: "Later", checking: "Controleren op updates…", upToDate: "Zapper is up-to-date.", downloading: "Update downloaden en controleren…", restarting: "Update gereed. Zapper wordt opnieuw gestart.", portableOnly: "Er is een nieuwe versie op GitHub. Automatische installatie is beschikbaar in de portable versie." },
  sv: { title: "Ny Zapper-version", message: "Version {latest} är tillgänglig. Du använder {current}.", install: "Hämta och installera", later: "Senare", checking: "Söker efter uppdateringar…", upToDate: "Zapper är uppdaterad.", downloading: "Hämtar och verifierar uppdateringen…", restarting: "Uppdateringen är klar. Zapper startas om.", portableOnly: "En ny version finns på GitHub. Automatisk installation finns i den portabla versionen." },
  no: { title: "Ny Zapper-versjon", message: "Versjon {latest} er tilgjengelig. Du bruker {current}.", install: "Last ned og installer", later: "Senere", checking: "Ser etter oppdateringer…", upToDate: "Zapper er oppdatert.", downloading: "Laster ned og kontrollerer oppdateringen…", restarting: "Oppdateringen er klar. Zapper starter på nytt.", portableOnly: "En ny versjon er tilgjengelig på GitHub. Automatisk installasjon er tilgjengelig i den portable versjonen." },
  da: { title: "Ny Zapper-version", message: "Version {latest} er tilgængelig. Du bruger {current}.", install: "Download og installer", later: "Senere", checking: "Søger efter opdateringer…", upToDate: "Zapper er opdateret.", downloading: "Downloader og kontrollerer opdateringen…", restarting: "Opdateringen er klar. Zapper genstarter.", portableOnly: "En ny version er tilgængelig på GitHub. Automatisk installation er tilgængelig i den portable version." },
  fi: { title: "Uusi Zapper-versio", message: "Versio {latest} on saatavilla. Käytössäsi on {current}.", install: "Lataa ja asenna", later: "Myöhemmin", checking: "Tarkistetaan päivityksiä…", upToDate: "Zapper on ajan tasalla.", downloading: "Ladataan ja tarkistetaan päivitystä…", restarting: "Päivitys on valmis. Zapper käynnistyy uudelleen.", portableOnly: "Uusi versio on saatavilla GitHubissa. Automaattinen asennus toimii portable-versiossa." },
  cs: { title: "Nová verze Zapper", message: "Je dostupná verze {latest}. Používáte {current}.", install: "Stáhnout a nainstalovat", later: "Později", checking: "Kontrola aktualizací…", upToDate: "Zapper je aktuální.", downloading: "Stahování a ověřování aktualizace…", restarting: "Aktualizace je připravena. Zapper se restartuje.", portableOnly: "Na GitHubu je dostupná nová verze. Automatická instalace je dostupná v portable verzi." },
  sk: { title: "Nová verzia Zapper", message: "Je dostupná verzia {latest}. Používate {current}.", install: "Stiahnuť a nainštalovať", later: "Neskôr", checking: "Kontrola aktualizácií…", upToDate: "Zapper je aktuálny.", downloading: "Sťahovanie a overovanie aktualizácie…", restarting: "Aktualizácia je pripravená. Zapper sa reštartuje.", portableOnly: "Na GitHube je dostupná nová verzia. Automatická inštalácia je dostupná v portable verzii." },
  hu: { title: "Új Zapper-verzió", message: "Elérhető a(z) {latest} verzió. Jelenlegi verzió: {current}.", install: "Letöltés és telepítés", later: "Később", checking: "Frissítések keresése…", upToDate: "A Zapper naprakész.", downloading: "A frissítés letöltése és ellenőrzése…", restarting: "A frissítés kész. A Zapper újraindul.", portableOnly: "Új verzió érhető el a GitHubon. Az automatikus telepítés a hordozható verzióban érhető el." },
  ro: { title: "Versiune nouă Zapper", message: "Versiunea {latest} este disponibilă. Folosiți {current}.", install: "Descarcă și instalează", later: "Mai târziu", checking: "Se caută actualizări…", upToDate: "Zapper este actualizat.", downloading: "Se descarcă și se verifică actualizarea…", restarting: "Actualizarea este pregătită. Zapper va reporni.", portableOnly: "O versiune nouă este disponibilă pe GitHub. Instalarea automată este disponibilă în versiunea portable." },
  tr: { title: "Yeni Zapper sürümü", message: "{latest} sürümü mevcut. Kullandığınız sürüm {current}.", install: "İndir ve yükle", later: "Daha sonra", checking: "Güncellemeler denetleniyor…", upToDate: "Zapper güncel.", downloading: "Güncelleme indiriliyor ve doğrulanıyor…", restarting: "Güncelleme hazır. Zapper yeniden başlatılacak.", portableOnly: "GitHub’da yeni bir sürüm var. Otomatik yükleme portable sürümde kullanılabilir." },
  id: { title: "Versi Zapper baru", message: "Versi {latest} tersedia. Anda menggunakan {current}.", install: "Unduh dan instal", later: "Nanti", checking: "Memeriksa pembaruan…", upToDate: "Zapper sudah terbaru.", downloading: "Mengunduh dan memverifikasi pembaruan…", restarting: "Pembaruan siap. Zapper akan dimulai ulang.", portableOnly: "Versi baru tersedia di GitHub. Instalasi otomatis tersedia pada versi portable." },
  ms: { title: "Versi Zapper baharu", message: "Versi {latest} tersedia. Anda menggunakan {current}.", install: "Muat turun dan pasang", later: "Kemudian", checking: "Memeriksa kemas kini…", upToDate: "Zapper sudah terkini.", downloading: "Memuat turun dan mengesahkan kemas kini…", restarting: "Kemas kini sedia. Zapper akan dimulakan semula.", portableOnly: "Versi baharu tersedia di GitHub. Pemasangan automatik tersedia dalam versi portable." },
  vi: { title: "Phiên bản Zapper mới", message: "Đã có phiên bản {latest}. Bạn đang dùng {current}.", install: "Tải xuống và cài đặt", later: "Để sau", checking: "Đang kiểm tra cập nhật…", upToDate: "Zapper đã được cập nhật.", downloading: "Đang tải xuống và xác minh bản cập nhật…", restarting: "Bản cập nhật đã sẵn sàng. Zapper sẽ khởi động lại.", portableOnly: "Có phiên bản mới trên GitHub. Cài đặt tự động khả dụng trong bản portable." },
  ru: { title: "Новая версия Zapper", message: "Доступна версия {latest}. У вас установлена {current}.", install: "Скачать и установить", later: "Позже", checking: "Проверка обновлений…", upToDate: "Zapper обновлён.", downloading: "Загрузка и проверка обновления…", restarting: "Обновление готово. Zapper будет перезапущен.", portableOnly: "На GitHub доступна новая версия. Автоматическая установка доступна в portable-версии." },
  uk: { title: "Нова версія Zapper", message: "Доступна версія {latest}. У вас встановлена {current}.", install: "Завантажити й установити", later: "Пізніше", checking: "Перевірка оновлень…", upToDate: "Zapper оновлено.", downloading: "Завантаження та перевірка оновлення…", restarting: "Оновлення готове. Zapper буде перезапущено.", portableOnly: "На GitHub доступна нова версія. Автоматичне встановлення доступне в portable-версії." },
  bg: { title: "Нова версия на Zapper", message: "Налична е версия {latest}. Използвате {current}.", install: "Изтегли и инсталирай", later: "По-късно", checking: "Проверка за актуализации…", upToDate: "Zapper е актуален.", downloading: "Изтегляне и проверка на актуализацията…", restarting: "Актуализацията е готова. Zapper ще се рестартира.", portableOnly: "В GitHub има нова версия. Автоматичното инсталиране е достъпно в portable версията." },
  el: { title: "Νέα έκδοση Zapper", message: "Η έκδοση {latest} είναι διαθέσιμη. Χρησιμοποιείτε την {current}.", install: "Λήψη και εγκατάσταση", later: "Αργότερα", checking: "Έλεγχος για ενημερώσεις…", upToDate: "Το Zapper είναι ενημερωμένο.", downloading: "Λήψη και επαλήθευση ενημέρωσης…", restarting: "Η ενημέρωση είναι έτοιμη. Το Zapper θα επανεκκινήσει.", portableOnly: "Υπάρχει νέα έκδοση στο GitHub. Η αυτόματη εγκατάσταση είναι διαθέσιμη στην portable έκδοση." },
  ar: { title: "إصدار جديد من Zapper", message: "الإصدار {latest} متاح. أنت تستخدم {current}.", install: "تنزيل وتثبيت", later: "لاحقًا", checking: "جارٍ التحقق من التحديثات…", upToDate: "Zapper محدّث.", downloading: "جارٍ تنزيل التحديث والتحقق منه…", restarting: "التحديث جاهز. سيُعاد تشغيل Zapper.", portableOnly: "يتوفر إصدار جديد على GitHub. التثبيت التلقائي متاح في النسخة المحمولة." },
  he: { title: "גרסה חדשה של Zapper", message: "גרסה {latest} זמינה. מותקנת אצלך {current}.", install: "הורדה והתקנה", later: "מאוחר יותר", checking: "בודק עדכונים…", upToDate: "Zapper מעודכן.", downloading: "מוריד ומאמת את העדכון…", restarting: "העדכון מוכן. Zapper יופעל מחדש.", portableOnly: "גרסה חדשה זמינה ב-GitHub. התקנה אוטומטית זמינה בגרסה הניידת." },
  hi: { title: "Zapper का नया संस्करण", message: "संस्करण {latest} उपलब्ध है। आप {current} उपयोग कर रहे हैं।", install: "डाउनलोड और इंस्टॉल करें", later: "बाद में", checking: "अपडेट जाँचे जा रहे हैं…", upToDate: "Zapper नवीनतम है।", downloading: "अपडेट डाउनलोड और सत्यापित किया जा रहा है…", restarting: "अपडेट तैयार है। Zapper पुनः शुरू होगा।", portableOnly: "GitHub पर नया संस्करण उपलब्ध है। स्वचालित इंस्टॉलेशन portable संस्करण में उपलब्ध है।" },
  zh: { title: "Zapper 新版本", message: "版本 {latest} 已发布。当前版本为 {current}。", install: "下载并安装", later: "稍后", checking: "正在检查更新…", upToDate: "Zapper 已是最新版本。", downloading: "正在下载并验证更新…", restarting: "更新已准备好。Zapper 将重新启动。", portableOnly: "GitHub 上有新版本。自动安装可在 portable 版本中使用。" },
  ja: { title: "Zapper の新しいバージョン", message: "バージョン {latest} が利用できます。現在は {current} です。", install: "ダウンロードしてインストール", later: "後で", checking: "更新を確認しています…", upToDate: "Zapper は最新です。", downloading: "更新をダウンロードして検証しています…", restarting: "更新の準備ができました。Zapper を再起動します。", portableOnly: "GitHub に新しいバージョンがあります。自動インストールは portable 版で利用できます。" },
  ko: { title: "새 Zapper 버전", message: "버전 {latest}을 사용할 수 있습니다. 현재 버전은 {current}입니다.", install: "다운로드 및 설치", later: "나중에", checking: "업데이트 확인 중…", upToDate: "Zapper가 최신 버전입니다.", downloading: "업데이트 다운로드 및 검증 중…", restarting: "업데이트가 준비되었습니다. Zapper를 다시 시작합니다.", portableOnly: "GitHub에 새 버전이 있습니다. 자동 설치는 portable 버전에서 사용할 수 있습니다." }
};

function updateText(key, values = {}) {
  const language = SUPPORTED_LANGUAGES.has(preferredLanguage) ? preferredLanguage : "en";
  let value = UPDATE_TEXT[language]?.[key] || UPDATE_TEXT.en[key] || key;
  for (const [name, replacement] of Object.entries(values)) {
    value = value.split(`{${name}}`).join(String(replacement ?? ""));
  }
  return value;
}

const VIEW_TITLES = {
  today: "Dzisiaj",
  device: "Urządzenie",
  therapy: "Terapia",
  schedule: "Harmonogram",
  profiles: "Profile i fazy",
  history: "Historia",
  archive: "Archiwum",
  frequencies: "Baza częstotliwości",
  guide: "Instrukcja",
};

let currentView = "today";
let activeSessionKind = null;
let appSettings = { language: "", language_source: "" };
let preferredLanguage = "en";
let firmwareFlashInfo = null;
let firmwareFlashBusy = false;
let updateCheckInFlight = false;
let updateInstallBusy = false;
let pendingUpdateInfo = null;

let snapshot = null;
let localizedFrequencies = null;
let localizedFrequencyLanguage = "";
let draftConfig = null;
let profilesDirty = false;
let selectedProfile = 0;
let selectedPhase = 0;
let selectedPersonID = "";
let editedPersonID = "";
let validatedAIJSON = "";
let lastAIContextPersons = [];
let scheduleDays = 7;
let toastTimer = null;
let armedDelete = "";
let armedDeleteTimer = null;
let pendingPersonDeletion = "";
const DESTRUCTIVE_ARM_MS = 12000;
let deviceStatus = { connected: false, ready: false, state: "disconnected", message: "Brak komunikacji" };
let devicePorts = [];
let devicePollTimer = null;
let scheduleCountdownTimer = null;
let scheduleRefreshInFlight = false;
let nextScheduleRefreshAt = 0;

document.addEventListener("DOMContentLoaded", initialize);

function normalizeBrowserLanguage(value) {
  const code = String(value || "").trim().toLowerCase().replace(/_/g, "-").split("-")[0];
  return SUPPORTED_LANGUAGES.has(code) ? code : "";
}

function detectPreferredLanguage() {
  const candidates = Array.isArray(navigator.languages) && navigator.languages.length
    ? navigator.languages
    : [navigator.language];
  for (const candidate of candidates) {
    const code = normalizeBrowserLanguage(candidate);
    if (code) return code;
  }
  return "en";
}

function firmwareLanguageFor(language) {
  const code = SUPPORTED_LANGUAGES.has(language) ? language : "en";
  return {
    code,
    lcdLanguage: FIRMWARE_LCD_ENGLISH_FALLBACK.has(code) ? "en" : code,
    englishFallback: FIRMWARE_LCD_ENGLISH_FALLBACK.has(code)
  };
}

async function loadLocalizedFrequencyCatalog(language) {
  if (!snapshot) return;
  const code = normalizeBrowserLanguage(language) || "en";
  if (code === "pl") {
    localizedFrequencies = snapshot.frequencies || [];
    localizedFrequencyLanguage = "pl";
    return;
  }
  if (localizedFrequencies && localizedFrequencyLanguage === code) return;

  // Publiczny build ma zawierać kompletny katalog dla KAŻDEGO języka. Nie
  // maskujemy braków fallbackiem na angielski lub polski: brak pliku jest błędem.
  const response = await fetch(`/assets/frequencies.${code}.json`, { cache: "no-store" });
  if (!response.ok) throw new Error(`Brak katalogu częstotliwości dla języka ${code}: HTTP ${response.status}`);
  const entries = await response.json();
  if (!Array.isArray(entries) || entries.length !== (snapshot.frequencies || []).length) {
    throw new Error(`Niepełny katalog częstotliwości ${code}`);
  }
  localizedFrequencies = entries;
  localizedFrequencyLanguage = code;
}

async function initializeLanguagePreference() {
  appSettings = await window.apiLoadSettings();
  let language = normalizeBrowserLanguage(appSettings?.language);
  // W trybie auto sprawdzamy język Windows/WebView2 przy każdym starcie. Dopiero
  // ręczny wybór w selektorze blokuje automatyczną zmianę przy kolejnych startach.
  if (!language || appSettings?.language_source !== "manual") {
    const detected = detectPreferredLanguage();
    language = detected || language || "en";
    if (appSettings?.language !== language || appSettings?.language_source !== "auto") {
      appSettings = await window.apiSetLanguage(language, "auto");
    }
  }
  preferredLanguage = language || "en";
  document.documentElement.lang = preferredLanguage;
  window.zapperPreferredLanguage = preferredLanguage;
  window.zapperFirmwareLanguage = firmwareLanguageFor(preferredLanguage);
  if (window.ZapperI18n) {
    await window.ZapperI18n.loadLanguage(preferredLanguage);
    window.ZapperI18n.setLanguage(preferredLanguage);
    populateLanguageSelector();
  }
}

function populateLanguageSelector() {
  const select = document.getElementById("language-select");
  if (!select || !window.ZapperI18n) return;
  window.ZapperI18n.populateSelect(select, preferredLanguage);
  const auto = document.createElement("option");
  auto.value = "auto";
  auto.textContent = window.ZapperI18n.t("autoWindows");
  select.prepend(auto);
  select.value = appSettings?.language_source === "manual" ? preferredLanguage : "auto";
}

function updateLocalizedDate() {
  const locale = window.ZapperI18n?.locale || LANGUAGE_LOCALES[preferredLanguage] || "en-US";
  document.getElementById("today-label").textContent = new Intl.DateTimeFormat(locale, {
    weekday: "long", day: "numeric", month: "long", year: "numeric"
  }).format(new Date());
}

function syncGuideLanguage() {
  const frame = document.getElementById("guide-frame");
  if (!frame) return;
  const language = normalizeBrowserLanguage(preferredLanguage) || "en";
  const next = `/guide?lang=${encodeURIComponent(language)}`;
  const current = frame.getAttribute("src") || "";
  if (current !== next) frame.setAttribute("src", next);
}

async function changeApplicationLanguage(event) {
  const requested = String(event?.target?.value || "auto");
  const automatic = requested === "auto";
  const language = automatic ? detectPreferredLanguage() : (normalizeBrowserLanguage(requested) || "en");
  try {
    if (window.ZapperI18n) await window.ZapperI18n.loadLanguage(language);
    appSettings = await window.apiSetLanguage(language, automatic ? "auto" : "manual");
    preferredLanguage = language;
    window.zapperPreferredLanguage = language;
    window.zapperFirmwareLanguage = firmwareLanguageFor(language);
    if (window.ZapperI18n) {
      window.ZapperI18n.setLanguage(language);
      populateLanguageSelector();
    }
    updateLocalizedDate();
    await loadLocalizedFrequencyCatalog(language);
    syncGuideLanguage();
    renderAll();
    await refreshFirmwareFlashInfo();
    if (pendingUpdateInfo && document.getElementById("update-dialog")?.open) showUpdateDialog(pendingUpdateInfo);
  } catch (error) {
    toast(normalizeError(error), true);
  }
}

async function notifyApplicationReady() {
  // Wymuszenie układu gwarantuje, że pierwsza widoczna klatka natywnego okna
  // zawiera już gotowy interfejs, a nie dokument pośredni WebView2.
  document.body.getBoundingClientRect();
  try {
    await window.apiApplicationReady();
  } catch (error) {
    window.zapperReportError("apiApplicationReady", error);
  }
}

async function initialize() {
  // Kazdy z tych kroków osobno: pojedynczy blad (np. brakujace id w HTML)
  // nie moze juz po cichu wylaczyc wszystkich pozostalych przyciskow.
  try {
    bindNavigation();
  } catch (error) {
    window.zapperReportError("bindNavigation", error);
  }
  try {
    bindStaticActions();
  } catch (error) {
    window.zapperReportError("bindStaticActions — czesc przyciskow NIE zostala podlaczona", error);
  }
  try {
    await initializeLanguagePreference();
  } catch (error) {
    preferredLanguage = "en";
    document.documentElement.lang = "en";
    window.zapperReportError("initializeLanguagePreference", error);
  }
  updateLocalizedDate();

  try {
    snapshot = await window.apiLoad();
    deviceStatus = await window.apiDeviceStatus();
    await loadLocalizedFrequencyCatalog(preferredLanguage);
    syncGuideLanguage();
    syncDraft();
    renderAll();
    const requestedView = new URLSearchParams(window.location.search).get("view");
    if (requestedView && VIEW_TITLES[requestedView]) openView(requestedView);
    syncTherapyView();
    document.getElementById("app-version").textContent = `v${snapshot.meta.version}`;
    document.getElementById("app-shell").setAttribute("aria-hidden", "false");
    document.getElementById("loading-screen").classList.add("is-hidden");
    await notifyApplicationReady();
    await refreshFirmwareFlashInfo();
    await refreshDevicePorts(false);
    devicePollTimer = setInterval(pollDeviceStatus, 750);
    scheduleCountdownTimer = setInterval(tickScheduleCountdowns, 1000);
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden) tickScheduleCountdowns();
    });
    tickScheduleCountdowns();
    setTimeout(() => checkForAppUpdate(false), 1200);
  } catch (error) {
    showFatal(error);
    await notifyApplicationReady();
  }
}

async function checkForAppUpdate(showUpToDate = false) {
  if (updateCheckInFlight || typeof window.apiCheckAppUpdate !== "function") return;
  updateCheckInFlight = true;
  if (showUpToDate) toast(updateText("checking"));
  try {
    const info = await window.apiCheckAppUpdate();
    if (info?.available) {
      showUpdateDialog(info);
    } else if (showUpToDate) {
      toast(updateText("upToDate"));
    }
  } catch (error) {
    // Brak internetu lub chwilowy błąd GitHuba nie może przeszkadzać przy starcie.
    // Błąd pokazujemy tylko wtedy, gdy użytkownik ręcznie kliknął numer wersji.
    if (showUpToDate) toast(normalizeError(error), true);
  } finally {
    updateCheckInFlight = false;
  }
}

function showUpdateDialog(info) {
  if (!info) return;
  pendingUpdateInfo = info;
  const dialog = document.getElementById("update-dialog");
  const installButton = document.getElementById("install-update");
  const laterButton = document.getElementById("later-update");
  const status = document.getElementById("update-dialog-status");
  document.getElementById("update-dialog-title").textContent = updateText("title");
  document.getElementById("update-dialog-message").textContent = updateText("message", {
    latest: info.latest_version,
    current: info.current_version
  });
  installButton.textContent = updateText("install");
  laterButton.textContent = updateText("later");
  installButton.hidden = !info.install_supported;
  installButton.disabled = updateInstallBusy;
  laterButton.disabled = updateInstallBusy;
  status.hidden = info.install_supported;
  status.textContent = info.install_supported ? "" : updateText("portableOnly");
  if (!dialog.open) dialog.showModal();
}

function closeUpdateDialog() {
  if (updateInstallBusy) return;
  const dialog = document.getElementById("update-dialog");
  if (dialog.open) dialog.close();
}

async function installAppUpdate() {
  if (updateInstallBusy || !pendingUpdateInfo?.install_supported) return;
  const installButton = document.getElementById("install-update");
  const laterButton = document.getElementById("later-update");
  const status = document.getElementById("update-dialog-status");
  updateInstallBusy = true;
  installButton.disabled = true;
  laterButton.disabled = true;
  status.hidden = false;
  status.textContent = updateText("downloading");
  try {
    await window.apiInstallAppUpdate();
    status.textContent = updateText("restarting");
  } catch (error) {
    status.textContent = normalizeError(error);
    toast(normalizeError(error), true);
    updateInstallBusy = false;
    installButton.disabled = false;
    laterButton.disabled = false;
  }
}

function bindNavigation() {
  document.querySelectorAll("[data-view]").forEach(button => {
    button.addEventListener("click", () => openView(button.dataset.view));
  });
}

function openView(name) {
  currentView = name;
  document.querySelectorAll("[data-view]").forEach(button => button.classList.toggle("is-active", button.dataset.view === name));
  document.querySelectorAll("[data-page]").forEach(page => page.classList.toggle("is-active", page.dataset.page === name));
  document.getElementById("view-title").textContent = VIEW_TITLES[name] || name;
  document.querySelector(".content").scrollTop = 0;
  if (snapshot && name === "device") renderDeviceView();
  if (snapshot && name === "therapy") renderTherapy();
}

function bindStaticActions() {
  document.getElementById("today-list").addEventListener("click", async event => {
    const dismissButton = event.target.closest("[data-dismiss-session-group]");
    if (dismissButton) {
      if (!armDestructive(dismissButton, `dismiss-group-${dismissButton.dataset.dismissSessionGroup}`, "Kliknij ponownie: odpuść ten termin")) return;
      await runAction(dismissButton, async () => {
        snapshot = await window.apiDismissSessionGroup(dismissButton.dataset.dismissSessionGroup);
        syncDraft();
        renderAll();
        toast("Termin odpuszczony — nie został zapisany jako wykonany");
      });
      return;
    }
    const button = event.target.closest("[data-session-done]");
    if (!button) return;
    if (button.dataset.outOfOrder === "true" && button.dataset.done !== "true" && !armDestructive(button, `out-of-order-done-${button.dataset.sessionDone}`, "Kliknij ponownie: wykonaj poza kolejnością")) return;
    await runAction(button, async () => {
      snapshot = await window.apiSetSessionDone(button.dataset.sessionDone, button.dataset.done !== "true");
      renderToday();
      renderHistory();
      renderSchedule();
      renderDeviceSessions();
      toast(button.dataset.done === "true" ? "Cofnięto oznaczenie sesji" : "Sesja zapisana w historii");
    });
  });

  document.getElementById("overdue-actions").addEventListener("click", async event => {
    const button = event.target.closest("[data-dismiss-overdue]");
    if (!button) return;
    if (!armDestructive(button, `dismiss-${button.dataset.dismissOverdue}`, "Kliknij ponownie: odpuść zaległości")) return;
    await runAction(button, async () => {
      snapshot = await window.apiDismissOverdueSessions(button.dataset.dismissOverdue);
      renderAll();
      toast("Zaległe sesje odpuszczone — nie zapisano ich jako wykonanych");
    });
  });

  document.getElementById("save-start-date").addEventListener("click", async event => {
    const date = document.getElementById("start-date").value;
    await runAction(event.currentTarget, async () => {
      snapshot = await window.apiSetStartDate(date);
      renderToday();
      renderSchedule();
      renderDeviceSessions();
      toast("Data rozpoczęcia została zapisana");
    });
  });

  document.getElementById("reset-progress").addEventListener("click", async event => {
    const button = event.currentTarget;
    if (!armDestructive(button, "reset-progress", "Kliknij ponownie, aby potwierdzić")) return;
    await runAction(button, async () => {
      snapshot = await window.apiResetProgress();
      renderToday();
      renderHistory();
      renderSchedule();
      renderDeviceSessions();
      toast("Postęp został zresetowany");
    });
  });

  document.getElementById("schedule-range").addEventListener("click", event => {
    const button = event.target.closest("[data-days]");
    if (!button) return;
    scheduleDays = Number(button.dataset.days);
    document.querySelectorAll("#schedule-range [data-days]").forEach(item => item.classList.toggle("is-active", item === button));
    renderSchedule();
  });

  document.getElementById("history-search").addEventListener("input", renderHistory);
  document.getElementById("frequency-search").addEventListener("input", renderFrequencies);
  document.getElementById("language-select").addEventListener("change", changeApplicationLanguage);
  document.getElementById("app-version").addEventListener("click", () => checkForAppUpdate(true));
  document.getElementById("app-version").addEventListener("keydown", event => {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    checkForAppUpdate(true);
  });
  document.getElementById("close-update-dialog").addEventListener("click", closeUpdateDialog);
  document.getElementById("later-update").addEventListener("click", closeUpdateDialog);
  document.getElementById("install-update").addEventListener("click", installAppUpdate);
  document.getElementById("refresh-device-ports").addEventListener("click", event => runAction(event.currentTarget, () => refreshDevicePorts(true)));
  document.getElementById("device-port").addEventListener("change", renderFirmwareUpdater);
  document.getElementById("connect-device").addEventListener("click", connectDevice);
  document.getElementById("disconnect-device").addEventListener("click", disconnectDevice);
  document.getElementById("flash-firmware").addEventListener("click", flashSelectedFirmware);
  document.getElementById("start-manual-device").addEventListener("click", startManualDevice);
  document.getElementById("pause-device").addEventListener("click", pauseDevice);
  document.getElementById("stop-device").addEventListener("click", stopDevice);
  document.getElementById("device-session-list").addEventListener("click", startProfileOnDevice);
  document.getElementById("therapy-actions-paused").addEventListener("click", startProfileOnDevice);
  document.getElementById("add-profile").addEventListener("click", addProfile);
  document.getElementById("add-phase").addEventListener("click", addPhase);
  document.getElementById("save-profiles").addEventListener("click", saveProfiles);
  document.getElementById("profile-list").addEventListener("click", selectProfileFromList);
  document.getElementById("profile-list").addEventListener("keydown", event => {
    if (event.key !== "Enter" && event.key !== " ") return;
    if (!event.target.closest(".profile-row")) return;
    event.preventDefault();
    selectProfileFromList(event);
  });
  document.getElementById("phase-list").addEventListener("click", selectPhaseFromList);
  document.getElementById("phase-inspector").addEventListener("click", handleInspectorClick);
  document.getElementById("phase-inspector").addEventListener("input", handleInspectorInput);
  document.getElementById("phase-inspector").addEventListener("change", handleInspectorInput);
  document.getElementById("close-person-dialog").addEventListener("click", () => document.getElementById("person-dialog").close());
  document.getElementById("cancel-person-dialog").addEventListener("click", () => document.getElementById("person-dialog").close());
  document.getElementById("confirm-add-person").addEventListener("click", guarded("addPerson", addPerson));
  document.getElementById("new-person-name").addEventListener("keydown", event => {
    if (event.key === "Enter") {
      event.preventDefault();
      document.getElementById("confirm-add-person").click();
    }
  });
  document.getElementById("toggle-person-mode").addEventListener("click", guarded("togglePersonDialogMode", togglePersonDialogMode));
  ["new-person-single-name", "new-person-single-id"].forEach(id => {
    document.getElementById(id).addEventListener("keydown", guarded("addPersonEnter", event => {
      if (event.key !== "Enter") return;
      event.preventDefault();
      document.getElementById("confirm-add-person").click();
    }));
  });
  document.getElementById("close-person-edit-dialog").addEventListener("click", () => {
    resetPersonDeleteConfirmation(null);
    document.getElementById("person-edit-dialog").close();
  });
  document.getElementById("cancel-person-edit-dialog").addEventListener("click", () => {
    if (pendingPersonDeletion) {
      const person = (snapshot?.persons || []).find(entry => entry.id === editedPersonID);
      resetPersonDeleteConfirmation(person || null);
      return;
    }
    document.getElementById("person-edit-dialog").close();
  });
  document.getElementById("confirm-person-edit").addEventListener("click", guarded("savePersonName", savePersonName));
  document.getElementById("delete-person").addEventListener("click", guarded("deleteEditedPerson", deleteEditedPerson));
  document.getElementById("person-edit-name").addEventListener("keydown", event => {
    if (event.key === "Enter") {
      event.preventDefault();
      document.getElementById("confirm-person-edit").click();
    }
  });
  document.getElementById("open-ai-tools").addEventListener("click", openAITools);
  document.getElementById("close-ai-tools").addEventListener("click", () => document.getElementById("ai-dialog").close());
  document.getElementById("generate-ai-context").addEventListener("click", generateAIContext);
  document.getElementById("toggle-all-ai-persons").addEventListener("click", toggleAllAIPersons);
  document.getElementById("ai-person-pick").addEventListener("change", updateAISelectAllLabel);
  document.getElementById("copy-ai-context").addEventListener("click", copyAIContext);
  document.getElementById("download-ai-context").addEventListener("click", downloadAIContext);
  document.querySelectorAll(".ai-step-tab").forEach(tab => {
    tab.addEventListener("click", () => selectAIStep(tab.dataset.step));
    tab.addEventListener("keydown", handleAIStepKeydown);
  });
  document.getElementById("preview-ai-import").addEventListener("click", previewAIImport);
  document.getElementById("apply-ai-import").addEventListener("click", applyAIImport);
  document.getElementById("ai-import-input").addEventListener("input", () => {
    validatedAIJSON = "";
    document.getElementById("apply-ai-import").disabled = true;
    document.getElementById("ai-import-preview").textContent = "JSON zmieniono — sprawdź go ponownie.";
  });
}

function syncDraft() {
  draftConfig = JSON.parse(JSON.stringify(snapshot.config));
  draftConfig.profiles ||= [];
  profilesDirty = false;
  selectedProfile = Math.min(selectedProfile, Math.max(0, draftConfig.profiles.length - 1));
  const profile = draftConfig.profiles[selectedProfile];
  selectedPhase = Math.min(selectedPhase, Math.max(0, (profile?.phases?.length || 0) - 1));
  if (!selectedPersonID || !(snapshot.persons || []).some(person => person.id === selectedPersonID)) {
    selectedPersonID = profile?.person_id || profile?.id || snapshot.persons?.find(person => person.active)?.id || snapshot.persons?.[0]?.id || "";
  }
  updateDirtyState();
}

function renderAll() {
  renderToday();
  renderDeviceView();
  renderSchedule();
  renderProfilesEditor();
  renderHistory();
  renderArchive();
  renderFrequencies();
  renderPersons();
  // Nie polegamy wyłącznie na MutationObserverze. Po każdym pełnym renderze
  // natychmiast przepuszczamy także nowo utworzone węzły przez i18n, żeby
  // żaden ekran nie mignął ani nie został z polsko-angielskimi wyspami.
  window.ZapperI18n?.apply(document);
}

function renderPersons() {
  const persons = snapshot.persons || [];
  const active = persons.filter(person => person.active);
  renderAIPersons(active);
  if (!active.some(person => person.id === selectedPersonID) && active.length) {
    selectedPersonID = active[0].id;
    selectedProfile = (draftConfig.profiles || []).findIndex(profile => (profile.person_id || profile.id) === selectedPersonID);
    selectedPhase = 0;
    renderProfilesEditor();
  }
  const hasProfile = (draftConfig?.profiles || []).some(profile => (profile.person_id || profile.id) === selectedPersonID);
  document.getElementById("add-profile").disabled = false;
  document.getElementById("add-profile").title = hasProfile ? uiText("personHasProgram") : selectedPersonID ? uiText("createProgramForSelected") : uiText("addPerson");
}

function aiPersonStats(personID) {
  const profiles = snapshot?.config?.profiles || [];
  const hasProgram = profiles.some(profile => (profile.person_id || profile.id) === personID);
  let sessions = 0;
  Object.values(snapshot?.progress?.completions || {}).forEach(entry => {
    if (entry.person_id === personID || entry.profile_id === personID) sessions += 1;
  });
  return { hasProgram, sessions };
}

function renderAIPersons(active) {
  const pick = document.getElementById("ai-person-pick");
  if (!pick) return;
  const previously = new Set(Array.from(pick.querySelectorAll("input[type=checkbox]:checked")).map(input => input.value));
  const previousModes = new Map(Array.from(pick.querySelectorAll(".ai-person-mode-select")).map(select => [select.dataset.personId, select.value]));
  if (!previously.size && (selectedPersonID || active.length)) {
    previously.add(selectedPersonID || active[0].id);
  }
  pick.innerHTML = active.map(person => {
    const { hasProgram, sessions } = aiPersonStats(person.id);
    const checked = previously.has(person.id);
    const mode = previousModes.get(person.id) || (hasProgram ? "continuation" : "new");
    const effectiveMode = hasProgram ? mode : "new";
    const noProgramReason = uiText("noActiveProgram");
    return `
    <div class="ai-person-card${checked ? " is-selected" : ""}" data-person-id="${escapeAttribute(person.id)}">
      <label class="ai-person-option">
        <input type="checkbox" value="${escapeAttribute(person.id)}"${checked ? " checked" : ""}>
        <span>${escapeHTML(person.name)}</span>
        <code>${escapeHTML(person.id)}</code>
      </label>
      <div class="ai-person-meta">
        <span class="ai-person-badge${hasProgram ? " is-on" : ""}">${hasProgram ? uiText("activeProgram") : uiText("withoutProgram")}</span>
        <span class="ai-person-sessions">${escapeHTML(uiFormat("dynActiveProgramSessions", { count: sessions }))}</span>
      </div>
      <label class="ai-person-mode">
        <span>${uiText("taskType")}</span>
        <select class="ai-person-mode-select" data-person-id="${escapeAttribute(person.id)}"${checked ? "" : " disabled"}>
          <option value="new"${effectiveMode === "new" ? " selected" : ""}>${uiText("newProgram")}</option>
          <option value="continuation"${effectiveMode === "continuation" ? " selected" : ""}${hasProgram ? "" : ` disabled title="${escapeAttribute(noProgramReason)}"`}>${uiText("continueCurrent")}</option>
        </select>
      </label>
    </div>`;
  }).join("") || `<span class="ai-person-empty">${uiText("noPeople")}</span>`;
  updateAISelectAllLabel();
}

function toggleAllAIPersons() {
  const boxes = Array.from(document.querySelectorAll("#ai-person-pick input[type=checkbox]"));
  const allChecked = boxes.length > 0 && boxes.every(box => box.checked);
  boxes.forEach(box => { box.checked = !allChecked; });
  updateAISelectAllLabel();
}

function updateAISelectAllLabel() {
  const boxes = Array.from(document.querySelectorAll("#ai-person-pick input[type=checkbox]"));
  const allChecked = boxes.length > 0 && boxes.every(box => box.checked);
  const button = document.getElementById("toggle-all-ai-persons");
  if (button) button.textContent = allChecked ? uiText("deselectAll") : uiText("selectAll");
  boxes.forEach(box => {
    const card = box.closest(".ai-person-card");
    if (!card) return;
    card.classList.toggle("is-selected", box.checked);
    const select = card.querySelector(".ai-person-mode-select");
    if (select) select.disabled = !box.checked;
  });
}

function selectAIStep(step) {
  if (!step) return;
  document.querySelectorAll(".ai-step-tab").forEach(tab => {
    const isActive = tab.dataset.step === step;
    tab.classList.toggle("is-active", isActive);
    tab.setAttribute("aria-selected", isActive ? "true" : "false");
    tab.tabIndex = isActive ? 0 : -1;
    const panel = document.getElementById(tab.getAttribute("aria-controls"));
    if (panel) panel.hidden = !isActive;
  });
}

function handleAIStepKeydown(event) {
  if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
  event.preventDefault();
  const tabs = Array.from(document.querySelectorAll(".ai-step-tab"));
  const index = tabs.indexOf(event.currentTarget);
  const next = tabs[(index + (event.key === "ArrowRight" ? 1 : tabs.length - 1)) % tabs.length];
  selectAIStep(next.dataset.step);
  next.focus();
}

function selectPerson(personID) {
  selectedPersonID = personID;
  const profileIndex = (draftConfig.profiles || []).findIndex(profile => (profile.person_id || profile.id) === selectedPersonID);
  if (profileIndex >= 0) {
    selectedProfile = profileIndex;
    selectedPhase = 0;
  } else {
    selectedProfile = -1;
    selectedPhase = 0;
  }
  renderProfilesEditor();
  renderPersons();
}

// Dialog dodawania osob ma dwa tryby: "single" (dwa osobne pola) i "multi" (textarea).
let personDialogMode = "single";

// Komunikat MUSI byc widoczny WEWNATRZ dialogu — toast i #error-banner sa rysowane
// pod modalem i jego ::backdrop, wiec uzytkownik ich nie zobaczy.
function showPersonDialogError(message) {
  const element = document.getElementById("person-add-error");
  if (!element) return;
  element.textContent = message || "";
  element.hidden = !message;
}

function clearPersonDialogError() {
  showPersonDialogError("");
}

function setPersonDialogMode(mode) {
  personDialogMode = mode === "multi" ? "multi" : "single";
  const single = document.getElementById("person-single-mode");
  const multi = document.getElementById("person-multi-mode");
  const toggle = document.getElementById("toggle-person-mode");
  const confirm = document.getElementById("confirm-add-person");
  const heading = document.getElementById("person-add-heading");
  const isMulti = personDialogMode === "multi";
  single.hidden = isMulti;
  multi.hidden = !isMulti;
  toggle.textContent = isMulti ? uiText("backToSingle") : uiText("addMany");
  confirm.textContent = isMulti ? uiText("addPeople") : uiText("addPerson");
  if (heading) heading.textContent = isMulti ? uiText("newPeople") : uiText("newPerson");
  clearPersonDialogError();
}

function togglePersonDialogMode() {
  setPersonDialogMode(personDialogMode === "multi" ? "single" : "multi");
  const focused = personDialogMode === "multi"
    ? document.getElementById("new-person-name")
    : document.getElementById("new-person-single-name");
  focused.focus();
}

function openPersonDialog() {
  const nameInput = document.getElementById("new-person-single-name");
  const idInput = document.getElementById("new-person-single-id");
  const textarea = document.getElementById("new-person-name");
  nameInput.value = "";
  idInput.value = "";
  textarea.value = "";
  setPersonDialogMode("single");
  document.getElementById("person-dialog").showModal();
  nameInput.focus();
}

// Składnia wejścia: "Imię" albo "Imię = id". Bez "=" ID generuje Go (nazwa + losowy sufiks).
function parsePersonNames(raw) {
  const seen = new Set();
  const entries = [];
  String(raw || "").split(/[\n,]/).forEach(part => {
    const chunk = part.trim();
    if (!chunk) return;
    const separator = chunk.indexOf("=");
    const name = (separator >= 0 ? chunk.slice(0, separator) : chunk).trim();
    const id = separator >= 0 ? chunk.slice(separator + 1).trim().toLowerCase() : "";
    if (!name) return;
    const key = (id ? `#${id}` : name.toLocaleLowerCase("pl-PL"));
    if (seen.has(key)) return;
    seen.add(key);
    entries.push({ name, id });
  });
  return entries;
}

function personIDInputError(id) {
  if (!id) return "";
  if (/\s/.test(id)) return `ID „${id}” nie może zawierać spacji`;
  if (!/^[a-z0-9_-]+$/.test(id)) return `ID „${id}” może zawierać tylko litery a-z, cyfry, podkreślenie i myślnik`;
  if (id.length > 40) return `ID „${id}” jest za długie (maksymalnie 40 znaków)`;
  return "";
}

function personCountLabel(count) {
  if (count === 1) return "Dodano 1 osobę";
  const rest = count % 100;
  const last = count % 10;
  if (last >= 2 && last <= 4 && !(rest >= 12 && rest <= 14)) return `Dodano ${count} osoby`;
  return `Dodano ${count} osób`;
}

async function addPerson(event) {
  const button = event.currentTarget;
  clearPersonDialogError();
  if (personDialogMode === "multi") return addPersonEntries(button, readMultiPersonEntries());
  return addPersonEntries(button, readSinglePersonEntry());
}

function readSinglePersonEntry() {
  const name = document.getElementById("new-person-single-name").value.trim();
  const id = document.getElementById("new-person-single-id").value.trim().toLowerCase();
  if (!name) {
    showPersonDialogError("Podaj nazwę osoby");
    return null;
  }
  const idError = personIDInputError(id);
  if (idError) {
    showPersonDialogError(idError);
    return null;
  }
  return [{ name, id }];
}

function readMultiPersonEntries() {
  const names = parsePersonNames(document.getElementById("new-person-name").value);
  if (!names.length) {
    showPersonDialogError("Podaj co najmniej jedno imię");
    return null;
  }
  const badEntry = names.find(entry => personIDInputError(entry.id));
  if (badEntry) {
    showPersonDialogError(personIDInputError(badEntry.id));
    return null;
  }
  return names;
}

async function addPersonEntries(button, names) {
  if (!names) return;
  const dialog = document.getElementById("person-dialog");
  await runAction(button, async () => {
    let firstPersonID = "";
    let added = 0;
    let failure = null;
    for (const entry of names) {
      try {
        const result = await window.apiAddPerson(entry.name, entry.id);
        snapshot = result.snapshot;
        if (!firstPersonID) firstPersonID = result.person_id;
        added += 1;
      } catch (error) {
        failure = error;
        break;
      }
    }
    if (firstPersonID) {
      selectedPersonID = firstPersonID;
      selectedPhase = 0;
    }
    syncDraft();
    selectedProfile = (draftConfig?.profiles || []).findIndex(profile => (profile.person_id || profile.id) === selectedPersonID);
    renderAll();
    if (failure) {
      showPersonDialogError(`Nie udało się dodać wszystkich osób: ${normalizeError(failure)}`);
      showError(normalizeError(failure));
      toast(`Nie udało się dodać wszystkich osób: ${normalizeError(failure)}`, true);
      return;
    }
    document.getElementById("new-person-name").value = "";
    document.getElementById("new-person-single-name").value = "";
    document.getElementById("new-person-single-id").value = "";
    clearPersonDialogError();
    dialog.close();
    toast(personCountLabel(added));
  });
}

function openPersonEditDialog(personID) {
  const person = (snapshot.persons || []).find(entry => entry.id === personID);
  if (!person) return;
  editedPersonID = person.id;
  const input = document.getElementById("person-edit-name");
  input.value = person.name || "";
  document.getElementById("person-edit-id").textContent = `Stałe ID ${person.id} pozostaje bez zmian.`;
  armedDelete = "";
  clearTimeout(armedDeleteTimer);
  resetPersonDeleteConfirmation(person);
  document.getElementById("person-edit-dialog").showModal();
  input.focus();
  input.select();
}

async function savePersonName(event) {
  const person = (snapshot.persons || []).find(entry => entry.id === editedPersonID);
  if (!person) return;
  const name = document.getElementById("person-edit-name").value.trim();
  if (!name) {
    toast("Podaj nazwę osoby", true);
    return;
  }
  const updated = { ...person, name };
  await runAction(event.currentTarget, async () => {
    snapshot = await window.apiUpdatePerson(updated);
    syncDraft();
    renderAll();
    document.getElementById("person-edit-dialog").close();
    toast("Nazwa osoby została zapisana");
  });
}

// Program osoby liczymy WYŁĄCZNIE z zapisanego stanu (snapshot.config), bo tylko on decyduje
// o wyniku apiUpdatePerson po stronie Go. Wcześniej brano też draftConfig, przez co lista mogła
// pokazywać „bez programu”, a przycisk usuwania i tak był zablokowany.
function personProgramProfile(personID) {
  return (snapshot?.config?.profiles || []).find(profile => (profile.person_id || profile.id) === personID) || null;
}

function resetPersonDeleteConfirmation(person) {
  pendingPersonDeletion = "";
  const deleteButton = document.getElementById("delete-person");
  const cancelButton = document.getElementById("cancel-person-edit-dialog");
  const hint = document.getElementById("person-delete-hint");
  const program = person ? personProgramProfile(person.id) : null;
  deleteButton.disabled = Boolean(program);
  deleteButton.textContent = "Usuń osobę";
  deleteButton.classList.remove("is-confirming");
  cancelButton.textContent = "Anuluj";
  hint.classList.remove("is-confirming");
  hint.textContent = program
    ? `Najpierw zakończ lub usuń program „${program.name || program.id}”, dopiero potem można usunąć tę osobę.`
    : "Usunięcie trwale usuwa osobę, jej zakończone programy i całą historię z archiwum. Tej operacji nie można cofnąć.";
  deleteButton.title = program
    ? "Najpierw usuń lub zakończ program tej osoby — dopiero potem można ją usunąć."
    : "Trwale usuwa osobę wraz z jej archiwum i historią.";
}

async function deleteEditedPerson(event) {
  const person = (snapshot.persons || []).find(entry => entry.id === editedPersonID);
  if (!person) return;
  const button = event.currentTarget;
  // Potwierdzenie NIE wygasa samo — inaczej wolniejsze kliknięcie nigdy nie usuwa osoby.
  if (pendingPersonDeletion !== person.id) {
    pendingPersonDeletion = person.id;
    button.textContent = `Potwierdź: usuń ${person.name}`;
    button.classList.add("is-confirming");
    button.title = "Kliknij ponownie, aby trwale usunąć osobę. „Nie usuwaj” anuluje.";
    document.getElementById("cancel-person-edit-dialog").textContent = "Nie usuwaj";
    const hint = document.getElementById("person-delete-hint");
    hint.classList.add("is-confirming");
    hint.textContent = `Kliknij „Potwierdź: usuń ${person.name}” jeszcze raz, aby trwale usunąć tę osobę razem z historią. Tej operacji nie można cofnąć.`;
    return;
  }
  const hintElement = document.getElementById("person-delete-hint");
  await runAction(button, async () => {
    try {
      snapshot = await window.apiDeletePerson(person.id);
    } catch (error) {
      // Komunikat MUSI trafic do elementu wewnatrz dialogu — toast i #error-banner
      // sa niewidoczne pod modalem.
      hintElement.textContent = `Nie udalo sie usunac osoby: ${normalizeError(error)}`;
      hintElement.classList.add("is-confirming");
      throw error;
    }
    if (selectedPersonID === person.id) selectedPersonID = "";
    selectedProfile = 0;
    selectedPhase = 0;
    syncDraft();
    renderAll();
    resetPersonDeleteConfirmation(null);
    document.getElementById("person-edit-dialog").close();
    toast(`Osoba ${person.name} została usunięta — razem z historią i archiwum`);
  });
  if (pendingPersonDeletion) resetPersonDeleteConfirmation(person);
}

function openAITools() {
  renderPersons();
  validatedAIJSON = "";
  document.getElementById("apply-ai-import").disabled = true;
  document.getElementById("ai-copy-hint").hidden = true;
  selectAIStep("context");
  document.getElementById("ai-dialog").showModal();
}

function selectedAIPersonRequests() {
  return Array.from(document.querySelectorAll("#ai-person-pick input[type=checkbox]:checked")).map(input => {
    const card = input.closest(".ai-person-card");
    const select = card?.querySelector(".ai-person-mode-select");
    const person = (snapshot.persons || []).find(entry => entry.id === input.value);
    return { id: input.value, name: person?.name || input.value, mode: select?.value === "continuation" ? "continuation" : "new" };
  });
}

async function generateAIContext(event) {
  const targets = selectedAIPersonRequests();
  if (!targets.length) {
    toast(uiText("selectOnePerson"), true);
    return;
  }
  await runAction(event.currentTarget, async () => {
    const context = String(await window.apiGenerateAIContext({ person_ids: targets.map(target => target.id), mode: targets[0].mode })).trim();
    document.getElementById("ai-context-output").value = context;
    lastAIContextPersons = targets;
    document.getElementById("ai-copy-hint").hidden = true;
    toast(uiFormat("dynAiTextReady", { count: targets.length }));
  });
}

async function copyAIContext(event) {
  const output = document.getElementById("ai-context-output");
  if (!output.value.trim()) {
    toast("Najpierw wygeneruj tekst", true);
    return;
  }
  await runAction(event.currentTarget, async () => {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(output.value);
    } else {
      output.focus();
      output.select();
      document.execCommand("copy");
    }
    document.getElementById("ai-copy-hint").hidden = false;
    toast("Skopiowano tekst dla AI — przejdź do kroku 2");
  });
}

function slugify(value) {
  return String(value || "")
    .replace(/ł/g, "l")
    .replace(/Ł/g, "L")
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLocaleLowerCase("pl")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function aiContextFileName() {
  const today = new Date();
  const stamp = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;
  if (lastAIContextPersons.length === 1) {
    const slug = slugify(lastAIContextPersons[0].name) || slugify(lastAIContextPersons[0].id) || "osoba";
    return `kontekst-ai-${slug}-${stamp}.md`;
  }
  return `kontekst-ai-${stamp}.md`;
}

function downloadAIContext() {
  const output = document.getElementById("ai-context-output");
  if (!output.value.trim()) {
    toast("Najpierw wygeneruj tekst", true);
    return;
  }
  const blob = new Blob([output.value], { type: "text/markdown;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = aiContextFileName();
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
  toast(`Zapisano plik ${link.download}`);
}

async function previewAIImport(event) {
  const raw = document.getElementById("ai-import-input").value.trim();
  await runAction(event.currentTarget, async () => {
    const preview = await window.apiPreviewAIProfile(raw);
    validatedAIJSON = raw;
    document.getElementById("ai-import-preview").innerHTML = (preview.persons || []).map(person => `<div class="ai-import-preview-person">
      <strong>${escapeHTML(person.person_name)} · <code>${escapeHTML(person.person_id)}</code></strong>
      <span>${escapeHTML(uiFormat("dynProgramPhaseDays", { programs: person.program_count, phases: person.phase_count, days: person.total_days }))}</span>
      ${person.replaces_active ? `<em>${escapeHTML(uiText("activeProgramWillArchive"))}</em>` : ""}
      <ul>${(person.summary || []).map(item => `<li>${escapeHTML(item)}</li>`).join("")}</ul>
      ${renderImportWarnings(person.warnings)}
      ${renderUnknownFieldsWarning(person.unknown_fields)}
    </div>`).join("");
    document.getElementById("apply-ai-import").disabled = false;
    toast(uiText("jsonValid"));
  });
}

function renderImportWarnings(warnings) {
  const items = warnings || [];
  if (!items.length) return "";
  return `<div class="ai-import-warning"><strong>${escapeHTML(uiText("checkSpacing"))}</strong><ul>${items.map(item => `<li>${escapeHTML(localizedSource(item))}</li>`).join("")}</ul></div>`;
}

// Ostrzeżenie o zignorowanych polach musi być renderowane wewnątrz podglądu —
// #toast i #error-banner chowają się pod modalnym dialogiem.
function renderUnknownFieldsWarning(fields) {
  const paths = fields || [];
  if (!paths.length) return "";
  return `<div class="ai-import-warning">
      <strong>${escapeHTML(uiFormat("dynIgnoredFields", { count: paths.length }))}</strong>
      <span>${escapeHTML(uiText("ignoredFieldsHelp"))}</span>
      <ul>${paths.map(path => `<li><code>${escapeHTML(path)}</code></li>`).join("")}</ul>
    </div>`;
}

async function applyAIImport(event) {
  if (!validatedAIJSON) return;
  const button = event.currentTarget;
  if (!armDestructive(button, "apply-ai-profile", uiText("startProgramConfirm"))) return;
  await runAction(button, async () => {
    snapshot = await window.apiApplyAIProfile(validatedAIJSON);
    const parsedPayload = JSON.parse(validatedAIJSON.replace(/^```(?:json)?\s*/i, "").replace(/\s*```$/, ""));
    const importedPersonID = Array.isArray(parsedPayload) ? (parsedPayload[0] || {}).person_id : parsedPayload.person_id;
    selectedPersonID = importedPersonID || selectedPersonID;
    validatedAIJSON = "";
    document.getElementById("ai-import-input").value = "";
    document.getElementById("ai-import-preview").textContent = "Profil został zaimportowany.";
    document.getElementById("apply-ai-import").disabled = true;
    document.getElementById("ai-dialog").close();
    syncDraft();
    renderAll();
    toast("Nowy program został rozpoczęty");
  });
}

async function refreshDevicePorts(showConfirmation = false) {
  devicePorts = (await window.apiDevicePorts()) || [];
  const select = document.getElementById("device-port");
  const selected = select.value || deviceStatus.port || deviceStatus.preferred_port || "";
  select.innerHTML = `<option value="">Wybierz port COM</option>${devicePorts.map(port => `<option value="${escapeAttribute(port)}">${escapeHTML(port)}</option>`).join("")}`;
  if (devicePorts.includes(selected)) select.value = selected;
  renderFirmwareUpdater();
  if (showConfirmation) toast(devicePorts.length ? uiFormat("dynFoundPorts", { count: devicePorts.length }) : uiText("noPorts"));
}

async function refreshFirmwareFlashInfo() {
  try {
    firmwareFlashInfo = await window.apiFirmwareFlashInfo(preferredLanguage);
  } catch (error) {
    firmwareFlashInfo = null;
    window.zapperReportError("refreshFirmwareFlashInfo", error);
  }
  renderFirmwareUpdater();
}

function displayLanguageName(code) {
  return window.ZapperI18n?.languages?.[code]?.name || String(code || "").toUpperCase();
}

function renderFirmwareUpdater() {
  const installed = document.getElementById("firmware-installed-version");
  if (!installed) return;
  const info = firmwareFlashInfo;
  installed.textContent = deviceStatus.firmware || "—";
  document.getElementById("firmware-latest-version").textContent = info?.latest_version || "5.1.0";
  document.getElementById("firmware-target-language").textContent = displayLanguageName(info?.language || preferredLanguage);
  const lcd = info?.lcd_language || firmwareLanguageFor(preferredLanguage).lcdLanguage;
  const fallback = Boolean(info?.english_fallback);
  document.getElementById("firmware-lcd-language").textContent = `${displayLanguageName(lcd)}${fallback ? ` · ${window.ZapperI18n?.t("englishFallback") || "English fallback"}` : ""}`;

  const toolState = document.getElementById("firmware-tool-state");
  toolState.classList.remove("is-ready", "is-error");
  if (!info) {
    toolState.textContent = window.ZapperI18n?.t("checkingTool") || "Sprawdzanie narzędzia…";
  } else if (!info.sketch_available) {
    toolState.classList.add("is-error");
    toolState.textContent = window.ZapperI18n?.t("sketchMissing") || "Brak wariantu firmware dla wybranego języka";
  } else if (!info.tool_available) {
    toolState.classList.add("is-error");
    toolState.textContent = uiText("toolMissing");
  } else {
    toolState.classList.add("is-ready");
    toolState.textContent = window.ZapperI18n?.t("toolReady") || "Narzędzie do wgrywania gotowe";
  }

  const busyState = ["running", "armed", "starting", "stopping", "reconnecting"].includes(deviceStatus.state || "");
  const port = document.getElementById("device-port")?.value || deviceStatus.port || "";
  const button = document.getElementById("flash-firmware");
  button.disabled = firmwareFlashBusy || busyState || !port || !info?.tool_available || !info?.sketch_available;

  const note = document.getElementById("firmware-flash-note");
  if (firmwareFlashBusy) note.textContent = window.ZapperI18n?.t("flashInProgress") || "Wgrywanie firmware…";
  else if (busyState) note.textContent = "Zatrzymaj bieżącą sesję przed wgrywaniem firmware.";
  else if (!port) note.textContent = window.ZapperI18n?.t("flashNote") || "Wybierz port COM. Program nie wgrywa firmware automatycznie przy starcie.";
  else if (!info?.tool_available) note.textContent = window.ZapperI18n?.t("toolMissing") || "Brak arduino-cli — wgrywanie z aplikacji jest niedostępne";
  else note.textContent = `COM: ${port} · ${info.variant_name || "firmware"} · ${info.latest_version || "5.1.0"}`;
}

async function flashSelectedFirmware(event) {
  const button = event.currentTarget;
  const port = document.getElementById("device-port").value || deviceStatus.port || "";
  if (!port || firmwareFlashBusy) return;
  firmwareFlashBusy = true;
  renderFirmwareUpdater();
  const output = document.getElementById("firmware-flash-output");
  output.hidden = true;
  output.textContent = "";
  try {
    if (deviceStatus.connected) {
      deviceStatus = await window.apiDeviceDisconnect();
      renderDeviceStatus();
    }
    const result = await window.apiFlashFirmware({
      port,
      language: preferredLanguage,
      old_bootloader: document.getElementById("firmware-bootloader").value === "old"
    });
    output.textContent = result.output || "OK";
    output.hidden = !output.textContent;
    toast(window.ZapperI18n?.t("flashDone") || "Firmware zostało wgrane");
    await new Promise(resolve => setTimeout(resolve, 1200));
    await refreshDevicePorts(false);
  } catch (error) {
    const message = normalizeError(error);
    output.textContent = message;
    output.hidden = false;
    toast(message, true);
  } finally {
    firmwareFlashBusy = false;
    await refreshFirmwareFlashInfo();
    renderFirmwareUpdater();
    button.blur();
  }
}

async function connectDevice(event) {
  const port = document.getElementById("device-port").value;
  await runAction(event.currentTarget, async () => {
    deviceStatus = await window.apiDeviceConnect(port);
    renderDeviceStatus();
    toast(`Otwieranie ${port} — czekam na płytkę`);
  });
}

async function disconnectDevice(event) {
  await runAction(event.currentTarget, async () => {
    deviceStatus = await window.apiDeviceDisconnect();
    renderDeviceStatus();
    toast("Płytka została rozłączona");
  });
}

async function startManualDevice(event) {
  const value = Number(document.getElementById("manual-frequency").value);
  const unitHz = Number(document.getElementById("manual-frequency-unit").value);
  const minutes = Number(document.getElementById("manual-duration").value);
  const frequencyMilliHz = Math.round(value * unitHz * 1000);
  const durationSeconds = Math.round(minutes * 60);
  await runAction(event.currentTarget, async () => {
    activeSessionKind = "manual";
    deviceStatus = await window.apiDeviceStartManual(frequencyMilliHz, durationSeconds);
    renderDeviceStatus();
    renderTherapy();
    syncTherapyView();
    toast("Sesja przygotowana — potwierdź ją na płytce");
  });
}

function hasSnapshot(value) {
  return Boolean(value && Object.keys(value).length > 0 && Array.isArray(value.today));
}

async function stopDevice(event) {
  await runAction(event.currentTarget, async () => {
    const result = await window.apiDeviceStop();
    deviceStatus = result.status;
    if (hasSnapshot(result.snapshot)) {
      snapshot = result.snapshot;
    } else {
      snapshot = await window.apiLoad();
    }
    syncDraft();
    renderAll();
    syncTherapyView();
    toast(result.pause && result.pause.remaining_seconds !== undefined
      ? `Zatrzymano i zapisano postęp — zostało ${formatDuration(result.pause.remaining_seconds)}. Wznowienie ruszy od tego miejsca`
      : "Wysłano polecenie zatrzymania");
  });
}

async function pauseDevice(event) {
  await runAction(event.currentTarget, async () => {
    const result = await window.apiDevicePause();
    deviceStatus = result.status;
    if (hasSnapshot(result.snapshot)) {
      snapshot = result.snapshot;
    } else {
      snapshot = await window.apiLoad();
    }
    syncDraft();
    renderAll();
    syncTherapyView();
    toast(result.pause && result.pause.remaining_seconds !== undefined
      ? `Sesja wstrzymana — pozostało ${formatDuration(result.pause.remaining_seconds)}`
      : "Sesja ręczna zatrzymana — tryb ręczny nie zapisuje postępu");
  });
}

async function startProfileOnDevice(event) {
  const dismissButton = event.target.closest("[data-dismiss-session-group]");
  if (dismissButton) {
    if (!armDestructive(dismissButton, `dismiss-group-${dismissButton.dataset.dismissSessionGroup}`, "Kliknij ponownie: odpuść ten termin")) return;
    await runAction(dismissButton, async () => {
      snapshot = await window.apiDismissSessionGroup(dismissButton.dataset.dismissSessionGroup);
      syncDraft();
      renderAll();
      toast("Termin odpuszczony — nie został zapisany jako wykonany");
    });
    return;
  }
  const cancelButton = event.target.closest("[data-cancel-pause]");
  if (cancelButton) {
    await runAction(cancelButton, async () => {
      if (cancelButton.dataset.manualPause === "true") {
        activeSessionKind = "manual";
        deviceStatus = await window.apiDeviceResumeManual(true);
        renderDeviceStatus();
        renderTherapy();
        syncTherapyView();
        toast("Sesja ręczna zaczyna się od początku — potwierdź na płytce");
        return;
      }
      snapshot = await window.apiCancelPausedSession(cancelButton.dataset.cancelPause);
      syncDraft();
      renderAll();
      syncTherapyView();
      toast("Usunięto częściowy postęp — sesja znów zacznie się od początku");
    });
    return;
  }
  const button = event.target.closest("[data-device-session]");
  if (!button) return;
  if (button.dataset.outOfOrder === "true" && !armDestructive(button, `out-of-order-start-${button.dataset.deviceSession}`, "Kliknij ponownie: wykonaj poza kolejnością")) return;
  await runAction(button, async () => {
    if (button.dataset.manualPause === "true") {
      activeSessionKind = "manual";
      deviceStatus = await window.apiDeviceResumeManual(false);
      renderDeviceStatus();
      renderTherapy();
      syncTherapyView();
      toast("Sesja ręczna wznowiona — potwierdź na płytce");
      return;
    }
    activeSessionKind = "profile";
    deviceStatus = await window.apiDeviceStartSession(button.dataset.deviceSession);
    renderDeviceStatus();
    renderTherapy();
    syncTherapyView();
    toast(`${button.dataset.paused === "true" ? "Wznowienie" : "Sesja"} ${button.dataset.profileName} gotowe — potwierdź na płytce`);
  });
}

async function pollDeviceStatus() {
  try {
    const previousState = deviceStatus.state;
    deviceStatus = await window.apiDeviceStatus();
    renderDeviceStatus();
    renderTherapy();
    // Ekran terapii otwiera się tylko przy ZMIANIE stanu — sesja w trakcie nie
    // wyrywa użytkownika z innego widoku przy każdym odpytywaniu.
    if (deviceStatus.state !== previousState) syncTherapyView();
    if (deviceStatus.state === "done" && previousState !== "done") {
      snapshot = await window.apiLoad();
      renderToday();
      renderHistory();
      renderSchedule();
      renderDeviceSessions();
      toast(activeSessionKind === "manual"
        ? "Płytka zakończyła sesję ręczną"
        : "Płytka zakończyła sesję — historia została uzupełniona");
    } else if (previousState === "running" && deviceStatus.state === "idle") {
      snapshot = await window.apiLoad();
      syncDraft();
      renderAll();
      toast("Sesja została zatrzymana na płytce");
    }
  } catch (error) {
    clearInterval(devicePollTimer);
    window.zapperReportError("pollDeviceStatus", error, {
      deviceStatus: deviceStatus,
      currentView: currentView,
      snapshot: snapshot ? {
        today: (snapshot.today || []).length,
        schedule: (snapshot.schedule || []).length,
        profiles: ((snapshot.config || {}).profiles || []).length,
        persons: (snapshot.persons || []).length
      } : null
    });
    devicePollTimer = setInterval(pollDeviceStatus, 750);
  }
}

function renderDeviceView() {
  renderDeviceSessions();
  renderDeviceStatus();
}

function sessionCompletionExists(sessionID) {
  return Boolean(sessionID && snapshot.progress?.completions?.[sessionID]);
}

// Scheduler rozbija serię 3x na trzy techniczne sesje, bo każda część ma własny
// odstęp i własny zapis wykonania. UI składa je z powrotem w jedną pozycję 0/3.
function groupSessionPlans(plans) {
  const groups = new Map();
  (plans || []).filter(plan => plan.status === "session").forEach(plan => {
    const key = plan.session_group_id || plan.session_id;
    if (!groups.has(key)) groups.set(key, { key, items: [] });
    groups.get(key).items.push(plan);
  });
  return [...groups.values()].map(group => {
    group.items.sort((first, second) => Number(first.repeat_index || 1) - Number(second.repeat_index || 1));
    group.total = Math.max(1, ...group.items.map(plan => Number(plan.repeat_count || plan.scheduling?.repetitions || 1)));
    const sample = group.items[0];
    group.doneCount = 0;
    for (let repeat = 1; repeat <= group.total; repeat++) {
      const sessionID = group.total > 1 ? `${group.key}:${repeat}` : sample.session_id;
      if (sessionCompletionExists(sessionID)) group.doneCount++;
    }
    group.actionPlan = group.items.find(plan => !sessionCompletionExists(plan.session_id)) || group.items[group.items.length - 1];
    group.done = group.doneCount >= group.total;
    group.overdue = group.items.some(plan => plan.overdue && !sessionCompletionExists(plan.session_id));
    return group;
  });
}

// Kilka zaległych terminów tego samego programu nie zajmuje kilku wierszy.
// Nadal pozostają osobnymi sesjami w danych; karta pokazuje najstarszą z kolejki.
function buildSessionQueues(plans) {
  const queues = new Map();
  groupSessionPlans(plans).forEach(group => {
    const plan = group.actionPlan;
    const key = [plan.profile_id, plan.run_id, plan.phase_id, plan.scheduling?.program_id || plan.program].join("|");
    if (!queues.has(key)) queues.set(key, { key, groups: [] });
    queues.get(key).groups.push(group);
  });
  return [...queues.values()].map(queue => {
    queue.groups.sort((first, second) => String(first.actionPlan.planned_date || "").localeCompare(String(second.actionPlan.planned_date || "")));
    queue.pendingGroups = queue.groups.filter(group => !group.done);
    queue.activeGroup = queue.pendingGroups[0] || queue.groups[queue.groups.length - 1];
    queue.actionPlan = queue.activeGroup.actionPlan;
    queue.done = queue.pendingGroups.length === 0;
    queue.overdue = queue.pendingGroups.some(group => group.overdue);
    return queue;
  });
}

function seriesLabel(group) {
  return group.total > 1
    ? uiFormat("dynParts", { done: group.doneCount, total: group.total })
    : uiText("sessionSingular");
}

function queueLabel(queue) {
  const count = queue.pendingGroups.length;
  return count > 1 ? uiFormat("dynAppointmentsQueue", { count }) : "";
}

function renderDeviceSessions() {
  const container = document.getElementById("device-session-list");
  const sessions = (snapshot.today || []).filter(plan => plan.status === "session");
  // Każda data jest osobnym terminem. Grupujemy wyłącznie techniczne części
  // jednego terminu (repeat_count), nigdy kilku zaległych dat tego samego programu.
  const groups = groupSessionPlans(sessions).filter(group => !group.done);
  if (!groups.length) {
    container.innerHTML = `<div class="empty-state"><strong>${escapeHTML(uiText("noTodaySessions"))}</strong><span>${escapeHTML(uiText("todayScheduleEmpty"))}</span></div>`;
    return;
  }
  container.innerHTML = groups.map(group => {
    const plan = group.actionPlan;
    const steps = plan.device_steps || [];
    const duration = steps.reduce((sum, step) => sum + Number(step.duration_seconds || 0), 0);
    const seriesProgress = group.total > 1 ? `${group.doneCount}/${group.total}` : "";
    const progress = group.total > 1 ? `${uiText("series")} ${seriesLabel(group)}` : "";
    const older = Number(plan.older_pending_count || 0);
    const orderWarning = localizedOlderWarning(older);
    const secondary = [progress, orderWarning].filter(Boolean).join(" · ");
    const startLabel = plan.paused ? uiText("resume") : older ? uiText("outOfOrder") : uiText("sendToBoard");
    return `<article class="device-session-row">
      <div><strong>${escapeHTML(plan.profile_name)}</strong><span>${escapeHTML(plan.overdue ? uiFormat("dynOverdueFrom", { date: formatShortDate(plan.planned_date) }) : (plan.phase_name || uiText("noPhase")))}</span></div>
      <div><strong>${escapeHTML(plan.program)}</strong><span>${escapeHTML(secondary || plan.note || "")}</span>${secondary && plan.note ? `<span>${escapeHTML(plan.note)}</span>` : ""}</div>
      <div class="device-step-summary">${plan.paused
        ? escapeHTML(uiFormat("dynPausedDuration", { remaining: formatDuration(plan.remaining_seconds || duration) }))
        : plan.available
          ? (steps.length ? `${seriesProgress ? `${seriesProgress} · ` : ""}${escapeHTML(uiFormat("dynStepsDuration", { count: steps.length, duration: formatDuration(duration) }))}` : escapeHTML(uiText("noSteps")))
          : scheduleWaitingMarkup(plan)}</div>
      <div class="device-session-actions"><button class="button primary" data-device-session="${escapeAttribute(plan.session_id)}" data-profile-name="${escapeAttribute(plan.profile_name)}" data-paused="${Boolean(plan.paused)}" data-out-of-order="${older > 0}" data-has-steps="${steps.length > 0}" data-available="${Boolean(plan.available)}" ${steps.length && plan.available ? "" : "disabled"}>${escapeHTML(startLabel)}</button>${plan.paused ? `<button class="button secondary" data-cancel-pause="${escapeAttribute(plan.session_id)}">${escapeHTML(uiText("startOver"))}</button>` : ""}<button class="button text-danger" data-dismiss-session-group="${escapeAttribute(plan.session_id)}">${escapeHTML(uiText("dismissTerm"))}</button></div>
    </article>`;
  }).join("");
}

function renderDeviceStatus() {
  const state = deviceStatus.state || "disconnected";
  const running = ["running", "starting", "stopping"].includes(state);
  const waiting = state === "armed";
  const recovering = state === "reconnecting" || (state === "connecting" && Boolean(deviceStatus.active_session_id));
  const busy = running || waiting || recovering;
  const ready = Boolean(deviceStatus.ready);
  const badge = document.getElementById("device-state-badge");
  badge.classList.toggle("is-ready", ready && !busy);
  badge.classList.toggle("is-running", running || waiting);
  badge.querySelector("strong").textContent = waiting ? uiText("confirmOnBoard") : running ? uiText("sessionRunning") : recovering ? uiText("recoveringConnection") : ready ? uiText("ready") : deviceStatus.connected ? uiText("connecting") : uiText("disconnected");

  const header = document.getElementById("device-header-status");
  header.classList.toggle("is-offline", !ready && !busy);
  header.classList.toggle("is-running", running || waiting);
  header.textContent = waiting ? uiText("confirmOnBoard") : running ? uiText("boardWorking") : recovering ? uiFormat("dynRecovering", { target: deviceStatus.port || uiText("connectionWord") }) : ready ? uiFormat("dynBoardPort", { port: deviceStatus.port }) : uiText("boardDisconnected");
  const dot = document.getElementById("device-nav-dot");
  dot.classList.toggle("is-ready", ready && !busy);
  dot.classList.toggle("is-running", running || waiting);

  document.getElementById("device-state").textContent = deviceStateLabel(state);
  document.getElementById("device-firmware").textContent = deviceStatus.firmware || "—";
  document.getElementById("device-frequency").textContent = deviceStatus.frequency_millihz ? formatFrequency(deviceStatus.frequency_millihz) : "—";
  document.getElementById("device-remaining").textContent = deviceStatus.remaining_ms ? formatClock(Math.ceil(deviceStatus.remaining_ms / 1000)) : "—";
  document.getElementById("device-step").textContent = deviceStatus.step_count
    ? waiting ? `—/${deviceStatus.step_count}` : `${deviceStatus.step_index}/${deviceStatus.step_count}`
    : "—";
  const activePanel = document.getElementById("active-session-panel");
  if (activePanel) activePanel.remove();
  document.getElementById("device-connection-message").textContent = deviceStatus.last_error || deviceStatus.message || uiText("selectBoardPort");
  document.getElementById("device-last-message").textContent = deviceStatus.message || uiText("noCommunication");
  document.getElementById("connect-device").disabled = Boolean(deviceStatus.connected) || recovering;
  document.getElementById("disconnect-device").disabled = !deviceStatus.connected && !recovering;
  document.getElementById("start-manual-device").disabled = !ready || busy;
  document.getElementById("pause-device").disabled = !ready || !["running", "armed", "starting"].includes(state);
  document.getElementById("stop-device").disabled = !ready || (!busy && state !== "armed");
  document.querySelectorAll("[data-device-session]").forEach(button => button.disabled = button.dataset.hasSteps !== "true" || button.dataset.available !== "true" || !ready || busy);
  const portSelect = document.getElementById("device-port");
  portSelect.disabled = Boolean(deviceStatus.connected) || recovering;
  if (!portSelect.value && deviceStatus.preferred_port && devicePorts.includes(deviceStatus.preferred_port)) portSelect.value = deviceStatus.preferred_port;
  renderFirmwareUpdater();
}

function renderTherapy() {
  const state = deviceStatus.state || "disconnected";
  const running = ["running", "starting", "stopping"].includes(state);
  const waiting = state === "armed";
  const recovering = state === "reconnecting" || (state === "connecting" && Boolean(deviceStatus.active_session_id));
  const busy = running || waiting || recovering;
  const manualPaused = Boolean(deviceStatus.manual_paused);
  const pausedPlan = !busy && !manualPaused && (snapshot.today || []).find(plan => plan.status === "session" && plan.paused && !plan.paused_recorded);
  document.getElementById("therapy-actions-busy").hidden = !busy;
  document.getElementById("therapy-actions-paused").hidden = !(pausedPlan || manualPaused);
  if (pausedPlan) {
    document.getElementById("therapy-eyebrow").textContent = uiText("sessionPausedUpper");
    document.getElementById("therapy-heading").textContent = pausedPlan.profile_name;
    document.getElementById("therapy-frequency").textContent = pausedPlan.program || "—";
    document.getElementById("therapy-time").textContent = formatClock(pausedPlan.remaining_seconds || 0);
    document.getElementById("therapy-status").textContent = uiFormat("dynRemainingAnytime", { remaining: formatDuration(pausedPlan.remaining_seconds || 0) });
    document.getElementById("therapy-step").textContent = "";
    const resumeButton = document.querySelector("#therapy-actions-paused [data-device-session]");
    if (resumeButton) {
      resumeButton.dataset.deviceSession = pausedPlan.session_id;
      resumeButton.dataset.profileName = pausedPlan.profile_name;
      delete resumeButton.dataset.manualPause;
    }
    const cancelPause = document.querySelector("#therapy-actions-paused [data-cancel-pause]");
    if (cancelPause) {
      cancelPause.dataset.cancelPause = pausedPlan.session_id;
      delete cancelPause.dataset.manualPause;
    }
    return;
  }
  if (manualPaused) {
    const remaining = deviceStatus.manual_pause_seconds || 0;
    document.getElementById("therapy-eyebrow").textContent = uiText("sessionPausedUpper");
    document.getElementById("therapy-heading").textContent = uiText("manualSession");
    document.getElementById("therapy-frequency").textContent = deviceStatus.frequency_millihz ? formatFrequency(deviceStatus.frequency_millihz) : "—";
    document.getElementById("therapy-time").textContent = formatClock(remaining);
    document.getElementById("therapy-status").textContent = uiFormat("dynRemainingContinueHere", { remaining: formatDuration(remaining) });
    document.getElementById("therapy-step").textContent = "";
    const resumeButton = document.querySelector("#therapy-actions-paused [data-device-session]");
    if (resumeButton) {
      resumeButton.dataset.deviceSession = "manual";
      resumeButton.dataset.profileName = uiText("manualSession");
      resumeButton.dataset.manualPause = "true";
    }
    const cancelPause = document.querySelector("#therapy-actions-paused [data-cancel-pause]");
    if (cancelPause) {
      cancelPause.dataset.cancelPause = "manual";
      cancelPause.dataset.manualPause = "true";
    }
    return;
  }
  document.getElementById("therapy-eyebrow").textContent = recovering ? uiText("recoveryUpper") : busy ? uiText("runningUpper") : uiText("therapy");
  document.getElementById("therapy-heading").textContent = deviceStatus.active_profile || (busy ? uiText("manualSession") : uiText("nothingRunning"));
  document.getElementById("therapy-frequency").textContent = deviceStatus.frequency_millihz ? formatFrequency(deviceStatus.frequency_millihz) : "—";
  document.getElementById("therapy-time").textContent = deviceStatus.remaining_ms ? formatClock(Math.ceil(deviceStatus.remaining_ms / 1000)) : "00:00";
  document.getElementById("therapy-status").textContent = waiting ? uiText("confirmUnlimited") : running ? uiText("sessionRunning") : recovering ? uiText("comRecovery") : uiText("sendFromDevice");
  document.getElementById("therapy-step").textContent = deviceStatus.step_count
    ? waiting ? uiFormat("dynStepProgress", { total: deviceStatus.step_count }) : uiFormat("dynStepIndex", { index: deviceStatus.step_index, total: deviceStatus.step_count })
    : "";
}

function syncTherapyView() {
  const state = deviceStatus.state || "disconnected";
  const recovering = state === "reconnecting" || (state === "connecting" && Boolean(deviceStatus.active_session_id));
  const busy = ["running", "starting", "stopping", "armed"].includes(state) || recovering;
  const manualPaused = Boolean(deviceStatus.manual_paused);
  const pausedPlan = !busy && !manualPaused && (snapshot.today || []).find(plan => plan.status === "session" && plan.paused && !plan.paused_recorded);
  if ((busy || pausedPlan || manualPaused) && currentView !== "therapy") openView("therapy");
  else if (!busy && !pausedPlan && !manualPaused && currentView === "therapy") openView("device");
}

function renderToday() {
  const plans = snapshot.today || [];
  const sessions = plans.filter(plan => plan.status === "session");
  const sessionGroups = groupSessionPlans(sessions);
  const completed = sessionGroups.filter(group => group.done).length;
  const overdue = sessionGroups.filter(group => group.overdue && !group.done).length;
  const dayNumber = plans[0]?.day_number ?? calculateDayNumber(snapshot.progress.start_date);
  document.getElementById("day-number").textContent = dayNumber > 0 ? dayNumber : "—";
  document.getElementById("start-date").value = snapshot.progress.start_date || snapshot.config.start_date || "";
  document.getElementById("today-summary").textContent = sessionGroups.length
    ? localizedTodaySummary(sessionGroups.length, completed, overdue, snapshot.today_remaining_seconds || 0)
    : uiText("noSessionsTodaySummary");
  const percent = sessionGroups.length ? (completed / sessionGroups.length) * 100 : 0;
  document.getElementById("daily-progress").style.width = `${percent}%`;

  renderOverdueActions();

  const list = document.getElementById("today-list");
  if (!plans.length) {
    list.innerHTML = `<div class="empty-state"><strong>${escapeHTML(uiText("noProfilesYet"))}</strong><span>${escapeHTML(uiText("addFirstProfilePhase"))}</span></div>`;
    return;
  }
  const sessionRows = sessionGroups.map(group => {
    const plan = group.actionPlan;
    const interactive = true;
    const done = group.done;
    const statusLabel = done ? uiText("completed") : plan.paused ? uiText("paused") : group.doneCount > 0
      ? uiFormat("dynGroupCompleted", { done: group.doneCount, total: group.total })
      : plan.overdue ? uiText("overdue") : !plan.available ? uiText("waiting") : uiText("toDo");
    const context = plan.overdue ? localizedPlannedContext(plan) : escapeHTML(plan.phase_name || uiText("noActivePhase"));
    const repeat = group.total > 1 ? `${uiText("series")} ${group.doneCount}/${group.total}` : (plan.repetitions || "1x");
    const older = Number(plan.older_pending_count || 0);
    const orderWarning = localizedOlderWarning(older);
    const note = plan.paused
      ? uiFormat("dynRemainingDevice", { remaining: formatDuration(plan.remaining_seconds || 0) })
      : !plan.available && plan.blocked_reason ? localizedBlockedReason(plan.blocked_reason)
      : ([orderWarning, plan.note || ""].filter(Boolean).join(" ") || uiText("noExtraNotes"));
    const noteMarkup = !plan.paused && !plan.available
      ? scheduleWaitingMarkup(plan)
      : escapeHTML(note);
    const nextPart = Math.min(group.total, group.doneCount + 1);
    const buttonLabel = done ? uiText("completedMark") : plan.available
      ? (group.total > 1 ? uiFormat("dynRunPart", { part: nextPart, total: group.total }) : uiText("markCompleted"))
      : (group.total > 1 ? uiFormat("dynPartWaiting", { part: nextPart, total: group.total }) : uiText("notYet"));
    return `<article class="session-row ${done ? "is-done" : ""} ${group.overdue && !done ? "is-overdue" : ""} ${!plan.available && interactive ? "is-waiting" : ""}">
      <div class="session-avatar">${escapeHTML(initials(plan.profile_name))}</div>
      <div class="session-person">
        <strong>${escapeHTML(plan.profile_name)}</strong>
        <span>${context}</span>
      </div>
      <div class="session-program">
        <strong>${escapeHTML(plan.program)}</strong>
        <span>${noteMarkup}</span>
      </div>
      <div class="session-meta">
        <strong>${escapeHTML(plan.time || "-")} · ${escapeHTML(repeat)}</strong>
        <span class="status-tag ${group.overdue && !done ? "overdue" : !plan.available && interactive ? "waiting" : done ? "complete" : "session"}">${escapeHTML(statusLabel)}</span>
      </div>
      <div class="session-row-actions"><button class="done-button ${done ? "is-done" : ""}" data-session-done="${escapeAttribute(plan.session_id || "")}" data-done="${done}" data-out-of-order="${older > 0}" ${interactive && (plan.available || done) ? "" : "disabled"}>${buttonLabel}</button>${done ? "" : `<button class="button text-danger compact" data-dismiss-session-group="${escapeAttribute(plan.session_id || "")}">${uiText("dismissTerm")}</button>`}</div>
    </article>`;
  });

  const nonSessionRows = plans.filter(plan => plan.status !== "session").map(plan => `<article class="session-row">
    <div class="session-avatar">${escapeHTML(initials(plan.profile_name))}</div>
    <div class="session-person"><strong>${escapeHTML(plan.profile_name)}</strong><span>${escapeHTML(plan.phase_name || uiText("noActivePhase"))}</span></div>
    <div class="session-program"><strong>${escapeHTML(plan.program)}</strong><span>${escapeHTML(plan.note || uiText("noExtraNotes"))}</span></div>
    <div class="session-meta"><strong>${escapeHTML(plan.time || "-")}</strong><span class="status-tag">${escapeHTML(statusText(plan.status, false))}</span></div>
    <button class="done-button" disabled>${uiText("noSessionButton")}</button>
  </article>`);
  list.innerHTML = [...sessionRows, ...nonSessionRows].join("");
}

// Zaległości potrafią urosnąć do dziesiątek pozycji czyszczonych po jednej dziennie,
// przez co profil wisi w nieskończoność. To jest jawne wyjście: odpuszczenie zaległych
// sesji. NIE oznacza ich jako wykonanych — zabiegi się nie odbyły i nie trafiają
// do historii ani do statystyk.
function renderOverdueActions() {
  const container = document.getElementById("overdue-actions");
  if (!container) return;
  const states = (snapshot.profile_states || []).filter(state => state.overdue_count > 0);
  if (!states.length) {
    container.innerHTML = "";
    return;
  }
  container.innerHTML = states.map(state => `<div class="overdue-banner">
    <div>
      <strong>${escapeHTML(uiFormat("dynProfileOverdueCount", { name: state.profile_name, count: state.overdue_count }))}</strong>
      <span>${uiText("overdueHelp")}</span>
    </div>
    <button class="button text-danger" data-dismiss-overdue="${escapeAttribute(state.profile_id || "")}">${uiText("dismissOverdue")}</button>
  </div>`).join("");
}

function renderSchedule() {
  const container = document.getElementById("schedule-content");
  if (!snapshot.schedule?.length) {
    container.innerHTML = `<div class="empty-state"><strong>${escapeHTML(uiText("noSchedule"))}</strong><span>${escapeHTML(uiText("addProfilesPlan"))}</span></div>`;
    return;
  }
  container.innerHTML = snapshot.schedule.map(profile => `<section class="schedule-profile">
    <h2>${escapeHTML(profile.profile_name)}</h2>
    <div class="table-wrap"><table>
      <thead><tr><th>Data</th><th>Dzień</th><th>Faza</th><th>Program</th><th>Czas</th><th>Status</th></tr></thead>
      <tbody>${profile.days.slice(0, scheduleDays).map(day => `<tr>
        <td><span class="table-primary">${formatShortDate(day.date)}</span></td>
        <td>${escapeHTML(day.day_name)}</td>
        <td>${escapeHTML(day.phase_name || "—")}</td>
        <td><span class="table-primary">${escapeHTML(day.program)}</span>${day.note ? `<span class="table-secondary">${escapeHTML(day.note)}</span>` : ""}</td>
        <td>${escapeHTML(day.time)}${day.repetitions !== "-" ? `<span class="table-secondary">${escapeHTML(day.repetitions)}</span>` : ""}</td>
        <td><span class="schedule-status ${escapeHTML(day.status)}${day.done ? " is-done" : ""}"></span>${escapeHTML(statusText(day.status, day.done))}</td>
      </tr>`).join("")}</tbody>
    </table></div>
  </section>`).join("");
}

function renderProfilesEditor() {
  const profiles = draftConfig.profiles || [];
  const profileList = document.getElementById("profile-list");
  const profileRows = profiles.map((profile, index) => {
    const personID = profile.person_id || profile.id || "";
    return `<div class="editor-list-item profile-row ${index === selectedProfile ? "is-active" : ""}" role="button" tabindex="0" data-profile-index="${index}">
    <span class="profile-row-main">
      <strong>${escapeHTML(profile.name || uiText("unnamedProfile"))}</strong>
      <span>${escapeHTML(personID || uiText("missingIdLabel"))} · ${profile.phases?.length || 0} ${escapeHTML(uiText("phaseCountWord"))}</span>
    </span>
    ${personID ? `<button type="button" class="row-edit-button" data-edit-person="${escapeAttribute(personID)}" title="Zmień nazwę osoby" aria-label="Zmień nazwę osoby">Edytuj</button>` : ""}
  </div>`;
  }).join("");
  const pendingRows = (snapshot.persons || [])
    .filter(person => person.active && !profiles.some(profile => (profile.person_id || profile.id) === person.id))
    .map(person => `<div class="editor-list-item profile-row is-pending ${selectedProfile < 0 && person.id === selectedPersonID ? "is-active" : ""}" role="button" tabindex="0" data-person-index="${escapeAttribute(person.id)}">
    <span class="profile-row-main">
      <strong>${escapeHTML(person.name || uiText("unnamedPerson"))}</strong>
      <span>${escapeHTML(person.id)} · ${escapeHTML(uiText("withoutProgram"))}</span>
    </span>
    <button type="button" class="row-edit-button" data-edit-person="${escapeAttribute(person.id)}" title="Zmień nazwę osoby" aria-label="Zmień nazwę osoby">Edytuj</button>
  </div>`).join("");
  profileList.innerHTML = profileRows + pendingRows || `<div class="empty-state"><span>${escapeHTML(uiText("addFirstProfile"))}</span></div>`;

  const profile = profiles[selectedProfile];
  const phaseList = document.getElementById("phase-list");
  document.getElementById("add-phase").disabled = !profile;
  if (!profile) {
    phaseList.innerHTML = "";
    document.getElementById("phase-inspector").innerHTML = `<div class="inspector-empty"><div><strong>${escapeHTML(uiText("selectAddProfile"))}</strong><p>${escapeHTML(uiText("phaseEditorHere"))}</p></div></div>`;
    return;
  }

  profile.phases ||= [];
  phaseList.innerHTML = profile.phases.length ? profile.phases.map((phase, index) => `<button class="editor-list-item ${index === selectedPhase ? "is-active" : ""}" data-phase-index="${index}">
    <strong>${escapeHTML(phase.name || uiText("unnamedPhase"))}</strong>
    <span>${phase.duration_days || 0} ${escapeHTML(uiText("dayCountWord"))} · ${escapeHTML(phase.mode === "interval" ? uiText("intervalWord") : uiText("weeklyMode"))}</span>
  </button>`).join("") : `<div class="empty-state"><span>${escapeHTML(uiText("addFirstPhase"))}</span></div>`;

  renderInspector(profile, profile.phases[selectedPhase]);
}

function renderInspector(profile, phase) {
  const inspector = document.getElementById("phase-inspector");
  if (!phase) {
    inspector.innerHTML = `<div class="inspector-header">
      <div><span class="eyebrow">PROFIL</span><h2>${escapeHTML(profile.name || "Profil bez nazwy")}</h2></div>
      <div><button class="archive-link" data-action="finish-profile">Zakończ program</button><button class="danger-link" data-action="delete-profile">Usuń profil</button></div>
    </div>
    <div class="form-field"><label>Osoba</label><input value="${escapeAttribute(profile.name || "")}" readonly></div>
    <div class="inspector-empty"><div><strong>Ten profil nie ma faz</strong><p>Dodaj fazę przyciskiem + w środkowej kolumnie.</p></div></div>`;
    return;
  }

  phase.schedule ||= {};
  phase.device_steps ||= [];
  phase.scheduling ||= {};
  const intervalFields = `<div class="form-section">
    <h3>Program interwałowy</h3>
    <div class="form-grid">
      <div class="form-field"><label>Program / częstotliwość</label><input data-field="program" value="${escapeAttribute(phase.program || "")}" placeholder="np. 30 kHz"></div>
      <div class="form-field"><label>Czas</label><input data-field="time" value="${escapeAttribute(phase.time || "")}" placeholder="np. 7 min"></div>
      <div class="form-field"><label>Dni przerwy między sesjami</label><input data-field="interval_gap" type="number" min="0" max="365" value="${Number(phase.interval_gap || 0)}"></div>
      <div class="form-field full"><label>Notatka</label><textarea data-field="note">${escapeHTML(phase.note || "")}</textarea></div>
      <div class="full" data-interval-notice>${intervalConsistencyNotice(phase)}</div>
    </div>
  </div>
  ${renderSchedulingEditor(phase.scheduling, "phase", false, phase)}
  <div class="form-section">${renderDeviceStepsEditor(phase.device_steps, "phase")}</div>`;

  const weeklyFields = `<div class="form-section">
    <h3>Rozpiska tygodniowa</h3>
    <div class="weekly-editor">${WEEKDAYS.map(key => {
      const label = localizedWeekdayName(key);
      const day = phase.schedule[key] || {};
      day.device_steps ||= [];
      day.scheduling ||= {};
      return `<div class="weekly-day-block">
        <div class="weekly-day-title">${label}</div>
        <div class="weekly-day-fields">
          <input data-day="${key}" data-day-field="freq" value="${escapeAttribute(day.freq || "")}" placeholder="Program / opis">
          <input data-day="${key}" data-day-field="time" value="${escapeAttribute(day.time || "")}" placeholder="Czas">
          <input data-day="${key}" data-day-field="note" value="${escapeAttribute(day.note || "")}" placeholder="Notatka">
        </div>
        ${renderSchedulingEditor(day.scheduling, key, true)}
        <div class="weekly-day-device">${renderDeviceStepsEditor(day.device_steps, key, true)}</div>
      </div>`;
    }).join("")}</div>
  </div>`;

  inspector.innerHTML = `<div class="inspector-header">
    <div><span class="eyebrow">${escapeHTML(profile.name)}</span><h2>${escapeHTML(phase.name || "Faza bez nazwy")}</h2></div>
    <div><button class="archive-link" data-action="finish-profile">Zakończ program</button><button class="danger-link" data-action="delete-phase">Usuń fazę</button><button class="danger-link" data-action="delete-profile">Usuń profil</button></div>
  </div>
  <div class="form-grid">
    <div class="form-field full"><label>Osoba</label><input value="${escapeAttribute(profile.name || "")}" readonly></div>
    <div class="form-field"><label>Nazwa fazy</label><input data-field="phase-name" value="${escapeAttribute(phase.name || "")}"></div>
    <div class="form-field"><label>Czas trwania w dniach</label><input data-field="duration_days" type="number" min="1" max="3650" value="${Number(phase.duration_days || 1)}"></div>
    <div class="form-field full"><label>Tryb harmonogramu</label>
      <div class="mode-switch">
        <button data-action="set-mode" data-mode="interval" class="${phase.mode === "interval" ? "is-active" : ""}">Interwałowy</button>
        <button data-action="set-mode" data-mode="weekly" class="${phase.mode !== "interval" ? "is-active" : ""}">Tygodniowy</button>
      </div>
    </div>
  </div>
  ${phase.mode === "interval" ? intervalFields : weeklyFields}`;
}

function selectProfileFromList(event) {
  const editButton = event.target.closest("[data-edit-person]");
  if (editButton) {
    event.stopPropagation();
    openPersonEditDialog(editButton.dataset.editPerson);
    return;
  }
  const personRow = event.target.closest("[data-person-index]");
  if (personRow) {
    selectPerson(personRow.dataset.personIndex);
    return;
  }
  const button = event.target.closest("[data-profile-index]");
  if (!button) return;
  selectedProfile = Number(button.dataset.profileIndex);
  const profile = draftConfig.profiles[selectedProfile];
  selectedPersonID = profile?.person_id || profile?.id || selectedPersonID;
  selectedPhase = 0;
  renderProfilesEditor();
  renderPersons();
}

function selectPhaseFromList(event) {
  const button = event.target.closest("[data-phase-index]");
  if (!button) return;
  selectedPhase = Number(button.dataset.phaseIndex);
  renderProfilesEditor();
}

async function addProfile(event) {
  const hasProfile = (draftConfig?.profiles || []).some(profile => (profile.person_id || profile.id) === selectedPersonID);
  if (!selectedPersonID || hasProfile) {
    openPersonDialog();
    return;
  }
  await runAction(event.currentTarget, async () => {
    snapshot = await window.apiCreateProfileForPerson(selectedPersonID);
    syncDraft();
    selectedProfile = draftConfig.profiles.findIndex(profile => (profile.person_id || profile.id) === selectedPersonID);
    selectedPhase = 0;
    renderAll();
    toast("Utworzono pusty program dla wybranej osoby");
  });
}

function addPhase() {
  const profile = draftConfig.profiles[selectedProfile];
  if (!profile) return;
  profile.phases ||= [];
  profile.phases.push({
    name: `Faza ${profile.phases.length + 1}`,
    duration_days: 7,
    mode: "interval",
    interval_gap: 1,
    program: "30 kHz",
    time: "7 min",
    note: "",
    device_steps: [],
    schedule: {},
  });
  selectedPhase = profile.phases.length - 1;
  setDirty();
  renderProfilesEditor();
}

function handleInspectorClick(event) {
  const button = event.target.closest("[data-action]");
  if (!button) return;
  const action = button.dataset.action;
  const profile = draftConfig.profiles[selectedProfile];
  const phase = profile?.phases?.[selectedPhase];
  if (action === "set-mode" && phase) {
    phase.mode = button.dataset.mode;
    phase.schedule ||= {};
    setDirty();
    renderProfilesEditor();
  }
  if (action === "add-step" && phase) {
    const steps = deviceStepsForOwner(phase, button.dataset.stepOwner);
    if (steps.length >= 12) {
      toast("Płytka obsługuje maksymalnie 12 kroków", true);
      return;
    }
    steps.push({ frequency_millihz: 30000000, duration_seconds: 420 });
    setDirty();
    renderProfilesEditor();
  }
  if (action === "remove-step" && phase) {
    const steps = deviceStepsForOwner(phase, button.dataset.stepOwner);
    steps.splice(Number(button.dataset.stepIndex), 1);
    setDirty();
    renderProfilesEditor();
  }
  if (action === "delete-profile" && profile) {
    if (!profile.id) {
      if (!armDestructive(button, `profile-${selectedProfile}`, "Kliknij ponownie: usuń niezapisany profil")) return;
      draftConfig.profiles.splice(selectedProfile, 1);
      selectedProfile = Math.max(0, selectedProfile - 1);
      selectedPhase = 0;
      setDirty();
      renderProfilesEditor();
      return;
    }
    if (profilesDirty) {
      toast("Najpierw zapisz zmiany w profilach", true);
      return;
    }
    if (!armDestructive(button, `profile-${profile.id}`, "Kliknij ponownie: usuń i archiwizuj")) return;
    runAction(button, async () => {
      snapshot = await window.apiDeleteProfile(profile.id);
      selectedProfile = 0;
      selectedPhase = 0;
      syncDraft();
      renderAll();
      toast("Profil usunięto — trafił do archiwum razem z historią");
    });
  }
  if (action === "finish-profile" && profile) {
    if (profilesDirty) {
      toast("Najpierw zapisz zmiany w profilach", true);
      return;
    }
    if (!profile.id) {
      toast("Najpierw zapisz profil", true);
      return;
    }
    if (!armDestructive(button, `finish-${profile.id}`, "Kliknij ponownie: przenieś do archiwum")) return;
    runAction(button, async () => {
      snapshot = await window.apiFinishProfile(profile.id);
      selectedProfile = 0;
      selectedPhase = 0;
      syncDraft();
      renderAll();
      toast("Program przeniesiono do archiwum");
    });
  }
  if (action === "delete-phase" && phase) {
    if (!armDestructive(button, `phase-${selectedProfile}-${selectedPhase}`, "Kliknij ponownie: usuń fazę")) return;
    profile.phases.splice(selectedPhase, 1);
    selectedPhase = Math.max(0, selectedPhase - 1);
    setDirty();
    renderProfilesEditor();
  }
}

function handleInspectorInput(event) {
  const input = event.target;
  const profile = draftConfig.profiles[selectedProfile];
  const phase = profile?.phases?.[selectedPhase];
  if (!profile) return;

  if (phase && input.dataset.schedulingOwner) {
    const scheduling = schedulingForOwner(phase, input.dataset.schedulingOwner);
    const field = input.dataset.schedulingField;
    if (field === "same_day_with") {
      scheduling.same_day_with = input.value.split(",").map(value => value.trim()).filter(Boolean);
    } else if (["repetitions", "break_between_minutes", "min_gap_minutes", "cooldown_after_minutes"].includes(field)) {
      scheduling[field] = Math.max(0, Number(input.value || 0));
    } else {
      scheduling[field] = input.value;
    }
    setDirty();
    if (input.dataset.schedulingOwner === "phase") refreshIntervalNotice();
    return;
  }

  const stepRow = input.closest(".device-step-row");
  if (phase && stepRow) {
    const steps = deviceStepsForOwner(phase, stepRow.dataset.stepOwner);
    const step = steps[Number(stepRow.dataset.stepIndex)];
    if (step) {
      const frequencyValue = Number(stepRow.querySelector('[data-step-field="frequency"]').value || 0);
      const unitHz = Number(stepRow.querySelector('[data-step-field="unit"]').value || 1);
      step.frequency_millihz = Math.max(0, Math.round(frequencyValue * unitHz * 1000));
      step.duration_seconds = Math.max(1, Math.round(Number(stepRow.querySelector('[data-step-field="duration"]').value || 1)));
      setDirty();
    }
    return;
  }

  if (phase && input.dataset.field === "phase-name") phase.name = input.value;
  if (phase && input.dataset.field === "duration_days") phase.duration_days = Math.max(1, Number(input.value || 1));
  if (phase && input.dataset.field === "interval_gap") phase.interval_gap = Math.max(0, Number(input.value || 0));
  if (phase && input.dataset.field === "program") phase.program = input.value;
  if (phase && input.dataset.field === "time") phase.time = input.value;
  if (phase && input.dataset.field === "note") phase.note = input.value;
  if (phase && input.dataset.day) {
    phase.schedule ||= {};
    phase.schedule[input.dataset.day] ||= { freq: "", time: "", note: "" };
    phase.schedule[input.dataset.day][input.dataset.dayField] = input.value;
  }
  setDirty();
  if (["interval_gap", "duration_days"].includes(input.dataset.field)) refreshIntervalNotice();
  if (event.type === "change" && ["profile-name", "phase-name", "duration_days"].includes(input.dataset.field)) renderProfilesEditor();
}

function renderDeviceStepsEditor(steps, owner, compact = false) {
  const rows = (steps || []).map((step, index) => {
    const display = editableFrequency(step.frequency_millihz || 0);
    return `<div class="device-step-row" data-step-owner="${escapeAttribute(owner)}" data-step-index="${index}">
      <label>Częstotliwość<input data-step-field="frequency" type="number" min="0.1" step="0.001" value="${display.value}"></label>
      <label>Jednostka<select data-step-field="unit">
        <option value="1" ${display.unit === 1 ? "selected" : ""}>Hz</option>
        <option value="1000" ${display.unit === 1000 ? "selected" : ""}>kHz</option>
        <option value="1000000" ${display.unit === 1000000 ? "selected" : ""}>MHz</option>
      </select></label>
      <label>Czas w sekundach<input data-step-field="duration" type="number" min="1" max="86400" value="${Number(step.duration_seconds || 1)}"></label>
      <button class="remove-step" data-action="remove-step" data-step-owner="${escapeAttribute(owner)}" data-step-index="${index}" aria-label="Usuń krok">×</button>
    </div>`;
  }).join("");
  return `<div class="device-steps-editor ${compact ? "is-compact" : ""}">
    <div class="device-steps-heading">
      <div><strong>${uiText("deviceSteps")}</strong>${compact ? "" : `<p>${escapeHTML(uiText("exactUsbValues"))}</p>`}</div>
      <button class="icon-button" data-action="add-step" data-step-owner="${escapeAttribute(owner)}" aria-label="Dodaj krok urządzenia">+</button>
    </div>
    ${rows || `<span class="table-secondary">Brak kroków — ta sesja nie może być jeszcze wysłana do płytki.</span>`}
  </div>`;
}

// intervalConsistencyNotice liczy okres planu na żywo, żeby użytkownik zobaczył
// sprzeczność JESZCZE PRZED zapisem — backend odrzuca zapis takiej konfiguracji,
// a bez tego dowiadywałby się o niej dopiero z komunikatu błędu.
function intervalConsistencyNotice(phase) {
  if (!phase || phase.mode !== "interval") return "";
  const gapDays = Math.max(0, Number(phase.interval_gap || 0));
  const durationDays = Math.max(1, Number(phase.duration_days || 1));
  // Faza mieszcząca tylko jeden termin nie ma "następnej" sesji — nic do porównania.
  if (durationDays <= gapDays + 1) return "";
  const periodMinutes = (gapDays + 1) * 1440;
  const minGap = Math.max(0, Number(phase.scheduling?.min_gap_minutes || 0));
  const period = `co ${gapDays + 1}. dzień, czyli co ${periodMinutes} min`;
  if (minGap > periodMinutes) {
    return `<div class="ai-import-warning">
      <strong>Plan niewykonalny — zapis zostanie odrzucony</strong>
      <span>Sesje są planowane ${period}, a minimalny odstęp między pełnymi sesjami wynosi ${minGap} min. Zmniejsz odstęp do najwyżej ${periodMinutes} min albo zwiększ liczbę dni przerwy.</span>
    </div>`;
  }
  if (minGap > 0 && minGap === periodMinutes) {
    return `<div class="ai-import-warning">
      <strong>Plan bez zapasu czasu</strong>
      <span>Minimalny odstęp ${minGap} min jest dokładnie równy okresowi planu (${period}). Sesje trzeba wykonywać co do minuty — każde opóźnienie przesunie kolejną sesję na następny dzień.</span>
    </div>`;
  }
  return "";
}

// Ostrzeżenie odświeżamy punktowo, bez przerysowania inspektora — pełny render
// zabrałby fokus z pola, w którym użytkownik właśnie pisze.
function refreshIntervalNotice() {
  const phase = draftConfig?.profiles?.[selectedProfile]?.phases?.[selectedPhase];
  const html = intervalConsistencyNotice(phase);
  document.querySelectorAll("[data-interval-notice]").forEach(node => { node.innerHTML = html; });
}

function renderSchedulingEditor(scheduling = {}, owner, compact = false, intervalPhase = null) {
  return `<details class="scheduling-editor form-section" ${compact ? "" : "open"}>
    <summary>Reguły kolejki i przerw</summary>
    <p>Jedno ID łączy wersję pojedynczą i wieloczęściową. Odstęp programu działa między pełnymi sesjami, a przerwa tylko wewnątrz serii. Zgodność musi być wpisana po obu stronach.</p>
    <div class="scheduling-grid">
      <label>ID programu<input data-scheduling-owner="${escapeAttribute(owner)}" data-scheduling-field="program_id" value="${escapeAttribute(scheduling.program_id || "")}" placeholder="np. program_a"></label>
      <label>Liczba części<input type="number" min="1" max="12" data-scheduling-owner="${escapeAttribute(owner)}" data-scheduling-field="repetitions" value="${Number(scheduling.repetitions || 1)}"></label>
      <label>Przerwa między częściami (min)<input type="number" min="0" max="1440" data-scheduling-owner="${escapeAttribute(owner)}" data-scheduling-field="break_between_minutes" value="${Number(scheduling.break_between_minutes || 0)}"></label>
      <label>Odstęp między pełnymi sesjami (min)<input type="number" min="0" max="43200" data-scheduling-owner="${escapeAttribute(owner)}" data-scheduling-field="min_gap_minutes" value="${Number(scheduling.min_gap_minutes || 0)}"></label>
      ${intervalPhase ? `<div class="full" data-interval-notice>${intervalConsistencyNotice(intervalPhase)}</div>` : ""}
      <label>Cooldown po sesji (min)<input type="number" min="0" max="43200" data-scheduling-owner="${escapeAttribute(owner)}" data-scheduling-field="cooldown_after_minutes" value="${Number(scheduling.cooldown_after_minutes || 0)}"></label>
      <label class="full">Zgodne tego samego dnia — ID po przecinku<input data-scheduling-owner="${escapeAttribute(owner)}" data-scheduling-field="same_day_with" value="${escapeAttribute((scheduling.same_day_with || []).join(", "))}" placeholder="program_b, program_c"></label>
    </div>
  </details>`;
}

function deviceStepsForOwner(phase, owner) {
  if (owner === "phase") {
    phase.device_steps ||= [];
    return phase.device_steps;
  }
  phase.schedule ||= {};
  phase.schedule[owner] ||= { freq: "", time: "", note: "", device_steps: [] };
  phase.schedule[owner].device_steps ||= [];
  return phase.schedule[owner].device_steps;
}

function schedulingForOwner(phase, owner) {
  if (owner === "phase") {
    phase.scheduling ||= {};
    return phase.scheduling;
  }
  phase.schedule ||= {};
  phase.schedule[owner] ||= { freq: "", time: "", note: "", device_steps: [], scheduling: {} };
  phase.schedule[owner].scheduling ||= {};
  return phase.schedule[owner].scheduling;
}

function editableFrequency(frequencyMilliHz) {
  const value = Number(frequencyMilliHz || 0);
  let unit = 1;
  if (value >= 1000000000) unit = 1000000;
  else if (value >= 1000000) unit = 1000;
  return { value: Number((value / (unit * 1000)).toFixed(6)), unit };
}

async function saveProfiles(event) {
  await runAction(event.currentTarget, async () => {
    snapshot = await window.apiSaveConfig(draftConfig);
    syncDraft();
    renderAll();
    toast("Profile i harmonogram zostały zapisane");
  });
}

function renderHistory() {
  const query = document.getElementById("history-search").value.trim().toLocaleLowerCase("pl");
  const rows = [];
  const modernKeys = new Set();
  Object.values(snapshot.progress.completions || {}).forEach(entry => {
    const date = String(entry.completed_at || entry.planned_date || "").slice(0, 10);
    modernKeys.add(`${date}|${entry.profile_name}`);
    rows.push({
      date,
      person: entry.profile_name,
      therapy: entry.therapy,
      time: entry.time,
      repetitions: entry.repetitions,
      phase: entry.phase,
      notes: entry.notes,
      completed_at: entry.completed_at,
      // Przebieg dzielony pauzą nie jest tym samym co jedno ciągłe wykonanie.
      kind: entry.split
        ? (entry.first_started_at
          ? uiFormat("dynSplitStarted", { date: formatShortDate(String(entry.first_started_at).slice(0, 10)) })
          : uiText("splitKind", "Split"))
        : uiText("scheduledKind", "Scheduled"),
    });
  });
  Object.entries(snapshot.progress.history || {}).forEach(([date, people]) => {
    Object.entries(people || {}).forEach(([person, entry]) => {
      if (entry?.done && !modernKeys.has(`${date}|${person}`)) rows.push({ date, person, ...entry, kind: uiText("scheduledKind", "Scheduled") });
    });
  });
  // Ręczne uruchomienia trybu ręcznego: widoczne, ale wyraźnie odróżnione od planu.
  (snapshot.manual_runs || []).forEach(run => {
    rows.push({
      date: String(run.started_at || "").slice(0, 10),
      person: uiText("manualModeDash", "— manual mode —"),
      therapy: formatFrequency(run.frequency_millihz),
      time: formatDuration(run.duration_seconds || 0),
      repetitions: "1x",
      phase: "-",
      notes: uiText("manualOutsidePlan", "Started manually outside the plan — it does not complete any scheduled session."),
      completed_at: run.started_at,
      kind: uiText("manualKind", "Manual"),
      kindCode: "manual",
    });
  });
  // Zaległości świadomie odpuszczone: NIE są wykonaniem, ale muszą być widoczne.
  Object.values(snapshot.progress.dismissed_sessions || {}).forEach(entry => {
    rows.push({
      date: String(entry.dismissed_at || entry.planned_date || "").slice(0, 10),
      person: entry.profile_name,
      therapy: entry.therapy,
      time: "-",
      repetitions: "-",
      phase: entry.phase,
      notes: uiFormat("dynDismissedOverdueFrom", { date: formatShortDate(entry.planned_date) }),
      completed_at: entry.dismissed_at,
      kind: uiText("dismissedKind", "Dismissed"),
      kindCode: "dismissed",
    });
  });
  // Częściowe wykonania przerwane przyciskiem Zatrzymaj: sesja wciąż czeka na
  // dokończenie, ale wiadomo ile zrobiono, ile zostało i kiedy to się działo.
  (snapshot.progress.partial_runs || []).forEach(run => {
    rows.push({
      date: String(run.recorded_at || run.planned_date || "").slice(0, 10),
      person: run.profile_name,
      therapy: run.therapy,
      time: formatDuration(run.done_seconds || 0),
      repetitions: "-",
      phase: run.phase,
      notes: uiFormat("dynPartialExecution", {
        done: formatDuration(run.done_seconds || 0),
        total: formatDuration(run.total_seconds || 0),
        remaining: formatDuration(run.remaining_seconds || 0)
      }),
      completed_at: run.recorded_at,
      kind: uiText("partialKind", "Partial"),
      kindCode: "partial",
    });
  });
  rows.sort((a, b) => String(b.completed_at || b.date).localeCompare(String(a.completed_at || a.date)) || a.person.localeCompare(b.person, "pl"));
  const filtered = rows.filter(row => !query || [row.date, row.person, row.therapy, row.phase, row.notes, row.kind].join(" ").toLocaleLowerCase("pl").includes(query));
  // Licznik dotyczy WYŁĄCZNIE faktycznie wykonanych sesji planu. Przebiegi ręczne
  // i odpuszczone zaległości są widoczne w tabeli, ale nie są wykonaniem planu.
  const performed = rows.filter(row => !["manual", "dismissed", "partial"].includes(row.kindCode)).length;
  const extra = rows.length - performed;
  document.getElementById("history-count").textContent = extra
    ? uiFormat("dynHistoryPerformedExtra", { count: performed, extra })
    : uiFormat("dynHistoryPerformed", { count: performed });
  document.getElementById("history-body").innerHTML = filtered.length ? filtered.map(row => `<tr class="${row.kindCode === "manual" ? "is-manual" : row.kindCode === "dismissed" ? "is-dismissed" : ""}">
    <td><span class="table-primary">${formatShortDate(row.date)}</span></td>
    <td>${escapeHTML(row.person)}</td>
    <td><span class="table-primary">${escapeHTML(row.therapy || "-")}</span><span class="table-secondary">${escapeHTML(row.repetitions || "-")}</span></td>
    <td>${escapeHTML(row.time || "-")}</td>
    <td>${escapeHTML(row.phase || "-")}</td>
    <td>${escapeHTML(row.notes || "")}</td>
    <td>${escapeHTML(row.kind || uiText("scheduledKind", "Scheduled"))}</td>
  </tr>`).join("") : `<tr><td colspan="7"><div class="empty-state"><strong>${escapeHTML(uiText("noResults", "No results"))}</strong><span>${escapeHTML(uiText("historyAfterSession", "History will appear after the first session is marked completed."))}</span></div></td></tr>`;
}

function renderArchive() {
  const container = document.getElementById("archive-list");
  if (!container) return;
  const profiles = [...(snapshot.archive || [])].sort((a, b) => String(b.finished_at).localeCompare(String(a.finished_at)));
  document.getElementById("archive-count").textContent = profiles.length
    ? `${profiles.length} ${uiText("completedProgramCountWord", "completed program(s)")} · ${uiText("archiveCountHint", "click an entry to view the saved plan")}`
    : uiText("noCompletedPrograms", "No program has been completed yet.");
  container.innerHTML = profiles.length ? profiles.map(renderArchiveEntry).join("") : `<div class="empty-state"><strong>${escapeHTML(uiText("archiveEmpty", "Archive is empty"))}</strong><span>${escapeHTML(uiText("archiveManualHelp", "A manually completed profile will be kept here with its summary."))}</span></div>`;
}

function archiveReasonText(reason) {
  const value = String(reason || "").trim();
  if (!value || value === "Zakończony") return uiText("finishedWord", "Completed");
  if (value === "Zakończony ręcznie") return uiText("finishedManually", "Finished manually");
  return localizedSource(value);
}

function renderArchiveEntry(entry) {
  const profile = entry.profile || {};
  const phases = profile.phases || [];
  const currentMatch = (snapshot.config?.profiles || []).find(candidate => archiveIdentityKey(candidate.name) === archiveIdentityKey(profile.name));
  const identityMismatch = currentMatch && currentMatch.id !== profile.id;
  const startDate = entry.progress?.start_date || "";
  const timeline = archiveTimeline(profile, startDate);
  const completions = Object.values(entry.progress?.completions || {}).sort((a, b) => String(a.completed_at).localeCompare(String(b.completed_at)));
  const totalDays = phases.reduce((sum, phase) => sum + Math.max(0, Number(phase.duration_days || 0)), 0);
  return `<details class="archive-entry">
    <summary class="archive-row">
      <div class="archive-mark">${escapeHTML(initials(profile.name))}</div>
      <div class="archive-identity">
        <strong>${escapeHTML(profile.name || uiText("profileUpper", "Profile"))}</strong>
        <span>${phases.length} ${escapeHTML(uiText("phaseCountWord", "phases"))} · ${totalDays} ${escapeHTML(uiText("dayCountWord", "days"))} · ${entry.completed_count || 0} ${escapeHTML(uiText("sessionCountWord", "completed sessions"))}</span>
        <small>${escapeHTML(uiText("personIdLabel", "Person ID:"))} <code>${escapeHTML(profile.id || uiText("missingId", "none"))}</code>${identityMismatch ? `<em>${escapeHTML(uiText("differentCurrentId", "Different ID from the current profile"))}</em>` : ""}</small>
      </div>
      <div><strong>${formatShortDate(String(entry.finished_at || "").slice(0, 10))}</strong><span>${escapeHTML(archiveReasonText(entry.reason))}</span></div>
      <div><strong>${startDate ? `${escapeHTML(uiText("startLabel", "Start"))} ${formatShortDate(startDate)}` : escapeHTML(uiText("noStartDate", "No start date"))}</strong><span>${escapeHTML(uiText("runLabel", "Run:"))} ${escapeHTML(profile.run_id || uiText("legacyFormat", "legacy format"))}</span></div>
      <span class="archive-chevron" aria-hidden="true"></span>
    </summary>
    <div class="archive-detail">
      <div class="archive-detail-heading">
        <div><span class="section-label">${escapeHTML(uiText("savedRunUpper", "SAVED RUN"))}</span><h2>${escapeHTML(uiText("dayByDayPlan", "Day-by-day plan"))}</h2></div>
        <p>${escapeHTML(uiText("archivedPlanImmutable", "This is an immutable copy of the plan from the moment it was completed."))}</p>
      </div>
      <div class="archive-detail-layout">
        <section class="archive-timeline" aria-label="${escapeAttribute(uiText("dayByDayPlan", "Day-by-day plan"))}">
          ${timeline || `<div class="archive-detail-empty">${escapeHTML(uiText("noDaysSaved", "No days in the saved profile."))}</div>`}
        </section>
        <aside class="archive-inspector">
          <section>
            <span class="section-label">${escapeHTML(uiText("phasesUpper", "PHASES"))}</span>
            ${phases.map((phase, index) => renderArchivedPhase(phase, index)).join("") || `<p class="archive-detail-empty">${escapeHTML(uiText("noPhasesSaved", "No phases."))}</p>`}
          </section>
          <section>
            <span class="section-label">${escapeHTML(uiText("executionsUpper", "EXECUTIONS"))}</span>
            ${completions.length ? completions.map(renderArchivedCompletion).join("") : `<p class="archive-detail-empty">${escapeHTML(uiText("noCompletionsSaved", "No completed session was recorded in this run."))}</p>`}
          </section>
        </aside>
      </div>
    </div>
  </details>`;
}

function archiveIdentityKey(value) {
  return String(value || "")
    .replace(/\([^)]*\)/g, " ")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLocaleLowerCase("pl")
    .replace(/[^a-z0-9]+/g, " ")
    .trim();
}

function archiveTimeline(profile, startDate) {
  const parsedStart = /^\d{4}-\d{2}-\d{2}$/.test(startDate || "") ? new Date(`${startDate}T00:00:00Z`) : null;
  const weekdays = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
  let absoluteDay = 0;
  const rows = [];
  (profile.phases || []).forEach(phase => {
    const duration = Math.max(0, Number(phase.duration_days || 0));
    for (let phaseDay = 0; phaseDay < duration; phaseDay += 1) {
      const date = parsedStart ? new Date(parsedStart.getTime() + absoluteDay * 86400000) : null;
      let program = uiText("dayBreak", "Rest day");
      let time = "";
      let note = "";
      let scheduling = {};
      let session = false;
      if (phase.mode === "interval") {
        const gap = Math.max(0, Number(phase.interval_gap || 0));
        session = phaseDay % (gap + 1) === 0;
        if (session) {
          program = phase.program || uiText("noPlan", "No program");
          time = phase.time || "";
          note = phase.note || "";
          scheduling = phase.scheduling || {};
        }
      } else if (date) {
        const daily = phase.schedule?.[weekdays[date.getUTCDay()]];
        if (daily?.freq) {
          session = true;
          program = daily.freq;
          time = daily.time || "";
          note = daily.note || "";
          scheduling = daily.scheduling || {};
        }
      }
      const repetitions = Math.max(1, Number(scheduling.repetitions || 1));
      rows.push(`<div class="archive-day ${session ? "has-session" : "is-break"}">
        <span class="archive-day-number">${String(absoluteDay + 1).padStart(2, "0")}</span>
        <span class="archive-day-date">${date ? formatShortDate(date.toISOString().slice(0, 10)) : `${escapeHTML(uiText("day", "Day"))} ${absoluteDay + 1}`}</span>
        <div><strong>${escapeHTML(program)}</strong><span>${escapeHTML(phase.name || uiText("phaseFallback", "Phase"))}${note ? ` · ${escapeHTML(note)}` : ""}</span></div>
        <span class="archive-day-time">${session ? `${escapeHTML(time || "-")} · ${repetitions}x` : escapeHTML(uiText("restLabel", "Rest"))}</span>
      </div>`);
      absoluteDay += 1;
    }
  });
  return rows.join("");
}

function renderArchivedPhase(phase, index) {
  const every = Math.max(0, Number(phase.interval_gap || 0)) + 1;
  let mode;
  if (phase.mode === "interval") {
    mode = uiFormat("dynEveryDays", { days: every });
  } else {
    mode = uiText("byWeekdays", "by weekdays");
  }
  return `<div class="archive-phase-record">
    <span>${escapeHTML(uiText("phaseFallback", "Phase"))} ${index + 1}</span>
    <strong>${escapeHTML(phase.name || uiText("unnamed", "Unnamed"))}</strong>
    <p>${escapeHTML(uiFormat("dynPhaseDurationMode", { days: Number(phase.duration_days || 0), mode }))}</p>
  </div>`;
}

function renderArchivedCompletion(entry) {
  return `<div class="archive-completion">
    <span>${formatShortDate(String(entry.completed_at || entry.planned_date || "").slice(0, 10))}</span>
    <strong>${escapeHTML(entry.therapy || uiText("manualSession", "Session"))}</strong>
    <small>${escapeHTML(entry.phase || "")} ${entry.time ? `· ${escapeHTML(entry.time)}` : ""}</small>
  </div>`;
}

function renderFrequencies() {
  const catalog = localizedFrequencies || snapshot.frequencies || [];
  const locale = window.ZapperI18n?.locale || LANGUAGE_LOCALES[preferredLanguage] || "en-US";
  const query = document.getElementById("frequency-search").value.trim().toLocaleLowerCase(locale);
  const entries = catalog.filter(item => !query || [item.name, item.frequency, item.description].join(" ").toLocaleLowerCase(locale).includes(query));
  const t = key => window.ZapperI18n?.t(key) || key;
  document.getElementById("frequency-count").textContent = query
    ? uiFormat("dynCatalogFiltered", { shown: entries.length, total: catalog.length })
    : uiFormat("dynCatalogCount", { count: entries.length });
  document.getElementById("frequency-list").innerHTML = entries.length ? entries.map(item => `<article class="frequency-row">
    <strong>${escapeHTML(item.name)}</strong>
    <span class="frequency-value">${escapeHTML(item.frequency)}</span>
    <p>${escapeHTML(item.description)}</p>
  </article>`).join("") : `<div class="empty-state"><strong>${escapeHTML(t("noResults"))}</strong><span>${escapeHTML(t("changeSearch"))}</span></div>`;
}

function setDirty() {
  profilesDirty = true;
  updateDirtyState();
}

function updateDirtyState() {
  document.getElementById("profiles-dirty").classList.toggle("is-visible", profilesDirty);
  const state = document.getElementById("save-state");
  state.textContent = profilesDirty ? uiText("unsavedChanges") : uiText("allSaved");
  state.classList.toggle("is-dirty", profilesDirty);
  document.getElementById("save-profiles").disabled = !profilesDirty;
}

async function runAction(button, action) {
  if (button.disabled) return;
  button.disabled = true;
  clearError();
  try {
    await action();
  } catch (error) {
    showError(normalizeError(error));
    toast(normalizeError(error), true);
    // #error-banner i #toast leza w normalnym przeplywie dokumentu, wiec gdy
    // otwarty jest modal <dialog>, sa rysowane POD nim i pod jego ::backdrop —
    // uzytkownik nie widzial zadnego komunikatu i przycisk wygladal na martwy.
    // Baner diagnostyczny trafia do top layer, wiec jest widoczny takze wtedy.
    window.zapperReportError("akcja zakonczona bledem: " + (button.dataset.deviceSession || button.textContent || ""), error);
  } finally {
    button.disabled = false;
    if (document.getElementById("device-state-badge")) renderDeviceStatus();
  }
}

function armDestructive(button, key, message) {
  if (armedDelete === key) {
    armedDelete = "";
    clearTimeout(armedDeleteTimer);
    button.classList.remove("is-confirming");
    return true;
  }
  armedDelete = key;
  const original = button.textContent;
  button.textContent = message;
  button.classList.add("is-confirming");
  toast(message);
  clearTimeout(armedDeleteTimer);
  armedDeleteTimer = setTimeout(() => {
    armedDelete = "";
    if (button.isConnected) {
      button.textContent = original;
      button.classList.remove("is-confirming");
    }
  }, DESTRUCTIVE_ARM_MS);
  return false;
}

function showError(message) {
  const banner = document.getElementById("error-banner");
  banner.textContent = message;
  banner.classList.add("is-visible");
}

function clearError() {
  document.getElementById("error-banner").classList.remove("is-visible");
}

function showFatal(error) {
  const loading = document.getElementById("loading-screen");
  loading.innerHTML = `<div class="loading-mark">!</div><strong>${escapeHTML(uiText("openAppFailed", "Could not open the application"))}</strong><p>${escapeHTML(normalizeError(error))}</p>`;
}

function toast(message, isError = false) {
  const element = document.getElementById("toast");
  element.textContent = message;
  element.classList.toggle("is-error", isError);
  element.classList.add("is-visible");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => element.classList.remove("is-visible"), 2800);
}

function normalizeError(error) {
  if (!error) return uiText("unknownError", "An unknown error occurred.");
  if (typeof error === "string") return error;
  return error.message || String(error);
}

function statusText(status, done) {
  const t = key => window.ZapperI18n?.t(key) || key;
  if (done) return t("completed");
  return ({ session: t("toDo"), break: t("breakStatus"), before: t("beforeStart"), complete: t("finishedWord"), empty: t("noPlan") })[status] || status;
}

function initials(name) {
  return String(name || "?").split(/\s+/).filter(Boolean).slice(0, 2).map(part => part[0]).join("").toUpperCase();
}

function formatShortDate(value) {
  const [year, month, day] = String(value).split("-").map(Number);
  if (!year || !month || !day) return value;
  const locale = window.ZapperI18n?.locale || LANGUAGE_LOCALES[preferredLanguage] || "en-US";
  return new Intl.DateTimeFormat(locale, { year: "numeric", month: "2-digit", day: "2-digit", timeZone: "UTC" })
    .format(new Date(Date.UTC(year, month - 1, day)));
}

function formatDuration(seconds) {
  const total = Math.max(0, Math.round(Number(seconds || 0)));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const remainder = total % 60;
  const locale = window.ZapperI18n?.locale || LANGUAGE_LOCALES[preferredLanguage] || "en-US";
  const unit = (value, name) => new Intl.NumberFormat(locale, {
    style: "unit", unit: name, unitDisplay: "short", maximumFractionDigits: 0
  }).format(value);
  if (hours) return [unit(hours, "hour"), minutes ? unit(minutes, "minute") : ""].filter(Boolean).join(" ");
  if (minutes) return [unit(minutes, "minute"), remainder ? unit(remainder, "second") : ""].filter(Boolean).join(" ");
  return unit(remainder, "second");
}

function formatFrequency(frequencyMilliHz) {
  const milliHz = Number(frequencyMilliHz || 0);
  if (milliHz >= 1000000000) return `${trimNumber(milliHz / 1000000000)} MHz`;
  if (milliHz >= 1000000) return `${trimNumber(milliHz / 1000000)} kHz`;
  return `${trimNumber(milliHz / 1000)} Hz`;
}

function trimNumber(value) {
  const locale = window.ZapperI18n?.locale || LANGUAGE_LOCALES[preferredLanguage] || "en-US";
  return Number(value.toFixed(6)).toLocaleString(locale, { maximumFractionDigits: 6 });
}

function formatClock(seconds) {
  const total = Math.max(0, Math.round(Number(seconds || 0)));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const remainder = total % 60;
  return hours
    ? `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`
    : `${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`;
}

function deviceStateLabel(state) {
  const t = key => window.ZapperI18n?.t(key) || key;
  return ({
    disconnected: t("disconnected"),
    connecting: t("connecting"),
    reconnecting: t("recovery"),
    idle: t("ready"),
    starting: t("starting"),
    running: t("working"),
    stopping: t("stopping"),
    armed: t("waitingConfirmation"),
    done: t("done"),
    error: t("error"),
  })[state] || state || "—";
}

function calculateDayNumber(value) {
  if (!value) return 1;
  const start = new Date(`${value}T00:00:00`);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  return Math.floor((today - start) / 86400000) + 1;
}

function polishCount(value, singular, pluralTwo, pluralMany) {
  if (value === 1) return singular;
  const lastTwo = value % 100;
  const last = value % 10;
  if (last >= 2 && last <= 4 && !(lastTwo >= 12 && lastTwo <= 14)) return pluralTwo;
  return pluralMany;
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>'"]/g, character => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" })[character]);
}

function escapeAttribute(value) {
  return escapeHTML(value).replace(/`/g, "&#96;");
}
