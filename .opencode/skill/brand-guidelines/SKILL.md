---
name: brand-guidelines
description: Apply Spotify Era Organizer brand identity to UI components. Use when building frontend, UI, web pages, era cards, mood visualization, styling, design system, theme, or any visual component. Provides colors, typography, mood-to-color mapping, and component styling guidelines for the Liquid Vinyl aesthetic.
---

# Spotify Era Organizer Brand Guidelines

## Aesthetic Direction: Liquid Vinyl

A **retro-futuristic, dark, immersive** visual identity inspired by vinyl records, sound waves, and the emotional journey of music listening eras.

**Key characteristics:**
- Dark mode with depth and warmth
- Iridescent mood gradients that shift based on audio features
- Organic, flowing shapes reminiscent of vinyl grooves and waveforms
- Industrial typography with a technical edge
- Subtle grain textures for analog warmth

---

## Colors

### Base Palette (Dark Mode)

```css
:root {
  /* Backgrounds */
  --bg-deep: #0a0a0b;        /* Deepest background, near-black with warmth */
  --bg-surface: #141416;     /* Card/surface background */
  --bg-elevated: #1c1c1f;    /* Elevated elements, modals, dropdowns */
  --bg-hover: #242428;       /* Hover states on surfaces */
  
  /* Text */
  --text-primary: #f5f5f7;   /* Primary text, warm white */
  --text-secondary: #8e8e93; /* Secondary/muted text */
  --text-tertiary: #48484a;  /* Disabled/hint text */
  
  /* Borders */
  --border-subtle: #2c2c2e;  /* Subtle dividers */
  --border-default: #3a3a3c; /* Default borders */
  --border-strong: #545456;  /* Emphasized borders */
}
```

### Mood Gradient Spectrum (Energy-Based)

Mood colors map **energy** (0.0-1.0) to a warm-to-cool spectrum:

| Energy | Name | Hex | HSL | Use Case |
|--------|------|-----|-----|----------|
| 0.0 | Deep Indigo | `#4a00e0` | `hsl(264, 100%, 44%)` | Ambient, chill, lo-fi |
| 0.25 | Electric Purple | `#8e2de2` | `hsl(274, 76%, 53%)` | Melancholic, introspective |
| 0.5 | Hot Magenta | `#c850c0` | `hsl(304, 53%, 55%)` | Balanced, moderate energy |
| 0.75 | Coral Fire | `#ff6b6b` | `hsl(0, 100%, 71%)` | Upbeat, warm, groovy |
| 1.0 | Electric Orange | `#ff9500` | `hsl(35, 100%, 50%)` | High-energy, intense, hype |

**Gradient stops for CSS:**
```css
--mood-gradient: linear-gradient(
  90deg,
  #4a00e0 0%,
  #8e2de2 25%,
  #c850c0 50%,
  #ff6b6b 75%,
  #ff9500 100%
);
```

### Accent Colors

```css
:root {
  --accent-primary: #ff6b6b;   /* Primary CTAs, active states */
  --accent-secondary: #8e2de2; /* Secondary highlights */
  --accent-tertiary: #c850c0;  /* Tertiary accents */
  
  --status-success: #32d74b;   /* Success, connected, created */
  --status-warning: #ff9f0a;   /* Warnings, pending */
  --status-error: #ff453a;     /* Errors, destructive actions */
  --status-info: #64d2ff;      /* Informational */
}
```

---

## Typography

### Font Stack (Google Fonts)

**Import URL:**
```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Bebas+Neue&family=JetBrains+Mono:wght@400;500;600&family=Oswald:wght@400;500;600;700&display=swap" rel="stylesheet">
```

**CSS Import (alternative):**
```css
@import url('https://fonts.googleapis.com/css2?family=Bebas+Neue&family=JetBrains+Mono:wght@400;500;600&family=Oswald:wght@400;500;600;700&display=swap');
```

### Font Assignments

