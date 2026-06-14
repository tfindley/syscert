import { chromium } from "playwright";

const base = process.env.BASE || "http://localhost:4321";
const shots = [
  { name: "home-hero", url: "/", width: 1440, height: 900, full: false, dsf: 1 },
  { name: "home-desktop", url: "/", width: 1440, height: 900, full: true, dsf: 1 },
  { name: "install-desktop", url: "/install/", width: 1440, height: 900, full: true, dsf: 1 },
  { name: "home-mobile", url: "/", width: 390, height: 844, full: true, dsf: 2 },
];

const browser = await chromium.launch();
for (const s of shots) {
  const ctx = await browser.newContext({
    viewport: { width: s.width, height: s.height },
    deviceScaleFactor: s.dsf,
    colorScheme: "dark",
  });
  const page = await ctx.newPage();
  await page.goto(base + s.url, { waitUntil: "networkidle", timeout: 45000 });
  await page.waitForTimeout(1500); // fonts + reveal animations settle
  await page.screenshot({ path: `/tmp/syscert-${s.name}.png`, fullPage: s.full });
  await ctx.close();
  console.log("shot", s.name);
}
await browser.close();
console.log("done");
