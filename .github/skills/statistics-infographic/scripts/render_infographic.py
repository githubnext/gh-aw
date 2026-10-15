"""Portable, deterministic eight-panel infographic renderer."""

from __future__ import annotations

import argparse
from datetime import timezone
from fractions import Fraction
from functools import lru_cache
import math
import os
from pathlib import Path
import tempfile

from PIL import Image, ImageDraw, ImageFont

from statistics_data import Activity, ChangeBin, DataError, Statistics, decimal, load_sources, percent


WIDTH, HEIGHT = 2560, 5040
BG, PANEL, WHITE, INK = "#0D1117", "#161B22", "#FFFFFF", "#C9D1D9"
GREEN, BLUE, RED, GRAY = "#3FB950", "#58A6FF", "#F85149", "#484F58"
Bbox = tuple[float, float, float, float]


class LayoutError(ValueError):
    """Text or a chart cannot fit without losing readability or accuracy."""


def discover_fonts(regular: Path | None, bold: Path | None) -> tuple[Path, Path]:
    if regular is not None or bold is not None:
        if regular is None or bold is None:
            raise LayoutError("Provide both --font-regular and --font-bold.")
        candidates = [(regular, bold)]
    else:
        windows = Path(os.environ.get("WINDIR", "C:/Windows")) / "Fonts"
        candidates = [
            (Path("/System/Library/Fonts/Supplemental/Arial.ttf"),
             Path("/System/Library/Fonts/Supplemental/Arial Bold.ttf")),
            (windows / "arial.ttf", windows / "arialbd.ttf"),
            (Path("/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf"),
             Path("/usr/share/fonts/truetype/liberation2/LiberationSans-Bold.ttf")),
            (Path("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"),
             Path("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf")),
        ]
    for pair in candidates:
        if all(path.is_file() for path in pair):
            return pair
    raise LayoutError("No regular/bold TrueType font pair found. Supply --font-regular and --font-bold.")


