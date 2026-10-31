import type { NextConfig } from "next";

const backendAPIURL = (
  process.env.BACKEND_API_URL ??
  "https://invisible-receptor-smell-shine.trycloudflare.com/api"
).replace(/\/$/, "");

const nextConfig: NextConfig = {
  /* config options here */
  cacheComponents: true,
  partialPrefetching: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${backendAPIURL}/:path*`,
      },
    ];
  },
};

export default nextConfig;
