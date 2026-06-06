/** @type {import('next').NextConfig} */
const nextConfig = {
  // Enable App Router (default in Next.js 14)
  experimental: {},
  // Ensure server-side env vars are available in route handlers
  env: {
    GO_RPC_URL: process.env.GO_RPC_URL || 'http://localhost:8080/rpc',
  },
};

module.exports = nextConfig;
