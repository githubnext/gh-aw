import { basename } from "node:path";
export function toNumber(value) {
    const numeric = Number(value ?? 0);
    return Number.isFinite(numeric) ? numeric : 0;
}
export function normalizeWorkflowID(value) {
    const raw = String(value ?? "").trim();
    if (!raw)
        return "";
    let name = basename(raw);
    const lowerName = name.toLowerCase();
    for (const suffix of [".lock.yml", ".yml", ".yaml", ".md"]) {
        if (lowerName.endsWith(suffix)) {
            name = name.slice(0, -suffix.length);
            break;
        }
    }
    return name.trim();
}
export function forecastDaysForWindow(window) {
    return window?.id === "1mo" ? 30 : 7;
}
export function getForecastMonthlyAIC(forecast) {
    if (!forecast || typeof forecast !== "object")
        return 0;
    const monteCarloP50 = toNumber(forecast.monthly_monte_carlo?.p50_projected_aic);
    if (monteCarloP50 > 0)
        return monteCarloP50;
    return toNumber(forecast.monthly_projected_aic);
}
export function applyForecastToUsageSummary(items, forecastWorkflows = []) {
    const forecastEntries = forecastWorkflows
        .map(forecast => [normalizeWorkflowID(forecast?.workflow_id || forecast?.workflow_path), getForecastMonthlyAIC(forecast)])
        .filter(([workflowID]) => Boolean(workflowID));
    const forecastByWorkflow = new Map(forecastEntries);
    return items.map(item => ({
        ...item,
        monthly_forecast_aic: forecastByWorkflow.get(item.workflow_id) ?? 0,
    }));
}
export function buildUsageSummary(runs, window, forecastWorkflows = []) {
    const usageByWorkflow = new Map();
    const effectiveDays = Number(window?.days ?? 0);
    if (!Number.isFinite(effectiveDays) || effectiveDays <= 0) {
        throw new Error(`report window '${window?.id ?? "unknown"}' is missing a valid positive day count.`);
    }
    for (const run of runs) {
        const workflowPath = typeof run?.workflow_path === "string" ? run.workflow_path.trim() : "";
        const workflowID = normalizeWorkflowID(workflowPath || run?.workflow_name);
        if (!workflowID)
            continue;
        const workflowName = String(run?.workflow_name ?? workflowID).trim() || workflowID;
        const aic = toNumber(run?.aic);
        const entry = usageByWorkflow.get(workflowID) ?? {
            workflow_id: workflowID,
            workflow_name: workflowName,
            workflow_path: workflowPath,
            run_count: 0,
            total_aic: 0,
            cost_per_run: 0,
            daily_aic: 0,
            monthly_forecast_aic: 0,
            last_run_at: "",
        };
        entry.run_count += 1;
        entry.total_aic += aic;
        if (!entry.workflow_path && workflowPath) {
            entry.workflow_path = workflowPath;
        }
        if (!entry.workflow_name && workflowName) {
            entry.workflow_name = workflowName;
        }
        const createdAt = typeof run?.created_at === "string" ? run.created_at : "";
        if (createdAt && (!entry.last_run_at || createdAt > entry.last_run_at)) {
            entry.last_run_at = createdAt;
        }
        usageByWorkflow.set(workflowID, entry);
    }
    const items = Array.from(usageByWorkflow.values())
        .map(entry => {
        const costPerRun = entry.run_count > 0 ? entry.total_aic / entry.run_count : 0;
        const dailyAIC = entry.total_aic / effectiveDays;
        return {
            ...entry,
            cost_per_run: costPerRun,
            daily_aic: dailyAIC,
            monthly_forecast_aic: 0,
        };
    })
        .sort((a, b) => {
        const dailyDelta = b.daily_aic - a.daily_aic;
        if (dailyDelta !== 0)
            return dailyDelta;
        return b.cost_per_run - a.cost_per_run;
    });
    return applyForecastToUsageSummary(items, forecastWorkflows);
}
