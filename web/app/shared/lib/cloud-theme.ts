"use client";

import { useCallback, useEffect, useState } from "react";

export type CloudTheme = "light" | "dark";
const themeKey = "atonixcorp-cloud-theme";

export function useCloudTheme(): [CloudTheme, () => void] {
  const [theme, setTheme] = useState<CloudTheme>("light");

  useEffect(() => {
    let savedTheme: CloudTheme = "light";
    try {
      const saved = localStorage.getItem(themeKey);
      if (saved === "dark" || saved === "light") savedTheme = saved;
    } catch {
      savedTheme = document.documentElement.dataset.theme === "dark" ? "dark" : "light";
    }

    document.documentElement.dataset.theme = savedTheme;
    const syncInitialState = window.setTimeout(() => setTheme(savedTheme), 0);

    const syncTheme = (event: StorageEvent) => {
      if (event.key !== themeKey) return;
      const nextTheme = event.newValue === "dark" ? "dark" : "light";
      document.documentElement.dataset.theme = nextTheme;
      setTheme(nextTheme);
    };
    window.addEventListener("storage", syncTheme);
    return () => {
      window.clearTimeout(syncInitialState);
      window.removeEventListener("storage", syncTheme);
    };
  }, []);

  const toggleTheme = useCallback(() => {
    setTheme((current) => {
      const nextTheme = current === "dark" ? "light" : "dark";
      document.documentElement.dataset.theme = nextTheme;
      try {
        localStorage.setItem(themeKey, nextTheme);
      } catch {
        // Keep the current-session theme even if browser storage is unavailable.
      }
      return nextTheme;
    });
  }, []);

  return [theme, toggleTheme];
}
