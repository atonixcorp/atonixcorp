import assert from "node:assert/strict";
import test from "node:test";
import { products, publicNavigation, solutions } from "./content.ts";

const baseURL = process.env.PUBLIC_TEST_BASE_URL;
assert.ok(baseURL, "Set PUBLIC_TEST_BASE_URL to a running local preview before running these integration tests.");
const publicPaths = [
  "/", "/products", "/solutions", "/platform", "/developers", "/about",
  "/portal", "/contact", "/sponsor", "/login", "/onboard",
  "/developers/documentation", "/developers/api-cli", "/developers/open-source",
  "/developers/community", "/about/roadmap",
  ...products.map((product) => `/products/${product.slug}`),
  ...solutions.map((solution) => `/solutions/${solution.slug}`),
];
const pages = new Map();
async function load(path) {
  if (!pages.has(path)) {
    const response = await fetch(new URL(path, baseURL), { redirect: "manual" });
    assert.equal(response.status, 200, `${path} should be publicly available`);
    pages.set(path, await response.text());
  }
  return pages.get(path);
}

test("public destinations render distinct headings with a shared accessible shell", async () => {
  const headings = new Set();
  const titles = new Set();
  for (const path of publicPaths) {
    const html = await load(path);
    const h1 = [...html.matchAll(/<h1(?:\s[^>]*)?>(.*?)<\/h1>/gs)];
    assert.equal(h1.length, 1, `${path} must have one primary heading`);
    const heading = h1[0][1].replace(/<[^>]*>/g, "").trim();
    assert.ok(!headings.has(heading), `${path} repeats another page's primary heading`);
    headings.add(heading);
    assert.equal([...html.matchAll(/<main(?:\s|>)/g)].length, 1, `${path} must have one main landmark`);
    assert.match(html, /id="public-content"/, `${path} must provide the skip-link target`);
    assert.match(html, /id="top"/, `${path} must provide the back-to-top target`);
    const title = html.match(/<title>(.*?)<\/title>/s)?.[1];
    assert.ok(title && !titles.has(title), `${path} must have a distinct document title`);
    titles.add(title);
    assert.match(html, /<meta name="description" content="[^"]+"/, `${path} needs a description`);
  }
});

test("navigation and public-page links resolve, including fragment destinations", async () => {
  const links = new Set(publicNavigation.flatMap((group) => [
    group.href, ...group.groups.flatMap((section) => section.links.map((link) => link.href)),
  ]));
  for (const path of publicPaths) {
    const html = await load(path);
    for (const match of html.matchAll(/<a\b[^>]*\bhref="([^"]+)"/g)) {
      const url = new URL(match[1].replaceAll("&amp;", "&"), new URL(path, baseURL));
      if (url.origin === new URL(baseURL).origin) links.add(`${url.pathname}${url.hash}`);
    }
  }
  for (const href of links) {
    if (!href.startsWith("/")) continue;
    const url = new URL(href, baseURL);
    if (publicPaths.includes(url.pathname)) {
      const html = await load(url.pathname);
      if (url.hash) assert.ok(html.includes(`id="${decodeURIComponent(url.hash.slice(1))}"`), `Missing anchor: ${href}`);
    } else {
      const response = await fetch(url, { redirect: "manual" });
      if (url.pathname === "/support") {
        assert.equal(response.status, 200, "The existing support shell must remain available; its APIs enforce authentication");
      } else {
        assert.equal(response.status, 307, `${href} must resolve to an authenticated workspace, not a missing page`);
        assert.ok(response.headers.get("location")?.includes("/login"), `${href} must preserve the sign-in boundary`);
      }
    }
  }
});

test("public menu destinations open dedicated pages, not homepage or hub sections", async () => {
  for (const menu of publicNavigation) {
    const destinations = new Set();
    for (const link of menu.groups.flatMap((group) => group.links)) {
      const url = new URL(link.href, baseURL);
      assert.equal(url.origin, new URL(baseURL).origin, `${link.label} needs its own platform page`);
      assert.notEqual(url.pathname, "/", `${link.label} must not open the homepage`);
      assert.equal(url.hash, "", `${link.label} must not be a section of another page`);
      assert.ok(!destinations.has(url.pathname), `${link.label} must have a distinct menu destination`);
      destinations.add(url.pathname);
      const html = await load(url.pathname);
      assert.match(html, /<h1(?:\s|>)/, `${link.label} must render its own page`);
    }
  }
});

test("unknown products and solutions return 404 rather than a generic marketing page", async () => {
  for (const path of ["/products/not-a-product", "/solutions/not-a-solution"]) {
    const response = await fetch(new URL(path, baseURL), { redirect: "manual" });
    assert.equal(response.status, 404, path);
  }
});

test("product pages publish integration limitations beside their workspace entry", async () => {
  for (const product of products) {
    const html = await load(`/products/${product.slug}`);
    assert.match(html, /id="product-availability"/);
    assert.ok(html.includes(`href="${product.workspace}"`), `${product.name} needs a working workspace destination`);
    for (const consideration of product.considerations) {
      const escaped = consideration.replaceAll("&", "&amp;").replaceAll("'", "&#x27;").replaceAll('"', "&quot;");
      assert.ok(html.includes(escaped), `${product.name} is missing an integration limitation`);
    }
  }
});
