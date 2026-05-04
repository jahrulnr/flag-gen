# Owl mascot: split assets & animation plan

This document describes the **split owl sprite** shipped with the web app and a **practical plan** to animate it in the Flag Gen admin UI (for example the Environments page) without relying on full vector conversion or heavy AI image generation.

## What exists in the repo

### Source

- The original **2×4 sprite sheet** (JPEG) is kept for reference as:
  - `web/public/assets/owl/sprite-sheet-original.jpg`
- Source dimensions: **1024×682** px.

### How frames are cropped (gutter-aware)

The sheet is **not** a clean `width÷4` × `height÷2` grid: vertical gutters sit near **x ≈ 259, 490, 729**, and the horizontal seam lies in a **wide black band** around **y ≈ 271–409** (split at **y = 340** so each row shares half of that band). Naive **256×341** cells cut through shelf/owl edges, so exports use:

- **Column bounds (px):** `0–259`, `259–490`, `490–729`, `729–1024`
- **Row bounds (px):** `0–340`, `340–682`

After each cell crop, pixels with **R, G, B ≤ 11** (same rule as “strictly below 12”) become **transparent**, then each frame is **tight-cropped** to visible pixels and **centered** on a **shared canvas** so animation does not jump in size.

### Split frames (PNG, alpha)

Eight frames are exported as separate PNG files with **near-black background removed** (RGB components all below **12** treated as transparent) so the owl can sit on arbitrary page backgrounds. All frames share the same pixel size (**190×222** for the current source; re-run the script after replacing the sheet if this changes):

| File | Row | Col | Description (pose) |
|------|-----|-----|----------------------|
| `web/public/assets/owl/frame-01.png` | top | 1 | Neutral, eyes open forward |
| `web/public/assets/owl/frame-02.png` | top | 2 | Sleepy / half-lids |
| `web/public/assets/owl/frame-03.png` | top | 3 | Eyes fully closed |
| `web/public/assets/owl/frame-04.png` | top | 4 | Head tilt slightly right, gaze up-right |
| `web/public/assets/owl/frame-05.png` | bottom | 1 | Strong tilt left, gaze up-left |
| `web/public/assets/owl/frame-06.png` | bottom | 2 | Slight tilt left, forward |
| `web/public/assets/owl/frame-07.png` | bottom | 3 | Wink (one eye closed), head tilt right |
| `web/public/assets/owl/frame-08.png` | bottom | 4 | Slight tilt right, forward |

**Runtime URL (dev/prod):** the Angular app uses `baseHref: /app/`, so frames are typically requested as:

`/app/assets/owl/frame-01.png` … `/app/assets/owl/frame-08.png`

In templates, a **root-relative** path that respects `baseHref` is appropriate (for example the same path shape the app uses for other static files under `public/`).

## Why split PNGs (not “AI → SVG” for every frame)

- The art uses **gradients and soft shading**; automatic raster-to-SVG tracing tends to produce **large, fragile SVGs** or a **flat** look.
- **Eight discrete PNGs** are easy to reason about, diff-friendly, and work well with **CSS or small Angular logic**.
- This approach **does not consume ongoing AI image quota** beyond any optional future touch-ups.

## Animation strategy (recommended)

### Option A — CSS sprite strip (single HTTP request)

If you later want **one network request** again: combine the eight PNGs horizontally (or use the original sheet) and animate **`background-position`** with stepped keyframes. The split frames remain useful for **editing** and **documentation**.

### Option B — `<img>` swap / stacked crossfade (eight files)

1. Add a small presentational component (for example `OwlMascot`) used in `environments-page` header or dashboard shell.
2. **Blink loop** (readable and on-brand):  
   `01 → 02 → 03 → 02 → 01` (hold on `01` between blinks).
3. **Idle variety** (optional second timeline): occasionally insert `04`–`08` with **longer holds** on neutral frames so motion feels calm, not frantic.
4. **Timing:** roughly **2.5–4 s** total per blink cycle; idle “look around” on a **6–12 s** period if combined.
5. **Accessibility:** decorative only → `aria-hidden="true"` on the wrapper; honor **`prefers-reduced-motion`** by showing **static `frame-01.png`** only (no timer / no CSS animation).

### Option C — Angular `animation` API

Use `trigger` / `transition` / `animate` to swap `[@fade]` between images if you want **crossfade** between frames (slightly smoother perception than hard cuts). Still keep reduced-motion behavior.

## Implementation checklist (for a future PR)

1. [ ] Create `OwlMascot` (or inline in page) with fixed **width/height** preserving the **frame aspect ratio** (currently **190:222**; scale with `max-height`, e.g. 120–160 px tall).
2. [ ] Wire frame sequence with `setInterval` / `requestAnimationFrame` or pure CSS `animation` + `steps()`.
3. [ ] Add **`prefers-reduced-motion: reduce`** handling.
4. [ ] Place mascot in **Environments** header or **dashboard shell** (product decision).
5. [ ] Smoke-test **SSR**: ensure no `window` access outside `afterNextRender` / browser-only branch if using timers.

## Regenerating frames from a new sprite

1. Replace `sprite-sheet-original.jpg` under `web/public/assets/owl/` (or pass `--src` explicitly).
2. From the repo root, run:

   ```bash
   python3 scripts/split_owl_sprite.py
   ```

   If the new sheet is **not** 1024×682, update `X_SPLITS`, `Y_SPLITS`, and the size check in `scripts/split_owl_sprite.py` after re-measuring gutters (same approach as above: find wide black vertical/horizontal runs, use their midpoints as cuts).

3. Update this doc if the **normalized canvas** size changes (search for `190×222`).

## Open decisions

- **Placement:** only Environments vs global dashboard chrome.
- **Interaction:** purely decorative vs subtle reaction (e.g. faster blink on route change) — optional scope creep.
- **Dark/light theme:** if backgrounds change a lot, verify contrast under the owl’s feet/shelf; adjust CSS `filter` only if necessary (prefer keeping art unchanged).
