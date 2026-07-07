#!/usr/bin/env python3
"""Generate a self-hosted star-history SVG (no third-party service).
ponytail: stdlib-only, duplicated per-repo; extract to a shared action if this
grows beyond a handful of repos.
"""
import sys, os, json, urllib.request
from datetime import datetime

repo, out = sys.argv[1], sys.argv[2]
tok = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN", "")


def api(url):
    h = {"Accept": "application/vnd.github.star+json", "User-Agent": "star-history-gen"}
    if tok:
        h["Authorization"] = f"Bearer {tok}"
    return json.load(urllib.request.urlopen(urllib.request.Request(url, headers=h)))


stars, page = [], 1
while True:
    batch = api(f"https://api.github.com/repos/{repo}/stargazers?per_page=100&page={page}")
    stars += [s["starred_at"] for s in batch]
    if len(batch) < 100:
        break
    page += 1
stars.sort()
n = len(stars)

W, H, P = 800, 250, 40
epochs = [datetime.strptime(s, "%Y-%m-%dT%H:%M:%SZ").timestamp() for s in stars]
t0, t1 = (epochs[0], epochs[-1]) if n > 1 and epochs[-1] > epochs[0] else (0, 1)


def X(t):
    return P + (W - 2 * P) * ((t - t0) / (t1 - t0) if t1 > t0 else 0)


def Y(i):
    return (H - P) - (H - 2 * P) * ((i + 1) / n if n else 0)


pts = [(X(epochs[i]), Y(i)) for i in range(n)] or [(P, H - P)]
line = " ".join(f"{x:.1f},{y:.1f}" for x, y in pts)
area = f"{pts[0][0]:.1f},{H - P} {line} {pts[-1][0]:.1f},{H - P}"
d0 = stars[0][:10] if n else ""
d1 = stars[-1][:10] if n else ""

svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}" width="100%" font-family="monospace">
<defs>
 <linearGradient id="l" x1="0" x2="1"><stop offset="0" stop-color="#a855f7"/><stop offset="1" stop-color="#e91e8c"/></linearGradient>
 <linearGradient id="f" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#a855f7" stop-opacity=".35"/><stop offset="1" stop-color="#a855f7" stop-opacity="0"/></linearGradient>
</defs>
<rect width="{W}" height="{H}" rx="10" fill="#0d1117"/>
<polygon points="{area}" fill="url(#f)"/>
<polyline points="{line}" fill="none" stroke="url(#l)" stroke-width="2.5" stroke-linejoin="round"/>
<text x="{P}" y="30" fill="#a855f7" font-size="16">&#9733; {n} stars</text>
<text x="{P}" y="{H - 14}" fill="#6b7280" font-size="11">{d0}</text>
<text x="{W - P}" y="{H - 14}" fill="#6b7280" font-size="11" text-anchor="end">{d1}</text>
<text x="{W - P}" y="30" fill="#6b7280" font-size="12" text-anchor="end">Star History</text>
</svg>'''

with open(out, "w") as f:
    f.write(svg)
print(f"{repo}: {n} stars -> {out}")
