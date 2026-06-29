export const CACHE_TTL_MS = 60_000;
export const DEFAULT_LOG_TIMEOUT_MINUTES = 1;
export const DEFAULT_REPORT_WINDOW_ID = "7d" as const;
export const DEFAULT_RUN_COUNT = 100;
export const MAX_LOG_CONTINUATIONS = 6;

export type ReportWindowID = "3d" | "7d" | "1mo";

export interface ReportWindow {
  id: ReportWindowID;
  label: string;
  startDate: string;
  days: number;
}

export const REPORT_WINDOWS: Record<ReportWindowID, ReportWindow> = {
  "3d": { id: "3d", label: "3 days", startDate: "-3d", days: 3 },
  "7d": { id: "7d", label: "7 days", startDate: "-1w", days: 7 },
  "1mo": { id: "1mo", label: "1 month", startDate: "-1mo", days: 30 },
};

export function getReportWindow(windowId?: string | null): ReportWindow {
  if (windowId && windowId in REPORT_WINDOWS) {
    return REPORT_WINDOWS[windowId as ReportWindowID];
  }
  return REPORT_WINDOWS[DEFAULT_REPORT_WINDOW_ID];
}
