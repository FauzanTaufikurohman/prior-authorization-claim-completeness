import type { NextConfig } from "next";

const backendAPIURL = (
  process.env.BACKEND_API_URL ?? "http://localhost:8081/api"
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
