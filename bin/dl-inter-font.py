#!/usr/bin/env python3
"""Download Inter WOFF2 files and rewrite CSS for local hosting."""
import os, re, subprocess, urllib.request

OUT = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "web", "static", "fonts")
os.makedirs(OUT, exist_ok=True)

CSS_URL = (
    "https://fonts.googleapis.com/css2?"
    "family=Inter:wght@400;500;600;700;800&display=swap"
)

# Fetch CSS
req = urllib.request.Request(
    CSS_URL,
    headers={"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"}
)
css = urllib.request.urlopen(req).read().decode("utf-8")

# Extract all unique font URLs
urls = set(re.findall(r"url\((https://fonts\.gstatic\.com[^)]+)\)", css))
print(f"Found {len(urls)} unique font files to download")

# Download each unique font file
downloaded = {}
for url in sorted(urls):
    # Extract filename from URL path
    filename = url.split("/")[-1].split("?")[0]
    local_path = os.path.join(OUT, filename)
    if not os.path.exists(local_path):
        print(f"  Downloading {filename}...")
        urllib.request.urlretrieve(url, local_path)
        size = os.path.getsize(local_path)
        print(f"    {size/1024:.1f} KB")
    else:
        print(f"  Already exists: {filename}")
    downloaded[url] = filename

# Rewrite CSS to use local paths
for url, filename in downloaded.items():
    css = css.replace(url, f"/static/fonts/{filename}")

# Remove unicode-range for latin-only — actually keep it, it's fine
# Write the local CSS
css_path = os.path.join(OUT, "inter.css")
with open(css_path, "w") as f:
    f.write("/* Inter font — self-hosted */\n")
    f.write(css)

print(f"\nCSS written to: {css_path}")
print(f"Total local font size: {sum(os.path.getsize(os.path.join(OUT, f)) for f in os.listdir(OUT) if f.endswith('.woff2'))/1024:.1f} KB")