| Role | Font | Fallback | Usage |
|------|------|----------|-------|
| **Display** | Bebas Neue | Oswald, Impact, sans-serif | Era titles, hero text, page headers |
| **Headlines** | Oswald | Bebas Neue, sans-serif | Section headers, card titles, labels |
| **Body/UI** | JetBrains Mono | SF Mono, Consolas, monospace | Track names, metadata, UI text |

```css
:root {
  --font-display: 'Bebas Neue', 'Oswald', 'Impact', sans-serif;
  --font-headline: 'Oswald', 'Bebas Neue', sans-serif;
  --font-body: 'JetBrains Mono', 'SF Mono', 'Consolas', monospace;
}
```

### Type Scale

```css
:root {
  --text-xs: 0.75rem;    /* 12px - fine print, timestamps */
  --text-sm: 0.875rem;   /* 14px - secondary text, labels */
  --text-base: 1rem;     /* 16px - body text */
  --text-lg: 1.25rem;    /* 20px - emphasized body */
  --text-xl: 1.5rem;     /* 24px - card titles */
  --text-2xl: 2rem;      /* 32px - section headers */
  --text-3xl: 3rem;      /* 48px - page titles */
  --text-hero: 4.5rem;   /* 72px - hero display */
}
```

### Typography Styling

**Display text (Bebas Neue):**
- Always uppercase
- Letter-spacing: `0.05em` to `0.1em`
- Line-height: `1.1`

**Headlines (Oswald):**
- Title case or uppercase
- Letter-spacing: `0.02em`
- Line-height: `1.2`

**Body text (JetBrains Mono):**
- Normal case
- Letter-spacing: `0`
- Line-height: `1.6`

---

## Mood-to-Color Algorithm

### Pseudocode

```
function moodToColor(energy, valence):
    // Energy maps to hue (cool indigo → warm orange)
    // energy 0.0 = 264° (indigo)
    // energy 1.0 = 35° (orange)
    hue = 264 - (energy * 229)
    if hue < 0:
        hue = hue + 360
    
    // Valence affects saturation (sad = desaturated, happy = vivid)
    saturation = 60 + (valence * 40)  // 60% to 100%
    
    // Valence also affects lightness (sad = darker, happy = brighter)
    lightness = 40 + (valence * 20)   // 40% to 60%
    
    return hsl(hue, saturation%, lightness%)
```

### Go Implementation Signature

```go
// MoodColor returns an HSL color string based on audio features
func MoodColor(energy, valence float64) string {
    hue := 264 - (energy * 229)
    if hue < 0 {
        hue += 360
    }
    saturation := 60 + (valence * 40)
    lightness := 40 + (valence * 20)
    return fmt.Sprintf("hsl(%.0f, %.0f%%, %.0f%%)", hue, saturation, lightness)
}
```

### CSS Custom Property Pattern

For dynamic mood colors in templates:
```html
<div class="era-card" style="--era-color: {{moodColor .Energy .Valence}}">
```

```css
.era-card {
  border-color: var(--era-color);
  box-shadow: 0 0 30px color-mix(in srgb, var(--era-color) 30%, transparent);
}
```

---

## Component Guidelines

### Era Cards

The primary UI element displaying a music listening era.

**Structure:**
```html
<article class="era-card" style="--era-color: hsl(...)">
  <div class="era-card__header">
    <h2 class="era-card__title">Late Night Vibes</h2>
    <span class="era-card__dates">Jan 15 - Feb 28, 2024</span>
  </div>
  <div class="era-card__mood-bar"></div>
  <div class="era-card__tracks">...</div>
  <div class="era-card__actions">...</div>
</article>
```

**Styling principles:**
- Background: `--bg-surface` with subtle glass-morphism (`backdrop-filter: blur(10px)`)
- Border: 1px solid with `--era-color` at 30% opacity
- Border-radius: `1.5rem` (organic, vinyl-inspired)
- Box-shadow: Soft glow using `--era-color`
- Hover: Elevate with increased glow intensity

