from html.parser import HTMLParser
from pathlib import Path
import json, re, sys

SKIP = {"script", "style", "code", "pre"}

class Parser(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.skip_depth = 0
        self.values = []
    def handle_starttag(self, tag, attrs):
        if tag.lower() in SKIP:
            self.skip_depth += 1
        if self.skip_depth == 0:
            for key, value in attrs:
                if key.lower() in {"title", "aria-label", "alt", "placeholder"} and value:
                    self.add(value)
    def handle_startendtag(self, tag, attrs):
        if self.skip_depth == 0:
            for key, value in attrs:
                if key.lower() in {"title", "aria-label", "alt", "placeholder"} and value:
                    self.add(value)
    def handle_endtag(self, tag):
        if tag.lower() in SKIP and self.skip_depth:
            self.skip_depth -= 1
    def handle_data(self, data):
        if self.skip_depth == 0:
            self.add(data)
    def add(self, value):
        value = re.sub(r"\s+", " ", str(value or "")).strip()
        if value and value not in self.values:
            self.values.append(value)

html = Path("app/instrukcja.html").read_text(encoding="utf-8")
p = Parser(); p.feed(html)
# The HTML <title> is outside body but is still user-visible in print/browser contexts.
for m in re.finditer(r"<title>(.*?)</title>", html, re.I | re.S):
    p.add(m.group(1))

payload = {value: value for value in p.values}
text = json.dumps(payload, ensure_ascii=True, separators=(",", ":"))
sys.stdout.write(text)
