# brand-assets/

Shared visual-system material for Chef's posts.

## Status

The current palette is **proposed, not approved**. It is based on the gradient
Chef supplied on 2026-08-20. Do not silently restyle an approved post asset with
it until Chef confirms the direction.

## Files

- `tokens.css` — reusable color tokens and the candidate signature background.
- `palette-preview.svg` — editable visual review sheet for the proposed palette.
- `palette-preview.png` — 1400×900 review export of the same sheet.

The preview's colors are under review; its typography and component layout are
illustrative and are not yet a design-system decision. Regenerate the PNG on
macOS with:

```bash
sips -s format png palette-preview.svg --out palette-preview.png
```

When the palette is approved, update the status here and in `BRAND.md`. Shared
templates can then be added for the agreed post formats. Post-specific exports
remain inside each post's own `assets/` directory.
