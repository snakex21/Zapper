from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
MAIN = ROOT / "README.md"
README_DIR = ROOT / "README"
CODES = [
    "pl","de","fr","es","it","pt","nl","sv","no","da","fi","cs","sk","hu","ro","tr","id","ms","vi",
    "ru","uk","bg","el","ar","he","hi","zh","ja","ko"
]


def fail(message: str) -> None:
    print(f"README ERROR: {message}", file=sys.stderr)
    raise SystemExit(1)


def read(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except Exception as exc:
        fail(f"cannot read {path.relative_to(ROOT)} as UTF-8: {exc}")


def structure(text: str):
    lines = text.splitlines()
    headings = [len(m.group(1)) for line in lines if (m := re.match(r"^(#{1,6})\s+", line))]
    numbered = sum(bool(re.match(r"^\d+\.\s+", line)) for line in lines)
    bullets = sum(bool(re.match(r"^-\s+", line)) for line in lines)
    fences = sum(line.startswith("```") for line in lines)
    return headings, numbered, bullets, fences


main = read(MAIN)
expected_structure = structure(main)
expected_codes = set(CODES)

root_bar = main.splitlines()[0] if main.splitlines() else ""
if not root_bar.startswith("**Languages:**"):
    fail("README.md does not start with the language navigation bar")
root_links = re.findall(r"\[[^\]]+\]\(([^)]+)\)", root_bar)
if len(root_links) != 30:
    fail(f"README.md language bar has {len(root_links)} links, expected 30")
for target in root_links:
    if not (ROOT / target).is_file():
        fail(f"README.md language link does not exist: {target}")

expected_files = {f"README.{code}.md" for code in CODES}
actual_files = {path.name for path in README_DIR.glob("README.*.md")}
missing_files = sorted(expected_files - actual_files)
extra_files = sorted(actual_files - expected_files)
if missing_files:
    fail(f"missing localized README files: {', '.join(missing_files)}")
if extra_files:
    fail(f"unexpected localized README files: {', '.join(extra_files)}")

# Technical literals enclosed in inline Markdown code are intentionally identical
# in all languages. This catches accidental translation of paths, commands and IDs.
technical_tokens = set(re.findall(r"`([^`\n]+)`", main))

for code in CODES:
    path = README_DIR / f"README.{code}.md"
    text = read(path)
    if "ZXQ" in text:
        fail(f"{path.name}: contains a leftover placeholder marker")
    if structure(text) != expected_structure:
        fail(
            f"{path.name}: Markdown structure {structure(text)} differs from main README {expected_structure}"
        )
    lines = text.splitlines()
    if not lines or not lines[0].startswith("**Languages:**"):
        fail(f"{path.name}: missing language navigation bar")
    links = re.findall(r"\[[^\]]+\]\(([^)]+)\)", lines[0])
    if len(links) != 30:
        fail(f"{path.name}: language bar has {len(links)} links, expected 30")
    for target in links:
        resolved = (README_DIR / target).resolve()
        if not resolved.is_file():
            fail(f"{path.name}: broken language link: {target}")
    present_tokens = set(re.findall(r"`([^`\n]+)`", text))
    missing_tokens = sorted(technical_tokens - present_tokens)
    if missing_tokens:
        fail(f"{path.name}: missing technical literals: {', '.join(missing_tokens)}")

print(
    f"README OK: 30/30 languages, 29 localized files, "
    f"structure={expected_structure}, technical literals={len(technical_tokens)}"
)
