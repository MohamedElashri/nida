import pluginSyntaxHighlight from "@11ty/eleventy-plugin-syntaxhighlight";
import { feedPlugin } from "@11ty/eleventy-plugin-rss";
import pluginNavigation from "@11ty/eleventy-navigation";
import markdownItAnchor from "markdown-it-anchor";

/** @param {import("@11ty/eleventy").UserConfig} eleventyConfig */
export default async function (eleventyConfig) {
  // Copy the contents of the `public` folder to the output folder.
  // For example, `./public/css/` ends up in `_site/css/`.
  eleventyConfig.addPassthroughCopy({ "./public/": "/" });
  eleventyConfig.addPassthroughCopy("./content/**/*.{svg,webp,png,jpeg}");

  eleventyConfig.addPlugin(pluginSyntaxHighlight);
  eleventyConfig.addPlugin(pluginNavigation);
  eleventyConfig.addPlugin(feedPlugin, {
    type: "atom", // or "rss", "json"
    outputPath: "/feed/feed.xml",
    collection: { name: "posts", limit: 10 },
  });

  eleventyConfig.addFilter("readableDate", (dateObj) => dateObj.toISOString());
  eleventyConfig.addShortcode("currentBuildDate", () => new Date().toISOString());
  eleventyConfig.addPairedShortcode("aside", (content, kind) => `<aside class="${kind}">${content}</aside>`);
  eleventyConfig.addCollection("featured", (api) => api.getFilteredByTag("featured"));
  eleventyConfig.amendLibrary("md", (mdLib) => mdLib.use(markdownItAnchor));
  eleventyConfig.setFrontMatterParsingOptions({
    excerpt: true,
    excerpt_separator: "<!-- excerpt -->",
  });
}

export const config = {
  templateFormats: ["md", "njk", "html", "liquid"],
  markdownTemplateEngine: "njk",
  htmlTemplateEngine: "njk",
  pathPrefix: "/blog/", // served from https://example.com/blog/
  dir: {
    input: "content",
    includes: "../_includes",
    data: "../_data",
    output: "_site",
  },
};
