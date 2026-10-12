# Önizleme: Go tuvaline (RGB ham bayt) yazı basar.
import json, sys
from PIL import Image, ImageDraw, ImageFont
F = "/usr/share/fonts/opentype/inter/"
files = {"d300": "InterDisplay-Light.otf", "d400": "InterDisplay-Regular.otf", "d600": "InterDisplay-SemiBold.otf",
         "t400": "Inter-Regular.otf", "t600": "Inter-SemiBold.otf", "t700": "Inter-Bold.otf"}
raw, js = sys.argv[1], sys.argv[2]
j = json.load(open(js))
W, H = j["w"], j["h"]
im = Image.frombytes("RGB", (W, H), open(raw, "rb").read())
cache = {}
def font(k, size):
    key = (k, size)
    if key not in cache:
        cache[key] = ImageFont.truetype(F + files[k], max(1, size))
    return cache[key]
anchors = {0: "lm", 1: "mm", 2: "rm"}
d = ImageDraw.Draw(im)
for c in j["cmds"]:
    f = font(c["k"], c["size"])
    col = tuple(c["c"])
    if c.get("clip"):
        layer = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        ImageDraw.Draw(layer).text((c["x"], c["y"]), c["s"], font=f, fill=col + (255,), anchor=anchors[c["a"]])
        x0, x1 = max(0, c["cx0"]), min(W, c["cx1"])
        if x1 > x0:
            region = layer.crop((x0, 0, x1, H))
            im.paste(region.convert("RGB"), (x0, 0), region)
        d = ImageDraw.Draw(im)
    else:
        d.text((c["x"], c["y"]), c["s"], font=f, fill=col, anchor=anchors[c["a"]])
open(raw, "wb").write(im.tobytes())
