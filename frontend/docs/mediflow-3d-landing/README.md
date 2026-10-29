# MediFlow 3D Landing Page

Responsive medical-tech landing page concept for **Prior Authorization & Claim Completeness**.

## Run locally
Open `index.html` in a modern browser. For a local dev server, from this directory run:

```bash
python -m http.server 8000
```

Then visit `http://localhost:8000`.

## Files
- `index.html` — page structure and copy
- `styles.css` — responsive visual styling, layered scenes, floating glass panels
- `script.js` — pinned story camera easing, crossfaded scene transitions, parallax, and pointer depth
- `assets/` — generated visual assets used by the page

## Notes
This is a front-end concept, not a connected healthcare workflow. All interface content is illustrative demo content. No real patient data or authorization decisions are processed. The scroll story uses a pinned CSS 3D perspective stage with JavaScript camera easing, not a WebGL renderer.

## Art direction
The visual direction follows the requested cinematic 3D feel: ENERGY 3 / RHYTHM 3 / MOTION 3. Deep ink and muted mint/teal keep the healthcare content legible while the perspective floor gives the pinned stage a spatial reference. Scene compositions change with the story, and scroll-driven depth and crossfades make the transition itself the focal interaction. Directional arrows mark links that move to another scene; the three workflow steps follow the concept's stated sequence of gathering documentation, checking requirements, and preparing a claim. The illustrative approval card hides on narrow or short screens when it would overlap the hero actions.