**CSS:**
```css
.era-card {
  background: linear-gradient(
    135deg,
    rgba(20, 20, 22, 0.9),
    rgba(28, 28, 31, 0.8)
  );
  backdrop-filter: blur(10px);
  border: 1px solid color-mix(in srgb, var(--era-color) 30%, transparent);
  border-radius: 1.5rem;
  box-shadow: 
    0 4px 24px rgba(0, 0, 0, 0.4),
    0 0 40px color-mix(in srgb, var(--era-color) 15%, transparent);
  transition: box-shadow 0.3s ease, transform 0.3s ease;
}

.era-card:hover {
  box-shadow: 
    0 8px 32px rgba(0, 0, 0, 0.5),
    0 0 60px color-mix(in srgb, var(--era-color) 25%, transparent);
  transform: translateY(-2px);
}

.era-card__title {
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-primary);
}

.era-card__dates {
  font-family: var(--font-body);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.era-card__mood-bar {
  height: 4px;
  background: var(--era-color);
  border-radius: 2px;
  box-shadow: 0 0 10px var(--era-color);
}
```

### Buttons

**Primary Button:**
```css
.btn-primary {
  font-family: var(--font-headline);
  font-size: var(--text-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-tertiary));
  color: var(--bg-deep);
  border: none;
  border-radius: 0.5rem;
  padding: 0.75rem 1.5rem;
  
  transition: all 0.2s ease;
  cursor: pointer;
}

.btn-primary:hover {
  box-shadow: 0 0 20px color-mix(in srgb, var(--accent-primary) 50%, transparent);
  transform: translateY(-1px);
}
```

**Secondary Button (Ghost):**
```css
.btn-secondary {
  font-family: var(--font-headline);
  font-size: var(--text-sm);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  
  background: transparent;
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  border-radius: 0.5rem;
  padding: 0.75rem 1.5rem;
  
  transition: all 0.2s ease;
  cursor: pointer;
}

.btn-secondary:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
```

### Track Lists

```css
.track-list {
  font-family: var(--font-body);
  font-size: var(--text-sm);
}

.track-item {
  display: grid;
  grid-template-columns: 2rem 1fr auto;
  gap: 1rem;
  align-items: center;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  transition: background 0.15s ease;
}

.track-item:nth-child(odd) {
  background: rgba(255, 255, 255, 0.02);
}

.track-item:hover {
  background: var(--bg-hover);
}

.track-item__number {
  color: var(--text-tertiary);
  font-size: var(--text-xs);
}

.track-item__name {
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.track-item__artist {
  color: var(--text-secondary);
  font-size: var(--text-xs);
}

.track-item__energy {
  width: 3rem;
  height: 3px;
  background: var(--bg-elevated);
  border-radius: 2px;
  overflow: hidden;
}

.track-item__energy-fill {
  height: 100%;
  background: var(--mood-gradient);
  border-radius: 2px;
}
```

### Loading States

**Skeleton loader with vinyl spin:**
```css
.skeleton {
  background: linear-gradient(
    90deg,
    var(--bg-surface) 0%,
    var(--bg-elevated) 50%,
    var(--bg-surface) 100%
  );
  background-size: 200% 100%;
  animation: skeleton-shimmer 1.5s infinite;
  border-radius: 0.5rem;
}

@keyframes skeleton-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.spinner {
  width: 2rem;
  height: 2rem;
  border: 2px solid var(--border-subtle);
  border-top-color: var(--accent-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
```

### Form Inputs

```css
.input {
  font-family: var(--font-body);
  font-size: var(--text-base);
  
  background: var(--bg-surface);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  border-radius: 0.5rem;
  padding: 0.75rem 1rem;
  
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.input:focus {
  outline: none;
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent-primary) 20%, transparent);
}

.input::placeholder {
  color: var(--text-tertiary);
}
```

---

## Visual Motifs

### Vinyl Grooves

Use concentric arcs or circles as decorative elements:
```css
.vinyl-texture {
  background-image: repeating-radial-gradient(
    circle at center,
    transparent 0px,
    transparent 2px,
    rgba(255, 255, 255, 0.02) 2px,
    rgba(255, 255, 255, 0.02) 4px
  );
}
```

