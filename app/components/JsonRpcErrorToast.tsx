"use client";

import React, { useEffect, useRef } from "react";
import type { APIErrorCode } from "@/app/lib/types";

type Props = {
  errorCode: string | null;
  onClose: () => void;
};

// Map API error codes to user-friendly messages
const ERROR_MESSAGES: Record<string, { title: string; message: string }> = {
  INVALID_RPC: {
    title: "Data Decoding Error",
    message: "Temporary issue decoding data from the backend. This usually resolves on its own.",
  },
  RPC_METHOD_NOT_FOUND: {
    title: "Unknown Request",
    message: "The requested operation is not recognised by the MCP server.",
  },
  SALESFORCE_DOWN: {
    title: "CRM Temporarily Unavailable",
    message:
      "Salesforce CRM is unreachable right now. Account data may be sourced from the cache. Retry later.",
  },
  SALESFORCE_ERROR: {
    title: "Salesforce Error",
    message: "An error occurred while fetching account data from Salesforce.",
  },
  SALES_DOWN: {
    title: "Sales Data Unavailable",
    message: "The sales database is temporarily unreachable. Please retry in a moment.",
  },
  JIRA_DOWN: {
    title: "Jira Unavailable",
    message: "Engineering ticket data cannot be loaded right now.",
  },
  MISSING_ACCOUNT_ID: {
    title: "Missing Account ID",
    message: "Please enter or select a valid Salesforce Account ID.",
  },
  UNKNOWN_ERROR: {
    title: "Unexpected Error",
    message: "Something went wrong. Please refresh and try again.",
  },
};

const AUTO_DISMISS_MS = 8000;

/**
 * JsonRpcErrorToast
 *
 * Renders a fixed-position toast notification when an API error code is
 * present. Auto-dismisses after AUTO_DISMISS_MS ms or on manual close.
 *
 * System Resilience Matrix — Scenario B:
 *   - INVALID_RPC (-32700 / -32600) → "Temporary issue decoding data."
 *   - RPC_METHOD_NOT_FOUND (-32601) → "Unknown Request."
 */
export default function JsonRpcErrorToast({ errorCode, onClose }: Props) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Auto-dismiss
  useEffect(() => {
    if (!errorCode) return;

    timerRef.current = setTimeout(() => {
      onClose();
    }, AUTO_DISMISS_MS);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [errorCode, onClose]);

  if (!errorCode) return null;

  const info = ERROR_MESSAGES[errorCode] ?? ERROR_MESSAGES.UNKNOWN_ERROR;

  return (
    <div
      className="mcp-toast-container"
      role="alert"
      aria-live="assertive"
      aria-atomic="true"
      id="jsonrpc-error-toast"
    >
      <div className="mcp-toast">
        <span className="mcp-toast-icon" aria-hidden="true">⚠️</span>

        <div className="mcp-toast-body">
          <p className="mcp-toast-title">{info.title}</p>
          <p className="mcp-toast-message">{info.message}</p>
          <span className="mcp-toast-code" aria-label={`Error code: ${errorCode}`}>
            {errorCode}
          </span>
        </div>

        <button
          className="mcp-toast-close"
          onClick={onClose}
          aria-label="Dismiss error notification"
          id="toast-close-button"
        >
          ×
        </button>
      </div>
    </div>
  );
}
