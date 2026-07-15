# Embedded fonts (VRE renderer)

The VRE renderer rasterizes SVG natively via [resvg](https://github.com/RazrFalcon/resvg)
(through `kanrichan/resvg-go`, WASM). It has no access to system fonts, so the
fonts it needs are embedded into the binary.

## NotoEmoji-Regular.ttf

Monochrome emoji glyphs (🛒 🚚 📦 📍 ✔ …) used by the message templates.

- Source: Google Fonts — <https://fonts.google.com/noto/specimen/Noto+Emoji>
- License: SIL Open Font License 1.1 (OFL) —
  <https://openfontlicense.org/open-font-license-official-text/>
- Copyright: The Noto Project Authors (<https://github.com/notofonts/noto-emoji>)

The text face (regular / medium / bold) is the **Go font** provided by
`golang.org/x/image/font/gofont` (BSD-3-Clause), loaded from that module rather
than vendored here.
