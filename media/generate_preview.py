"""Regenerate the original repository diagram: python -m pip install pillow."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent
FONT_DIR = Path('/System/Library/Fonts/Supplemental')
# On Linux, point FONT_DIR at a directory containing Arial or change these names.
def font(size, bold=False):
    return ImageFont.truetype(str(FONT_DIR / ('Arial Bold.ttf' if bold else 'Arial.ttf')), size)

im = Image.new('RGB', (1280, 640), '#0b1324')
d = ImageDraw.Draw(im)
d.rounded_rectangle((54, 48, 276, 85), 15, fill='#163c40')
d.text((71, 54), 'OPEN SOURCE  /  MIT', font=font(19, True), fill='#79e0cb')
d.text((54, 123), 'Universal', font=font(69, True), fill='#f2f6ff')
d.text((54, 198), 'Bluetooth SDK', font=font(69, True), fill='#f2f6ff')
d.text((57, 300), 'One daemon. Your tools.', font=font(31), fill='#b7c6dd')
d.text((57, 345), 'A foundation for nearby apps.', font=font(29), fill='#b7c6dd')
for y,label in ((429,'Go CLI + MCP'),(467,'Linux RFCOMM + hardware-free demo')):
    d.ellipse((58,y+8,66,y+16), fill='#79e0cb')
    d.text((81,y),label,font=font(23),fill='#e6efff')
for rect,title,subtitle in [((770,130,1225,227),'CLI  /  AI  /  MCP','local tools'),((770,282,1225,384),'ubtd','one local API'),((770,439,1225,522),'stub  |  Linux RFCOMM','current drivers')]:
    d.rounded_rectangle(rect,18,fill='#14223b',outline='#354965',width=2)
    cx=(rect[0]+rect[2])//2
    d.text((cx,rect[1]+19),title,font=font(29,True),fill='#ecf4ff',anchor='mt')
    d.text((cx,rect[1]+56),subtitle,font=font(18),fill='#91a7c5',anchor='mt')
for a,b in ((227,282),(384,439)):
    d.line((998,a,998,b-9), fill='#79e0cb',width=3)
    d.polygon([(991,b-14),(1005,b-14),(998,b-5)],fill='#79e0cb')
d.line((55,554,1225,554),fill='#354965',width=1)
d.text((56,580),'sraodev/universal-bluetooth-sdk',font=font(23),fill='#93accb')
d.text((1224,580),'EXPERIMENTAL',font=font(20,True),fill='#fac875',anchor='rt')
im.save(ROOT/'social-preview.png',optimize=True)
print('Saved 1280x640 RGB PNG')
