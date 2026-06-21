# Lighthouse Performance & Accessibility Improvements

## Baseline Scores (salesmee.com)
| Category | Score | Target |
|----------|-------|--------|
| Performance | **41** | 90+ |
| Accessibility | **75** | 90+ |
| Best Practices | **96** | — |
| SEO | **100** | — |

## Core Web Vitals
| Metric | Value | Target |
|--------|-------|--------|
| FCP | 4.5s | <1.8s |
| LCP | 4.8s | <2.5s |
| TBT | 2380ms | <200ms |
| CLS | 0 | <0.1 |
| SI | 5.8s | <3.4s |
| TTI | 7.4s | <3.8s |

## Phase 1 — Quick Wins (Performance +15-20)
- [ ] **Remove Tailwind CDN** — `landing_head.html` loads `cdn.tailwindcss.com` (124KB JS, ~2s parse). Replace with `dist/styles.css` (134KB static CSS, zero parse cost)
- [ ] **Enable compression** — Brotli/gzip for all text assets in Go server
- [ ] **Cache-Control headers** — far-future expiry for `/static/*` assets
- [ ] **Optimize salesmee.svg** — unminified 124KB; SVGO can reduce ~80%+

## Phase 2 — Medium Effort (Performance +10-15)
- [ ] **Subset FontAwesome** — replace full `all.min.css` (22KB) + `fa-solid-900.woff2` (154KB) with only used icons
- [ ] **Defer non-critical JS** — landing page JS should be async/defer, not render-blocking
- [ ] **Preconnect** — add `<link rel="preconnect">` for external origins

## Phase 3 — Accessibility (75 → 95+)
- [ ] **Icon-only buttons need aria-label** — close menu button, testimonial dots
- [ ] **Color contrast** — `text-slate-400` on light backgrounds is too light; use `text-slate-600` or darker
- [ ] **Heading hierarchy** — h4 used where h2/h3 expected; restructure sequentially
- [ ] **Social links need names** — Facebook/Instagram/TikTok/YouTube/Reddit icon links need `aria-label`
