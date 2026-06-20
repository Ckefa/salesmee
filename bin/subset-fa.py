#!/usr/bin/env python3
"""Subset Font Awesome to only the icons used by SalesMee."""
import os, sys, json, subprocess, tempfile, shutil, re
import yaml

PROJECT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FA_DIR = os.path.join(PROJECT, "node_modules", "@fortawesome", "fontawesome-free")
OUT_DIR = os.path.join(PROJECT, "web", "static", "fonts")
CSS_OUT = os.path.join(PROJECT, "web", "static", "css", "fontawesome-subset.css")

# All unique icon names used in the project (without fa- prefix)
# Solid icons (fas)
SOLID_ICONS = [
    "adjust","arrow-left","arrow-right","arrows-rotate","arrow-up","at","bag-shopping","ban","bars",
    "bell","bolt","book","book-open","box","box-open","boxes","building","bullseye","calendar",
    "calendar-alt","calendar-check","calendar-day","calendar-days","calendar-plus","calendar-week",
    "camera","car","cart-plus","chart-bar","chart-line","chart-pie","chart-simple","check",
    "check-circle","check-double","chevron-down","chevron-left","chevron-right","child","circle",
    "clipboard-list","clock","code","comment","comment-dots","comments","compass","concierge-bell",
    "cookie-bite","copy","credit-card","crown","cube","cut","dollar-sign","download","dumbbell",
    "edit","ellipsis","ellipsis-h","ellipsis-v","envelope","eraser","exclamation-circle",
    "exclamation-triangle","external-link-alt","eye","eye-slash","file","file-alt","file-invoice",
    "file-invoice-dollar","fire","flag-checkered","flask","gavel","gem","globe","graduation-cap",
    "hand-holding-dollar","hand-pointer","handshake","heart-pulse","history","home","hourglass-half",
    "id-card","image","images","inbox","infinity","info-circle","key","keyboard","layer-group",
    "lightbulb","link","location-dot","lock","lock-open","map-marker-alt","message","microchip",
    "microphone","minus","mobile-screen","money-bill-wave","moon","palette","paper-plane","paperclip",
    "pause","paw","pen","phone","play","plus","plus-circle","print","qrcode","receipt","redo",
    "reply","rocket","rotate","save","scissors","search","share-alt","share-nodes","shield",
    "shield-alt","shield-halved","shopping-bag","shopping-cart","sign-in-alt","sign-out-alt","spa",
    "sparkles","spinner","spray-can-sparkles","star","star-half-alt","steps","sticky-note","store",
    "store-alt","store-slash","sun","sync","sync-alt","tag","tasks","thumbs-up","times",
    "times-circle","tooth","tractor","trash","trash-alt","trash-can","user","user-check",
    "user-gear","user-plus","user-tie","users","users-cog","utensils","wand-magic-sparkles","water",
    "wrench",
]

# Regular icons (far)
REGULAR_ICONS = [
    "clock","smile","star","sticky-note",
]

# Brand icons (fab)
BRAND_ICONS = [
    "facebook","facebook-f","google","instagram","reddit","tiktok","whatsapp","x-twitter","youtube",
]

def load_icons_yml():
    path = os.path.join(FA_DIR, "metadata", "icons.yml")
    with open(path) as f:
        return yaml.safe_load(f)

def get_unicodes(icons, style, data):
    unicodes = []
    for name in icons:
        if name not in data:
            print(f"  WARNING: icon '{name}' not found in FA metadata")
            continue
        info = data[name]
        if style not in info.get("styles", []):
            print(f"  NOTE: icon '{name}' not available in {style} style, checking...")
            # Try to use anyway - it might have the codepoint
        if "unicode" in info:
            unicodes.append(info["unicode"])
        else:
            print(f"  WARNING: icon '{name}' has no unicode value")
    return unicodes

def subset_font(input_path, output_path, unicodes, font_name):
    if not os.path.exists(input_path):
        print(f"  ERROR: Input font not found: {input_path}")
        return False
    if not unicodes:
        print(f"  No unicodes to subset for {font_name}")
        return False
    
    hex_str = ",".join(f"U+{u.upper()}" for u in unicodes)
    subprocess.run([
        "pyftsubset", input_path,
        f"--unicodes={hex_str}",
        f"--output-file={output_path}",
        "--flavor=woff2",
        "--layout-features=",
    ], check=True)
    return True

