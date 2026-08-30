from pathlib import Path
from html.parser import HTMLParser
import json, re, subprocess, sys

LANGUAGES = [
    "en","pl","de","fr","es","it","pt","nl","sv","no","da","fi","cs","sk","hu","ro","tr","id","ms","vi",
    "ru","uk","bg","el","ar","he","hi","zh","ja","ko"
]
ROOT = Path(__file__).resolve().parents[1]
LOCALES = ROOT / "locales"


def fail(message):
    print("I18N ERROR:", message, file=sys.stderr)
    raise SystemExit(1)


def load_json(path):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        fail(f"{path}: {exc}")


info = json.loads(subprocess.check_output(["node", "tools/i18n_introspect.js"], cwd=ROOT, text=True, encoding="utf-8"))
text = info["text"]
packs = info["packs"]
all_keys = set(text)
placeholder = re.compile(r"\{([A-Za-z0-9_]+)\}")

# UI: every language must resolve every key. en/pl are source languages; all
# others combine the hand-written seed pack with the committed locale package.
for code in LANGUAGES:
    if code == "pl":
        effective = {key: pair[0] for key, pair in text.items()}
    elif code == "en":
        effective = {key: pair[1] for key, pair in text.items()}
    else:
        path = LOCALES / f"ui.{code}.json"
        if not path.is_file():
            fail(f"missing UI locale {path}")
        locale_pack = load_json(path)
        if not isinstance(locale_pack, dict):
            fail(f"UI locale {code} is not an object")
        effective = dict(packs.get(code) or {})
        effective.update(locale_pack)
    missing = sorted(all_keys - set(effective))
    if missing:
        fail(f"UI {code}: missing {len(missing)} keys: {', '.join(missing[:12])}")
    for key in all_keys:
        value = effective.get(key)
        if not isinstance(value, str) or not value.strip():
            fail(f"UI {code}: empty value for {key}")
        expected_vars = set(placeholder.findall(text[key][1]))
        actual_vars = set(placeholder.findall(value))
        if expected_vars != actual_vars:
            fail(f"UI {code}/{key}: placeholders {sorted(actual_vars)} != {sorted(expected_vars)}")

# Guide: every visible source fragment and translated attribute must exist in
# every non-Polish package. The guide has no fallback in release builds.
guide_source = json.loads(subprocess.check_output([sys.executable, "tools/guide_source.py"], cwd=ROOT, text=True, encoding="utf-8"))
guide_keys = set(guide_source)
for code in LANGUAGES:
    if code == "pl":
        continue
    path = LOCALES / f"guide.{code}.json"
    if not path.is_file():
        fail(f"missing guide locale {path}")
    data = load_json(path)
    missing = sorted(guide_keys - set(data))
    if missing:
        fail(f"guide {code}: missing {len(missing)} fragments: {missing[:6]}")
    for key in guide_keys:
        if not isinstance(data.get(key), str) or not data[key].strip():
            fail(f"guide {code}: empty translation for {key!r}")

# Frequency database: Polish comes from the main application data; all other
# languages have a bundled catalog. Every bundled catalog must have the same
# number of entries and the same frequency values as English.
assets = ROOT / "app" / "assets"
english_catalog = load_json(assets / "frequencies.en.json")
if not isinstance(english_catalog, list) or not english_catalog:
    fail("English frequency catalog is missing or empty")
english_freqs = [str(item.get("frequency", "")) for item in english_catalog]
for code in LANGUAGES:
    if code == "pl":
        continue
    path = assets / f"frequencies.{code}.json"
    if not path.is_file():
        fail(f"missing frequency catalog {path}")
    data = load_json(path)
    if not isinstance(data, list) or len(data) != len(english_catalog):
        fail(f"frequency catalog {code}: {len(data) if isinstance(data, list) else 'invalid'} entries, expected {len(english_catalog)}")
    if [str(item.get("frequency", "")) for item in data] != english_freqs:
        fail(f"frequency catalog {code}: frequency values/order differ from English")
    for index, item in enumerate(data):
        if not str(item.get("name", "")).strip() or not str(item.get("description", "")).strip():
            fail(f"frequency catalog {code}: blank text at index {index}")

# Static index: natural-language text must be registered in TEXT so later DOM
# changes cannot silently leave a Polish/English island in another language.
def normalize_ws(value):
    return re.sub(r"\s+", " ", str(value or "")).strip()

known_sources = {normalize_ws(pair[0]) for pair in text.values()} | {normalize_ws(pair[1]) for pair in text.values()}
technical = {"Z", "Zapper", "Hz", "kHz", "MHz", "+", "×", "—", ".", "00:00", "5.1.0", "v10.3.1"}

class IndexParser(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.skip = 0
        self.values = []
    def handle_starttag(self, tag, attrs):
        if tag.lower() in {"script", "style"}:
            self.skip += 1
        if not self.skip:
            for key, value in attrs:
                if key.lower() in {"placeholder", "title", "aria-label"} and value:
                    self.add(value)
    def handle_endtag(self, tag):
        if tag.lower() in {"script", "style"} and self.skip:
            self.skip -= 1
    def handle_data(self, data):
        if not self.skip:
            self.add(data)
    def add(self, value):
        value = normalize_ws(value)
        if value:
            self.values.append(value)

parser = IndexParser()
parser.feed((ROOT / "app" / "web" / "index.html").read_text(encoding="utf-8"))
for value in dict.fromkeys(parser.values):
    if value in technical or value in known_sources or re.fullmatch(r"v?\d+(?:\.\d+)*", value):
        continue
    if re.search(r"[A-Za-zĄĆĘŁŃÓŚŹŻąćęłńóśźż]{2,}", value):
        fail(f"index.html text is not registered in i18n: {value!r}")

print(f"I18N OK: {len(LANGUAGES)} languages, {len(all_keys)} UI keys, {len(guide_keys)} guide fragments, {len(english_catalog)} frequency entries")
