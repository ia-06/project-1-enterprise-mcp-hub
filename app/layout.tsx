import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Enterprise MCP Hub — Customer 360 Intelligence",
  description:
    "A native Model Context Protocol (MCP) server connecting a Next.js frontend to PostgreSQL, Jira, and Salesforce data sources with built-in resilience.",
  keywords: ["MCP", "enterprise", "customer 360", "Salesforce", "Jira", "PostgreSQL", "Next.js"],
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap"
          rel="stylesheet"
        />
      </head>
      <body className="mcp-app-body">
        {children}
      </body>
    </html>
  );
}