### Sound Wave Divider

```css
.wave-divider {
  height: 40px;
  background: url("data:image/svg+xml,...") repeat-x center;
  opacity: 0.3;
}
```

### Grain Overlay

Add analog warmth with a subtle noise texture:
```css
.grain-overlay::after {
  content: '';
  position: fixed;
  inset: 0;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)'/%3E%3C/svg%3E");
  opacity: 0.03;
  pointer-events: none;
  z-index: 9999;
}
```

### Iridescent Glow

For special emphasis (hero sections, featured eras):
```css
.iridescent {
  background: linear-gradient(
    135deg,
    var(--accent-secondary),
    var(--accent-tertiary),
    var(--accent-primary)
  );
  background-size: 200% 200%;
  animation: iridescent-shift 8s ease infinite;
}

@keyframes iridescent-shift {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}
```

---

## CSS Variables Reference (Complete)

Copy this entire block into your base CSS file:

```css
:root {
  /* === COLORS === */
  
  /* Backgrounds */
  --bg-deep: #0a0a0b;
  --bg-surface: #141416;
  --bg-elevated: #1c1c1f;
  --bg-hover: #242428;
  
  /* Text */
  --text-primary: #f5f5f7;
  --text-secondary: #8e8e93;
  --text-tertiary: #48484a;
  
  /* Borders */
  --border-subtle: #2c2c2e;
  --border-default: #3a3a3c;
  --border-strong: #545456;
  
  /* Accents */
  --accent-primary: #ff6b6b;
  --accent-secondary: #8e2de2;
  --accent-tertiary: #c850c0;
  
  /* Status */
  --status-success: #32d74b;
  --status-warning: #ff9f0a;
  --status-error: #ff453a;
  --status-info: #64d2ff;
  
  /* Mood Spectrum */
  --mood-calm: #4a00e0;
  --mood-low: #8e2de2;
  --mood-mid: #c850c0;
  --mood-high: #ff6b6b;
  --mood-intense: #ff9500;
  --mood-gradient: linear-gradient(90deg, #4a00e0, #8e2de2, #c850c0, #ff6b6b, #ff9500);
  
  /* === TYPOGRAPHY === */
  
  /* Fonts */
  --font-display: 'Bebas Neue', 'Oswald', 'Impact', sans-serif;
  --font-headline: 'Oswald', 'Bebas Neue', sans-serif;
  --font-body: 'JetBrains Mono', 'SF Mono', 'Consolas', monospace;
  
  /* Scale */
  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.25rem;
  --text-xl: 1.5rem;
  --text-2xl: 2rem;
  --text-3xl: 3rem;
  --text-hero: 4.5rem;
  
  /* === SPACING === */
  --space-xs: 0.25rem;
  --space-sm: 0.5rem;
  --space-md: 1rem;
  --space-lg: 1.5rem;
  --space-xl: 2rem;
  --space-2xl: 3rem;
  --space-3xl: 4rem;
  
  /* === EFFECTS === */
  --radius-sm: 0.25rem;
  --radius-md: 0.5rem;
  --radius-lg: 1rem;
  --radius-xl: 1.5rem;
  --radius-full: 9999px;
  
  --shadow-sm: 0 2px 8px rgba(0, 0, 0, 0.3);
  --shadow-md: 0 4px 16px rgba(0, 0, 0, 0.4);
  --shadow-lg: 0 8px 32px rgba(0, 0, 0, 0.5);
  
  --blur-sm: 4px;
  --blur-md: 10px;
  --blur-lg: 20px;
  
  /* === TRANSITIONS === */
  --transition-fast: 0.15s ease;
  --transition-normal: 0.2s ease;
  --transition-slow: 0.3s ease;
}

/* Base styles */
html {
  background: var(--bg-deep);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 16px;
  line-height: 1.6;
}

body {
  min-height: 100vh;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Selection */
::selection {
  background: var(--accent-secondary);
  color: var(--text-primary);
}
```