class Canvas:
    def __init__(self, fonts: tuple[Path, Path], height: int = HEIGHT) -> None:
        self.fonts = fonts
        self.image = Image.new("RGB", (WIDTH, height), BG)
        self.draw = ImageDraw.Draw(self.image)
        self.labels: list[tuple[Bbox, str]] = []
        self.clip = (0, 0, WIDTH, height)

    @lru_cache(maxsize=32)
    def face(self, size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
        return ImageFont.truetype(str(self.fonts[int(bold)]), size)

    def text(self, value: str, x: float, y: float, size: int = 36,
             color: str = INK, bold: bool = False, width: float | None = None,
             max_lines: int = 1) -> None:
        face = self.face(size, bold)
        available = self.clip[2] - x - 16 if width is None else width
        lines: list[str] = []
        for paragraph in value.split("\n"):
            line = ""
            for word in paragraph.split():
                candidate = f"{line} {word}".strip()
                if self.draw.textlength(candidate, font=face) > available:
                    if not line:
                        raise LayoutError(f"Word does not fit: {word!r}")
                    lines.append(line)
                    line = word
                else:
                    line = candidate
            lines.append(line)
        if len(lines) > max_lines:
            raise LayoutError(f"Text needs more than {max_lines} line(s): {value!r}")
        for i, line in enumerate(lines):
            position = (x, y + i * size * 1.25)
            box = self.draw.textbbox(position, line, font=face)
            if not (self.clip[0] <= box[0] <= box[2] <= self.clip[2]
                    and self.clip[1] <= box[1] <= box[3] <= self.clip[3]):
                raise LayoutError(f"Text leaves its panel: {line!r}")
            if line:
                self.labels.append((box, line))
            self.draw.text(position, line, fill=color, font=face)

    def center(self, value: str, x: float, y: float, size: int = 32,
               bold: bool = False, width: float | None = None) -> None:
        length = self.draw.textlength(value, font=self.face(size, bold))
        if width is not None and length > width:
            raise LayoutError(f"Chart label does not fit its slot: {value!r}")
        self.text(value, x - length / 2, y, size, WHITE if bold else INK, bold,
                  width=length + 1)

    def card(self, index: int, title: str) -> tuple[int, int, int, int]:
        x = 72 + (index % 2) * 1228 if index < 6 else 72
        y = 330 + (index // 2 if index < 6 else index - 3) * 920
        width = 2416 if index >= 6 else 1188
        self.draw.rounded_rectangle((x, y, x + width, y + 860), radius=28,
                                    fill=PANEL, outline="#30363D", width=2)
        self.clip = (x + 16, y + 16, x + width - 16, y + 844)
        self.text(f"{index + 1:02d}", x + 40, y + 33, 34, BLUE, True)
        self.text(title, x + 120, y + 30, 48, WHITE, True)
        return x + 40, y + 125, width - 80, y + 743

    def stacked(self, values: list[int], colors: list[str], x: int, y: int,
                width: int) -> None:
        total = sum(values)
        if total == 0:
            self.text("No completed / eligible records", x, y, 32)
            return
        left = float(x)
        for value, color in zip(values, colors):
            right = left + width * value / total
            if value:
                self.draw.rectangle((left, y, right, y + 38), fill=color)
            left = right

    def verify(self) -> None:
        for i, (a, label_a) in enumerate(self.labels):
            for b, label_b in self.labels[i + 1:]:
                if max(a[0], b[0]) < min(a[2], b[2]) and max(a[1], b[1]) < min(a[3], b[3]):
                    raise LayoutError(f"Text overlap: {label_a!r} / {label_b!r}")


def draw_change_histogram(c: Canvas, bins: tuple[ChangeBin, ...], population: int,
                          x: int, y: int, width: int) -> None:
    c.text(f"{population:,} PRs per series | Green: added | Red: removed | Translucent overlap",
           x, y, 38, WHITE)
    if not bins:
        c.text("No PR diff records", x, y + 220, 38)
        return
    left, right, baseline, height = x + 120, x + width - 20, y + 440, 330
    maximum = max(max(b.added, b.deleted) for b in bins)
    ticks = sorted({0, (maximum + 1) // 2, maximum})
    c.text("Number of PRs", x, y + 58, 32)
    for tick in ticks:
        gy = baseline - tick * height / maximum
        c.draw.line((left, gy, right, gy), fill=GRAY, width=2)
        c.text(f"{tick:,}", x, gy - 20, 32, width=100)
    slot = (right - left) / len(bins)
    wrap_ranges = any(c.draw.textlength(f"{b.lower:,}-{b.upper - 1:,}", font=c.face(30)) > slot - 8 for b in bins)
    bars = ImageDraw.Draw(c.image, "RGBA")
    for i, bucket in enumerate(bins):
        middle = left + (i + .5) * slot
        for value, color in ((bucket.added, (63, 185, 80)), (bucket.deleted, (248, 81, 73))):
            if value:
                bars.rectangle((middle - slot * .42, baseline - value * height / maximum,
                                middle + slot * .42, baseline),
                               fill=(*color, 120), outline=(*color, 255), width=3)
        if wrap_ranges:
            c.center(f"{bucket.lower:,}", middle, baseline + 20, size=30, width=slot - 8)
            c.center(f"- {bucket.upper - 1:,}", middle, baseline + 56, size=30, width=slot - 8)
        else:
            c.center(f"{bucket.lower:,}-{bucket.upper - 1:,}", middle, baseline + 20,
                     size=30, width=slot - 8)
    c.center("Lines per PR (inclusive ranges)", (left + right) / 2, baseline + (106 if wrap_ranges else 90), size=34)


def draw_activity(c: Canvas, activity: tuple[Activity, ...], population: int, x: int, y: int, width: int) -> None:
    c.text(f"Contribution authors | Same denominator: {population:,} PRs | Blue: human | Green: agent/bot", x, y, 38, WHITE)
    for metric_index, (label, values) in enumerate((
        ("Conversation comments per PR", [g.comments for g in activity[:2]]),
        ("Commits per PR", [g.commits for g in activity[:2]]),
    )):
        top = y + 90 + metric_index * 265
        c.text(label, x, top, 42, WHITE, True)
        means = [Fraction(value, population) if population else Fraction(0) for value in values]
        maximum = max(max(means), Fraction(1))
        left, right = x + 650, x + width - 210
        for row, (group, total, mean, color) in enumerate(zip(activity[:2], values, means, (BLUE, GREEN))):
            row_y = top + 80 + row * 75
            c.text(f"{'Human' if row == 0 else 'Agent/bot'} ({total:,} total)", x, row_y, 36)
            if mean:
                c.draw.rectangle((left, row_y + 5, left + float(mean / maximum) * (right - left), row_y + 45),
                                 fill=color)
            c.text(group.average(total, population), right + 30, row_y, 40, WHITE, True, width=170)


def render(s: Statistics, fonts: tuple[Path, Path], title: str = "Development, in numbers") -> Canvas:
    c = Canvas(fonts)
    t = c.text
    all_prs = s.all_prs
    starred = "*" if s.components["cca_no_issue_assumed_human_prs"] else ""
    t(s.repository.upper(), 72, 52, 38, BLUE, True)
    t(title, 72, 110, 94, WHITE, True)
    t(f"{all_prs.total:,} pull requests | Snapshot {s.snapshot.isoformat()}", 72, 236, 42)

    x, y, w, foot = c.card(0, "AW vs human origins")
    for offset, outcome, label, color in [
        (0, s.outcomes[0], "AW-started", GREEN),
        (560, s.outcomes[1], f"Human-started{starred}", BLUE),
    ]:
        t(percent(outcome.total, all_prs.total), x + offset, y, 112, color, True)
        t(label, x + offset, y + 145, 42, WHITE, True)
        t(f"{outcome.total:,} PRs", x + offset, y + 211, 40)
    c.stacked([o.total for o in s.outcomes], [GREEN, BLUE, GRAY], x, y + 302, w)
    t(f"Other / unattributed: {s.outcomes[2].total:,} PRs ({percent(s.outcomes[2].total, all_prs.total)})", x, y + 369, 40, WHITE)
    p = s.components
    t(f"AW = {p['direct_aw_prs']:,} direct + {p['cca_aw_issue_prs']:,} AW-issue CCA", x, y + 444)
    t(f"Human = {p['cca_human_issue_prs']:,} human-issue CCA + {p['cca_no_issue_assumed_human_prs']:,} assumed", x, y + 501)
    caveat = "Origins are inferred, not verified session starts."
    if starred:
        caveat = "* No identified source issue = assumed human-started CCA. " + caveat
    t(caveat, x, foot, 34, width=w, max_lines=2)

    x, y, w, foot = c.card(1, "Merged vs closed")
    t(all_prs.merge_rate, x, y, 112, GREEN, True)
    t("Merge rate", x, y + 145, 42, WHITE, True)
    t(all_prs.ratio, x + 560, y + 15, 98, BLUE, True)
    t("Merged : closed", x + 560, y + 145, 42, WHITE, True)
    t(f"{all_prs.merged:,} merged / {all_prs.closed:,} closed-unmerged", x, y + 227, 40, WHITE)
    c.stacked([all_prs.merged, all_prs.closed], [GREEN, GRAY], x, y + 302, w)
    for i, (label, outcome) in enumerate(zip(("AW-started", f"Human-started{starred}", "Other / unattributed"), s.outcomes)):
        row_y = y + 390 + i * 70
        t(label, x, row_y)
        t(outcome.merge_rate, x + 570, row_y, 40, WHITE, True)
        t(outcome.ratio, x + 815, row_y, 38)
    t(f"Rates exclude {all_prs.opened:,} open PRs. n/a = undefined denominator. Includes tests and placeholders; not causal quality.",
      x, foot, 34, width=w, max_lines=2)

    x, y, w, foot = c.card(2, "Time to merge")
    t(percent(s.under(6), all_prs.merged, 0), x, y, 102, BLUE, True)
    t("of merged PRs took under 6 hours", x + 310, y + 24, 40, WHITE, True, width=w - 310, max_lines=2)
    median = "n/a" if s.median_hours is None else decimal(Fraction(str(s.median_hours)) * 60, 0)
    p90 = "n/a" if s.p90_hours is None else decimal(Fraction(str(s.p90_hours)), 1)
    t(f"Median: {median} min | P90: {p90} h | n = {all_prs.merged:,}", x, y + 145, 38)
    if len(s.bins) > 10:
        raise LayoutError("At most ten duration bands fit; combine bands explicitly, preserving the 1/6/24-hour boundaries.")
    baseline, plot_height = y + 510, 235
    maximum = max(b.count for b in s.bins)
    maximum = max(1, math.ceil(maximum / 1000) * 1000 if maximum > 1000 else maximum)
    slot = w / len(s.bins)
    for i in (1, 2, 3):
        gy = baseline - i / 4 * plot_height
        c.draw.line((x, gy, x + w, gy), fill=GRAY, width=2)
    for i, bucket in enumerate(s.bins):
        height = bucket.count / maximum * plot_height
        middle = x + (i + 0.5) * slot
        if bucket.count:
            c.draw.rectangle((middle - slot * .29, baseline - height, middle + slot * .29, baseline), fill=BLUE)
        label = bucket.label.replace(" min", "m").replace(" hours", "h").replace(" days", "d")
        c.center(f"{bucket.count:,}", middle, baseline - height - 53, bold=True, width=slot - 4)
        c.center(label, middle, baseline + 19, width=slot - 4)
    t(f"{percent(s.under(1), all_prs.merged)} under 1 hour | {percent(s.under(24), all_prs.merged)} under 24 hours",
      x, y + 579, 34, WHITE, True)
    t("PR creation to merge; merged PRs only. Unequal-duration bands show counts, not density.",
      x, foot, 34, width=w, max_lines=2)

    x, y, w, foot = c.card(3, "Community impact")
    t(f"{s.landed:,}", x, y, 112, GREEN, True)
    t("issues landed in main", x + 295, y + 33, 46, WHITE, True)
    t(f"{s.eligible_issues:,} eligible issues / {s.labelled_issues:,} community-labelled", x, y + 145, 38)
    t(f"{s.authors:,}", x, y + 232, 78, BLUE, True)
    t("Author accounts", x, y + 333)
    t(percent(s.landed, s.eligible_issues), x + 565, y + 232, 78, GREEN, True)
    t("Eligible issues landed", x + 565, y + 333)
    c.stacked([s.landed, s.closed_no_link, s.open_issues], [GREEN, GRAY, BLUE], x, y + 411, w)
    t(f"{s.landed:,} landed / {s.closed_no_link:,} closed, no merged link / {s.open_issues:,} open", x, y + 474, 34)
    t(f"{s.cca_delivery:,} / {s.delivery_prs:,} delivery PRs by CCA ({percent(s.cca_delivery, s.delivery_prs)})", x, y + 548, 39, WHITE, True)
    t("Non-member account proxy, not verified people. Main-branch integration is not proof of release.",
      x, foot, 34, width=w, max_lines=2)

    x, y, w, foot = c.card(4, "Go / JS code map")
    t(f"{s.lines:,}", x, y, 88, BLUE, True)
    t(f"nonblank lines in {sum(g.files for g in s.groups):,} files", x, y + 112, 40, WHITE)
    go = sum(g.lines for g in s.groups if g.language == "Go")
    tests = sum(g.lines for g in s.groups if g.role == "Tests")
    t(f"{percent(go, s.lines)} Go | {percent(s.lines - go, s.lines)} JS | {percent(tests, s.lines)} in test files", x, y + 184, 37)
    left = float(x)
    colors = {("Go", "Implementation"): "#1F6FEB", ("Go", "Tests"): "#1158A7",
              ("JavaScript", "Implementation"): "#238636", ("JavaScript", "Tests"): "#196C2E"}
    for lang in ("Go", "JavaScript"):
        groups = [g for g in s.groups if g.language == lang]
        language_lines = sum(g.lines for g in groups)
        if language_lines == 0:
            continue
        block_width = w * language_lines / s.lines
        top = float(y + 250)
        for role in ("Implementation", "Tests"):
            g = next(group for group in groups if group.role == role)
            block_height = 140 * g.lines / language_lines
            if g.lines:
                c.draw.rectangle((left, top, left + block_width, top + block_height), fill=colors[(lang, role)])
            top += block_height
        left += block_width
    if s.lines == 0:
        t("No nonblank Go / JS lines", x, y + 280, 38)
    ordered = sorted(s.groups, key=lambda g: (g.language != "Go", g.role != "Implementation"))
    for i, group in enumerate(ordered):
        gy = y + 412 + i * 45
        color = colors[(group.language, group.role)]
        c.draw.rectangle((x, gy + 10, x + 18, gy + 32), fill=color)
        label = f"{'Go' if group.language == 'Go' else 'JS'} {'impl.' if group.role == 'Implementation' else 'tests'}"
        t(label, x + 34, gy, 32)
        t(f"{group.lines:,} lines / {group.files:,} files", x + 340, gy, 32)
    t("Comments included; not test coverage. Go / JS only; detected generated / fixture content filtered.",
      x, foot, 34, width=w, max_lines=2)

    x, y, w, foot = c.card(5, "Weekly code change")
    t(f"{len(s.weeks)} {'week' if len(s.weeks) == 1 else 'weeks'}", x, y, 88, BLUE, True)
    added, deleted = sum(r.added for r in s.weeks), sum(r.deleted for r in s.weeks)
    t(f"{decimal(Fraction(added, 1000000), 2)}M added / {decimal(Fraction(deleted, 1000000), 2)}M deleted", x, y + 112, 44, WHITE, True)
    if s.weeks:
        if len(s.weeks) > 260:
            raise LayoutError("More than 260 weekly periods would be unreadable; use a wider timeline or explicit aggregation.")
        t(f"{s.weeks[0].start.isoformat()} - {s.weeks[-1].start.isoformat()} (week starts)", x, y + 184)
        left, right, zero = x + 105, x + w - 10, y + 413
        maximum = max(1, max(max(r.added, r.deleted) for r in s.weeks))
        scale = 138 / maximum
        slot = (right - left) / len(s.weeks)
        for tick in (maximum, 0, -maximum):
            ly = zero - tick * scale
            c.draw.line((left, ly, right, ly), fill=GRAY, width=2)
            label = str(tick) if maximum < 1000 else decimal(Fraction(tick, 1000), 0) + "K"
            t(label, x, ly - 20, 32)
        for i, week in enumerate(s.weeks):
            bx = left + i * slot
            if week.added:
                c.draw.rectangle((bx + slot * .05, zero - week.added * scale, bx + slot * .47, zero), fill=GREEN)
            if week.deleted:
                c.draw.rectangle((bx + slot * .54, zero, bx + slot * .98, zero + week.deleted * scale), fill=RED)
        for i in sorted({round(j * (len(s.weeks) - 1) / 3) for j in range(4)}):
            middle = left + (i + .5) * slot
            label = s.weeks[i].start.strftime("%m-%d")
            length = c.draw.textlength(label, font=c.face(32))
            t(label, max(left, min(middle - length / 2, right - length)), zero + 173, 32)
        t("Green: added | Red: deleted", x + 150, y + 235, 32)
    else:
        t("No weekly history records", x, y + 280, 38)
    t("First-parent Go / JS diff lines, including blanks. Renames count delete/add; boundary weeks may be partial.",
      x, foot + 16, 32, width=w, max_lines=2)
    x, y, w, foot = c.card(6, "Lines added / removed per PR")
    draw_change_histogram(c, s.changes, all_prs.total, x, y, w)
    t("Shared equal-width bins; counts, not line totals. Every census PR contributes once to each series, including zero values.",
      x, foot, 34, width=w, max_lines=2)
    t("GitHub PR additions/deletions across all files; merged, closed-unmerged and open PRs. Not weekly Git churn.",
      x, foot + 55, 32, width=w)
    x, y, w, foot = c.card(7, "Comments and commits per PR: human vs agent")
    draw_activity(c, s.activity, all_prs.total, x, y, w)
    t(f"Grouped by comment writer and primary commit author, never PR creator. Unattributed: {s.activity[2].comments:,} comments / {s.activity[2].commits:,} commits.",
      x, foot, 34, width=w, max_lines=2)
    t("Conversation comments only; no inline reviews. Linked primary commit author only, not committer/coauthors. Bots may be non-AI.",
      x, foot + 55, 32, width=w)
    c.clip = (0, 0, WIDTH, HEIGHT)
    collected = s.community_collected.astimezone(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    t(f"Snapshots: PR census {s.snapshot.isoformat()} | Community {collected}", 72, 4910, 32)
    t(f"Code: {s.commit[:12]} | Sources: GitHub API + pinned Git history | AW = agentic workflow; CCA = Copilot coding agent", 72, 4965, 30)
    c.verify()
    return c


def save(canvas: Canvas, output: Path) -> None:
    """Publish only a complete, verified PNG; leave an existing file intact on error."""
    canvas.verify()
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(dir=output.parent, suffix=".png", delete=False) as handle:
        temporary = Path(handle.name)
    try:
        canvas.image.save(temporary, format="PNG", dpi=(300, 300))
        with Image.open(temporary) as check:
            check.verify()
        temporary.replace(output)
    finally:
        temporary.unlink(missing_ok=True)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--input", type=Path, help="JSON object containing the seven source summaries.")
    source.add_argument("--data-dir", type=Path, help="Directory containing the seven canonical summary filenames.")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--title", default="Development, in numbers")
    parser.add_argument("--font-regular", type=Path)
    parser.add_argument("--font-bold", type=Path)
    args = parser.parse_args()
    try:
        data = Statistics.from_sources(load_sources(args.data_dir or args.input, args.data_dir is not None))
        canvas = render(data, discover_fonts(args.font_regular, args.font_bold), args.title)
        save(canvas, args.output)
    except (DataError, LayoutError, OSError, ValueError) as error:
        parser.error(str(error))
    print(f"Saved {args.output}: {WIDTH} x {HEIGHT}, eight panels, validated counts and text layout.")


if __name__ == "__main__":
    main()
