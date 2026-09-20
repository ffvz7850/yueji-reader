from PIL import Image, ImageDraw, ImageFont, ImageFilter
import os

def gen(size, path):
    # 圆角渐变方块
    img = Image.new("RGBA", (size, size), (0,0,0,0))
    d = ImageDraw.Draw(img)
    # 渐变：上 #6366f1 -> 下 #8b5cf6
    top = (99, 102, 241); bottom = (139, 92, 246)
    for y in range(size):
        t = y / size
        c = tuple(int(top[i]*(1-t) + bottom[i]*t) for i in range(3))
        d.line([(0,y),(size,y)], fill=c+(255,))
    # 圆角 mask
    mask = Image.new("L", (size, size), 0)
    md = ImageDraw.Draw(mask)
    r = int(size * 0.22)
    md.rounded_rectangle([0,0,size-1,size-1], radius=r, fill=255)
    img.putalpha(mask)
    # 白色"悦"字
    font_path = r"C:\Windows\Fonts\msyhbd.ttc"
    if not os.path.exists(font_path):
        font_path = r"C:\Windows\Fonts\msyh.ttc"
    fs = int(size * 0.60)
    try:
        font = ImageFont.truetype(font_path, fs)
    except Exception:
        font = ImageFont.load_default()
    text = "悦"
    bbox = d.textbbox((0,0), text, font=font)
    tw, th = bbox[2]-bbox[0], bbox[3]-bbox[1]
    tx = (size - tw)//2 - bbox[0]
    ty = (size - th)//2 - bbox[1] + int(size*0.02)
    d.text((tx, ty), text, font=font, fill=(255,255,255,255))
    img.save(path, "PNG")
    print(path, img.size)

gen(192, "icon-192.png")
gen(512, "icon-512.png")
