# Önizleme için yazı tipi genişlik tablosu (Inter, 100 px).
import json, sys
from PIL import ImageFont
F = "/usr/share/fonts/opentype/inter/"
fonts = {"d300": "InterDisplay-Light.otf", "d400": "InterDisplay-Regular.otf", "d600": "InterDisplay-SemiBold.otf",
         "t400": "Inter-Regular.otf", "t600": "Inter-SemiBold.otf", "t700": "Inter-Bold.otf"}
chars = [chr(c) for c in range(32, 127)] + list("çÇğĞıİöÖşŞüÜâÂ…↑↓–—·%")
out = {}
for k, f in fonts.items():
    fo = ImageFont.truetype(F + f, 100)
    out[k] = {ch: fo.getlength(ch) for ch in chars}
json.dump(out, open(sys.argv[1], "w"), ensure_ascii=False)