def build_css():
    lines = []
    lines.append("/*! Font Awesome Free Subset — SalesMee */")
    lines.append("@font-face {")
    lines.append("  font-family: 'Font Awesome 6 Free Subset';")
    lines.append("  font-style: normal;")
    lines.append("  font-weight: 900;")
    lines.append("  font-display: block;")
    lines.append("  src: url('/static/fonts/fa-solid-subset.woff2') format('woff2');")
    lines.append("}")
    lines.append("")
    lines.append("@font-face {")
    lines.append("  font-family: 'Font Awesome 6 Free Subset';")
    lines.append("  font-style: normal;")
    lines.append("  font-weight: 400;")
    lines.append("  font-display: block;")
    lines.append("  src: url('/static/fonts/fa-regular-subset.woff2') format('woff2');")
    lines.append("}")
    lines.append("")
    lines.append("@font-face {")
    lines.append("  font-family: 'Font Awesome 6 Brands Subset';")
    lines.append("  font-style: normal;")
    lines.append("  font-weight: 400;")
    lines.append("  font-display: block;")
    lines.append("  src: url('/static/fonts/fa-brands-subset.woff2') format('woff2');")
    lines.append("}")
    lines.append("")
    lines.append(".fas, .fa-solid { font-family: 'Font Awesome 6 Free Subset'; font-weight: 900; }")
    lines.append(".far, .fa-regular { font-family: 'Font Awesome 6 Free Subset'; font-weight: 400; }")
    lines.append(".fab, .fa-brands { font-family: 'Font Awesome 6 Brands Subset'; font-weight: 400; }")
    lines.append("")
    lines.append(".fa, .fas, .far, .fab, .fa-solid, .fa-regular, .fa-brands {")
    lines.append("  -moz-osx-font-smoothing: grayscale;")
    lines.append("  -webkit-font-smoothing: antialiased;")
    lines.append("  display: inline-block;")
    lines.append("  font-style: normal;")
    lines.append("  font-variant: normal;")
    lines.append("  line-height: 1;")
    lines.append("  text-rendering: auto;")
    lines.append("}")
    lines.append("")
    # FA v5 compatibility — map legacy names used in the codebase
    # Already handled by the font subset since these are aliases
    
    with open(CSS_OUT, "w") as f:
        f.write("\n".join(lines) + "\n")
    print(f"  CSS written to: {CSS_OUT}")

def main():
    os.makedirs(OUT_DIR, exist_ok=True)
    
    data = load_icons_yml()
    print(f"Loaded {len(data)} icons from metadata")
    
    # Solid subset
    solid_unicodes = get_unicodes(SOLID_ICONS, "solid", data)
    print(f"Solid: {len(solid_unicodes)} unicodes")
    
    # Regular subset
    reg_unicodes = get_unicodes(REGULAR_ICONS, "regular", data)
    print(f"Regular: {len(reg_unicodes)} unicodes")
    
    # Brands subset
    brand_unicodes = get_unicodes(BRAND_ICONS, "brands", data)
    print(f"Brands: {len(brand_unicodes)} unicodes")
    
    # Subset fonts
    print("\nSubsetting solid font...")
    solid_in = os.path.join(FA_DIR, "webfonts", "fa-solid-900.woff2")
    solid_out = os.path.join(OUT_DIR, "fa-solid-subset.woff2")
    subset_font(solid_in, solid_out, solid_unicodes, "solid")
    
    print("Subsetting regular font...")
    reg_in = os.path.join(FA_DIR, "webfonts", "fa-regular-400.woff2")
    reg_out = os.path.join(OUT_DIR, "fa-regular-subset.woff2")
    subset_font(reg_in, reg_out, reg_unicodes, "regular")
    
    print("Subsetting brands font...")
    brand_in = os.path.join(FA_DIR, "webfonts", "fa-brands-400.woff2")
    brand_out = os.path.join(OUT_DIR, "fa-brands-subset.woff2")
    subset_font(brand_in, brand_out, brand_unicodes, "brands")
    
    # Build CSS
    print("\nBuilding CSS...")
    build_css()
    
    # Report sizes
    print("\n--- Font Sizes ---")
    for name in ["fa-solid-subset.woff2", "fa-regular-subset.woff2", "fa-brands-subset.woff2"]:
        path = os.path.join(OUT_DIR, name)
        if os.path.exists(path):
            size = os.path.getsize(path)
            print(f"  {name}: {size/1024:.1f} KB")
    
    orig_size = os.path.getsize(os.path.join(FA_DIR, "webfonts", "fa-solid-900.woff2"))
    print(f"\n  Original solid: {orig_size/1024:.1f} KB")
    print(f"  Done! Replace CDN link with /static/css/fontawesome-subset.css")

if __name__ == "__main__":
    main()
