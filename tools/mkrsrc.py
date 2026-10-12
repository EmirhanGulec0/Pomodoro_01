# Windows kaynak nesnesi (.syso, COFF) üretir: simge grubu, VERSIONINFO, manifest.
# Kullanım: python3 mkrsrc.py <icon_png_klasörü> <çıktı.syso>
import io, struct, sys, xml.dom.minidom
from PIL import Image

png_dir, out_path = sys.argv[1], sys.argv[2]
SIZES = [16, 20, 24, 32, 40, 48, 64, 128, 256]
LANG = 0x041F  # Türkçe

# ---------------------------------------------------------------- simgeler
def dib_entry(img):
    w, h = img.size
    img = img.convert("RGBA")
    px = img.load()
    hdr = struct.pack("<IiiHHIIiiII", 40, w, h * 2, 1, 32, 0, 0, 0, 0, 0, 0)
    xor = bytearray()
    for y in range(h - 1, -1, -1):  # alttan üste
        for x in range(w):
            r, g, b, a = px[x, y]
            xor += bytes((b, g, r, a))
    row = ((w + 31) // 32) * 4
    andm = bytearray()
    for y in range(h - 1, -1, -1):
        bits = bytearray(row)
        for x in range(w):
            if px[x, y][3] == 0:
                bits[x // 8] |= 0x80 >> (x % 8)
        andm += bits
    return hdr + bytes(xor) + bytes(andm)

icons = []
for i, s in enumerate(SIZES):
    img = Image.open(f"{png_dir}/icon_{s}.png").convert("RGBA")
    if s >= 256:
        buf = io.BytesIO(); img.save(buf, "PNG"); data = buf.getvalue()
    else:
        data = dib_entry(img)
    icons.append((i + 1, s, data))

grp = struct.pack("<HHH", 0, 1, len(icons))
for iid, s, data in icons:
    bw = 0 if s >= 256 else s
    grp += struct.pack("<BBBBHHIH", bw, bw, 0, 0, 1, 32, len(data), iid)

# ---------------------------------------------------------------- VERSIONINFO
def pad4(b):
    return b + b"\0" * ((4 - len(b) % 4) % 4)

def block(key, value=b"", children=b"", wtype=1, vlen=0):
    data = pad4(struct.pack("<HHH", 0, vlen, wtype) + (key + "\0").encode("utf-16-le"))
    if value:
        data += value
    if children:
        data = pad4(data) + children
    data = bytearray(data)
    struct.pack_into("<H", data, 0, len(data))
    return bytes(data)

def string(key, val):
    v = (val + "\0").encode("utf-16-le")
    return pad4(block(key, v, wtype=1, vlen=len(v) // 2))

VER = (2, 0, 0, 0)
strings = [
    ("FileDescription", "Pomodoro"),
    ("FileVersion", "2.0.0"),
    ("InternalName", "WorkX"),
    ("LegalCopyright", "Kişisel kullanım"),
    ("OriginalFilename", "WorkX.exe"),
    ("ProductName", "Pomodoro"),
    ("ProductVersion", "2.0.0"),
]
table = b"".join(string(k, v) for k, v in strings)
st = pad4(block("041F04B0", children=table, wtype=1, vlen=0))
sfi = pad4(block("StringFileInfo", children=st, wtype=1, vlen=0))
var = pad4(block("Translation", struct.pack("<HH", LANG, 1200), wtype=0, vlen=4))
vfi = pad4(block("VarFileInfo", children=var, wtype=1, vlen=0))
ms, ls = (VER[0] << 16) | VER[1], (VER[2] << 16) | VER[3]
ffi = struct.pack("<13I", 0xFEEF04BD, 0x00010000, ms, ls, ms, ls, 0x3F, 0, 0x00040004, 1, 0, 0, 0)
versioninfo = block("VS_VERSION_INFO", ffi, children=sfi + vfi, wtype=0, vlen=len(ffi))

# ---------------------------------------------------------------- manifest
manifest = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="WorkX.Pomodoro" version="2.0.0.0" processorArchitecture="amd64"/>
  <description>Pomodoro</description>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
      <supportedOS Id="{1f676c76-80e1-4239-95bb-83d0f6d0da78}"/>
      <supportedOS Id="{4a2f28e3-53b9-4441-ba9c-d69d4a4a6e38}"/>
    </application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2, PerMonitor</dpiAwareness>
    </windowsSettings>
  </application>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
</assembly>
"""
xml.dom.minidom.parseString(manifest)  # bozuk XML exe'yi açılmaz yapar; burada yakala
manifest = manifest.encode("utf-8")

# ---------------------------------------------------------------- kaynak ağacı
# tür → [(ad, veri)]
tree = {
    3: [(iid, data) for iid, s, data in icons],  # RT_ICON
    14: [(1, grp)],                               # RT_GROUP_ICON
    16: [(1, versioninfo)],                       # RT_VERSION
    24: [(1, manifest)],                          # RT_MANIFEST
}
types = sorted(tree)

def dir_size(n):
    return 16 + 8 * n

# yerleşim: kök dizin, tür dizinleri, ad dizinleri (dil), veri girdileri, ham veri
off = dir_size(len(types))
type_dir_off = {}
for t in types:
    type_dir_off[t] = off
    off += dir_size(len(tree[t]))
name_dir_off = {}
for t in types:
    for name, _ in tree[t]:
        name_dir_off[(t, name)] = off
        off += dir_size(1)
entry_off = {}
for t in types:
    for name, _ in tree[t]:
        entry_off[(t, name)] = off
        off += 16
data_off = {}
for t in types:
    for name, data in tree[t]:
        off = (off + 7) & ~7
        data_off[(t, name)] = off
        off += len(data)
total = (off + 7) & ~7

sec = bytearray(total)
def put(o, b):
    sec[o:o + len(b)] = b
def dirhdr(o, n):
    put(o, struct.pack("<IIHHHH", 0, 0, 0, 0, 0, n))

relocs = []
dirhdr(0, len(types))
for i, t in enumerate(types):
    put(16 + 8 * i, struct.pack("<II", t, 0x80000000 | type_dir_off[t]))
    dirhdr(type_dir_off[t], len(tree[t]))
    for j, (name, data) in enumerate(sorted(tree[t])):
        put(type_dir_off[t] + 16 + 8 * j, struct.pack("<II", name, 0x80000000 | name_dir_off[(t, name)]))
        nd = name_dir_off[(t, name)]
        dirhdr(nd, 1)
        put(nd + 16, struct.pack("<II", LANG, entry_off[(t, name)]))
        eo = entry_off[(t, name)]
        put(eo, struct.pack("<IIII", data_off[(t, name)], len(data), 0, 0))
        relocs.append(eo)  # OffsetToData → RVA (ADDR32NB)
        put(data_off[(t, name)], data)

# ---------------------------------------------------------------- COFF
n_rel = len(relocs)
raw_ptr = 20 + 40
rel_ptr = raw_ptr + total
sym_ptr = rel_ptr + 10 * n_rel
fh = struct.pack("<HHIIIHH", 0x8664, 1, 0, sym_ptr, 1, 0, 0)
sh = struct.pack("<8sIIIIIIHHI", b".rsrc\0\0\0", 0, 0, total, raw_ptr, rel_ptr, 0, n_rel, 0, 0x40000040)
rel = b"".join(struct.pack("<IIH", r, 0, 3) for r in relocs)  # IMAGE_REL_AMD64_ADDR32NB
sym = struct.pack("<8sIhHBB", b".rsrc\0\0\0", 0, 1, 0, 3, 0)
strtab = struct.pack("<I", 4)
open(out_path, "wb").write(fh + sh + bytes(sec) + rel + sym + strtab)
print(f"{out_path}: {len(icons)} simge, sürüm bilgisi, manifest; {total} bayt kaynak, {n_rel} yer değiştirme")
