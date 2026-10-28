"use client";

type SponsorAnalyticsEvent = Record<string, string>;

declare global {
  interface Window {
    dataLayer?: Array<Record<string, string>>;
    gtag?: (command: "event", eventName: string, parameters: SponsorAnalyticsEvent) => void;
  }
}

export function trackSponsorEvent(
  eventName: string,
  parameters: SponsorAnalyticsEvent = {},
) {
  const payload = { event: eventName, ...parameters };

  if (window.dataLayer) {
    window.dataLayer.push(payload);
  } else {
    window.gtag?.("event", eventName, parameters);
  }

  window.dispatchEvent(new CustomEvent("atonix:analytics", { detail: payload }));
}
