import { chromium } from "playwright";

const base = process.env.BASE || "http://localhost:4321";
const W = Number(process.env.W || 375);
const pages = [
  "/",
  "/install/",
  "/changelog/",
  "/docs/quick-start/",
  "/docs/configuration/",
  "/docs/examples/",
  "/docs/advanced-install/",
  "/docs/distributing/",
  "/docs/troubleshooting/",
  "/docs/faq/",
  "/docs/roadmap/",
];

const browser = await chromium.launch();
for (const url of pages) {
  const ctx = await browser.newContext({
    viewport: { width: W, height: 800 },
    deviceScaleFactor: 2,
    colorScheme: "dark",
  });
  const page = await ctx.newPage();
  await page.goto(base + url, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(800);

  const report = await page.evaluate((vw) => {
    const docW = document.documentElement.scrollWidth;
    const offenders = [];
    const all = document.body.querySelectorAll("*");
    for (const el of all) {
      const r = el.getBoundingClientRect();
      // element extends past the right edge of the viewport
      if (r.right > vw + 1 && r.width > 0) {
        offenders.push({
          tag: el.tagName.toLowerCase(),
          cls: (el.className && el.className.toString().slice(0, 40)) || "",
          right: Math.round(r.right),
          width: Math.round(r.width),
          text: (el.textContent || "").trim().slice(0, 30),
        });
      }
    }
    // keep the widest few, de-noise nested duplicates
    offenders.sort((a, b) => b.right - a.right);
    return { docW, vw, overflow: docW - vw, offenders: offenders.slice(0, 8) };
  }, W);

  const slug = url.replace(/\//g, "_") || "_home";
  await page.screenshot({ path: `/tmp/mob${slug}.png`, fullPage: true });

  const flag = report.overflow > 1 ? `⚠ OVERFLOW +${report.overflow}px` : "ok";
  console.log(`\n${url}  (docW=${report.docW}, vw=${report.vw})  ${flag}`);
  if (report.overflow > 1) {
    for (const o of report.offenders) {
      console.log(`   ${o.tag}.${o.cls}  right=${o.right} w=${o.width}  "${o.text}"`);
    }
  }
  await ctx.close();
}
await browser.close();
console.log("\nscreenshots: /tmp/mob_*.png");
