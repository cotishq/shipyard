"use client";

import { SHIPYARD_TOKEN_KEY } from "./constants";

export function readToken(): string {
  if (typeof window === "undefined") {
    return "";
  }

  return window.localStorage.getItem(SHIPYARD_TOKEN_KEY) ?? "";
}

export function writeToken(token: string): void {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(SHIPYARD_TOKEN_KEY, token);
}

export function clearToken(): void {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.removeItem(SHIPYARD_TOKEN_KEY);
}
