import type { Metadata } from "next";
import "./globals.css";
import Nav from "./components/Nav";
import Footer from "./components/Footer";

export const metadata: Metadata = {
  title: "Enterprise MCP Hub | Connect Your Data Stack.",
  description:
    "A production-grade Model Context Protocol server that wires Jira Cloud, Salesforce CRM, and Postgres into any AI agent workflow via JSON-RPC 2.0. Zero mock data. Live enterprise APIs.",
  keywords: ["MCP", "Model Context Protocol", "enterprise middleware", "Salesforce", "Jira", "Supabase", "AI agents", "JSON-RPC"],
  openGraph: {
    title: "Enterprise MCP Hub",
    description: "Connect your enterprise data stack. In one middleware.",
    type: "website",
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400&display=swap"
          rel="stylesheet"
        />
      </head>
      <body>
        <Nav />
        {children}
        <Footer />
      </body>
    </html>
  );
}
