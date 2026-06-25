package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/github/gh-aw/pkg/impactscore"
)

type uiPayload struct {
	Repo          string              `json:"repo"`
	GeneratedAt   string              `json:"generated_at"`
	WorkflowRanks []workflowRankForUI `json:"workflow_ranks"`
	WorkItems     []workItemForUI     `json:"work_items"`
}

type workflowRankForUI struct {
	Workflow              string  `json:"workflow"`
	AttributedImpactScore float64 `json:"attributed_impact_score"`
	LinkedItems           int     `json:"linked_items"`
	TotalAICCost          float64 `json:"total_aic_cost"`
	ActionMinutes         float64 `json:"action_minutes"`
}

type workItemForUI struct {
	Number           int                          `json:"number"`
	ItemType         string                       `json:"item_type"`
	State            string                       `json:"state"`
	StateReason      string                       `json:"state_reason,omitempty"`
	Title            string                       `json:"title"`
	HTMLURL          string                       `json:"html_url"`
	LastImpactScore  float64                      `json:"last_impact_score"`
	ScoreSource      string                       `json:"score_source"`
	ScoreExplanation impactscore.ScoreExplanation `json:"score_explanation,omitzero"`
	SourceWorkflows  []string                     `json:"source_workflows"`
}

func writeUIArtifact(outDir string, result output) error {
	data, err := json.Marshal(uiPayloadFromOutput(result))
	if err != nil {
		return err
	}
	var html bytes.Buffer
	fprintf := func(format string, args ...any) { fmt.Fprintf(&html, format, args...) }
	fprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Impact Score Dashboard</title>
<style>
  :root { color-scheme: dark; --font-sans:"Mona Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; --font-mono:"Mona Sans Mono", ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; --ink:#F2F5F3; --muted:#B6BFB8; --line:#3f4942; --panel:#101411; --canvas:#232925; --canvas-inset:#0A241B; --canvas-subtle:#313a34; --accent:#0FBF3E; --accent-emphasis:#5FED83; --amber:#DB9D00; --red:#FE4C25; --blue:#3094FF; --gray:#909692; }
  * { box-sizing:border-box; }
  body { margin:0; font:500 15px/1.45 var(--font-sans); color:var(--ink); background:var(--panel); }
  section { padding:20px 24px 28px; overflow:auto; }
  h2 { margin:0; font-size:19px; line-height:1.2; font-weight:800; }
  .run-header { display:grid; gap:3px; margin:0 0 14px; }
  .run-title { font-size:20px; line-height:1.2; font-weight:800; }
  .run-subtitle, .muted { color:var(--muted); font-size:13px; line-height:1.35; font-weight:650; }
  .tabs { display:flex; gap:8px; margin-bottom:16px; }
  .tabs button, .action-button { border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:6px; padding:8px 13px; font:inherit; font-weight:800; cursor:pointer; }
  .tabs button:hover, .action-button:hover { background:var(--canvas-subtle); border-color:var(--muted); }
  .tabs button.active { background:var(--canvas-subtle); box-shadow:inset 0 2px 0 var(--accent); }
  .action-button.primary { background:var(--accent); border-color:var(--accent-emphasis); color:#04130A; }
  .panel { display:none; }
  .panel.active { display:block; }
	.summary { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:12px; margin-bottom:16px; }
	.metric, .plot-card, .impact-rail { border:1px solid var(--line); border-radius:8px; background:var(--canvas); }
	.metric.workflow-count { grid-column:span 2; }
  .metric { padding:13px 14px; }
  .metric strong { display:block; font-size:24px; line-height:1.15; }
  .plot-stack { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:16px; margin-bottom:16px; align-items:stretch; }
  .plot-card { min-height:650px; display:flex; flex-direction:column; padding:16px; }
  .plot-card h2 { margin-bottom:12px; }
  .plot-card svg { display:block; width:100%%; min-height:570px; flex:1 1 auto; border-radius:6px; font-family:var(--font-sans); }
	.work-items-layout { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:16px; }
	.item-filter { width:100%%; border:1px solid var(--line); border-radius:6px; background:var(--panel); color:var(--ink); padding:8px 10px; font:inherit; font-weight:650; }
	.item-table { overflow:auto; border:1px solid var(--line); border-radius:7px; background:var(--panel); }
	.item-table table { width:100%%; border-collapse:collapse; table-layout:fixed; }
	.item-table th, .item-table td { padding:8px 9px; border-bottom:1px solid var(--line); text-align:left; vertical-align:top; font-size:12px; line-height:1.3; }
	.item-table th { position:sticky; top:0; z-index:1; background:var(--canvas-subtle); color:var(--muted); font-size:11px; text-transform:uppercase; }
	.item-table td { overflow-wrap:anywhere; }
	.item-table a { color:var(--ink); text-decoration:none; }
	.item-table a:hover { color:var(--accent-emphasis); text-decoration:underline; }
  .workflow-mark { transition:opacity 120ms ease, stroke-width 120ms ease; }
  .workflow-dim { opacity:0.22 !important; }
  .workflow-highlight { opacity:1 !important; stroke:#F2F5F3 !important; stroke-width:3 !important; }
  .workflow-bar.workflow-highlight { stroke-width:2 !important; }
  .impact-rail { max-height:700px; overflow:auto; display:grid; align-content:start; gap:8px; padding:16px; }
  .impact-score { justify-self:start; border:1px solid #0FBF3E66; border-radius:999px; background:#0FBF3E22; color:#8CF2A6; padding:2px 7px; font:800 11px/1.3 var(--font-mono); }
	@media (max-width: 1000px) { .plot-stack, .work-items-layout { grid-template-columns:1fr; } .summary { grid-template-columns:repeat(2,minmax(0,1fr)); } .metric.workflow-count { grid-column:span 2; } }
</style>
</head>
<body>
<main>
<section>
  <div class="run-header"><div id="repoTitle" class="run-title"></div><div id="subtitle" class="run-subtitle"></div></div>
	<div class="tabs"><button class="active" data-tab="workflows">Workflows</button><button data-tab="workItems">Work Items</button></div>
  <div id="workflows" class="panel active">
    <div class="summary" id="summary"></div>
    <div class="plot-stack">
      <div class="plot-card"><h2 id="costChartTitle">Workflow Impact Score / Cost Signal</h2><svg id="impactCostChart" viewBox="0 0 800 650" role="img" aria-label="Workflow impact score versus cost signal plot"></svg></div>
			<div class="plot-card"><h2>Workflow Impact Score Ranking</h2><svg id="impactRankChart" viewBox="0 0 800 650" role="img" aria-label="Workflow impact score ranking chart"></svg></div>
    </div>
  </div>
	<div id="workItems" class="panel">
		<div class="work-items-layout">
			<div class="impact-rail" aria-label="Agentic workflow work items"><h2>Workflow Work Items</h2><input id="workflowWorkFilter" class="item-filter" type="search" placeholder="Filter items"><div id="workflowWorkItems" class="item-table"></div></div>
			<div class="impact-rail" aria-label="Other work items"><h2>Other Work Items</h2><input id="otherWorkFilter" class="item-filter" type="search" placeholder="Filter items"><div id="otherWorkItems" class="item-table"></div></div>
		</div>
	</div>
</section>
</main>
<script id="payload" type="application/json">%s</script>
<script>
const payload = JSON.parse(document.getElementById('payload').textContent);
const zones = {'keep / scale':'#0FBF3E','optimize':'#DB9D00','waste review':'#FE4C25','needs cost':'#3094FF','monitor':'#909692'};
const hasAICCost = payload.workflow_ranks.some(row => Number(row.total_aic_cost || 0) > 0);
let hoveredWorkflow = '';
function fmt(n,d=2){ return Number(n||0).toLocaleString(undefined,{maximumFractionDigits:d}); }
function esc(s){ return String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }
function attr(s){ return esc(s).replace(/\n/g, '&#10;'); }
function costValue(row){ return hasAICCost ? Number(row.total_aic_cost || 0) : Number(row.action_minutes || 0); }
function costLabel(){ return hasAICCost ? 'AIC cost' : 'Actions minutes'; }
function costSummaryLabel(){ return hasAICCost ? 'Total AIC' : 'Total action min'; }
function costTooltip(row){ return 'AIC: ' + fmt(row.total_aic_cost) + '\nActions minutes: ' + fmt(row.action_minutes); }
function median(values){ if(!values.length){ return 0; } const mid = Math.floor(values.length/2); return values.length %% 2 ? values[mid] : (values[mid-1] + values[mid]) / 2; }
function thresholds(rows){ const costed = rows.filter(row => costValue(row) > 0); return {impact: median(costed.filter(row => row.attributed_impact_score > 0).map(row => row.attributed_impact_score).sort((a,b)=>a-b)), cost: median(costed.map(row => costValue(row)).sort((a,b)=>a-b))}; }
function displayZone(row, threshold){ const rowCost = costValue(row); const highImpact = row.attributed_impact_score >= threshold.impact && row.attributed_impact_score > 0; const highCost = rowCost >= threshold.cost && rowCost > 0; if(row.attributed_impact_score > 0 && rowCost <= 0){ return 'needs cost'; } if(highImpact && !highCost){ return 'keep / scale'; } if(highImpact && highCost){ return 'optimize'; } if(highCost){ return 'waste review'; } return 'monitor'; }
function rows(){ const base = payload.workflow_ranks.map(row => ({...row})); const threshold = thresholds(base); return base.map(row => ({...row, display_zone: displayZone(row, threshold), tuned_score: row.attributed_impact_score + row.linked_items * 0.15 - Math.log1p(costValue(row)) * 0.25})).sort((a,b)=>b.tuned_score-a.tuned_score); }
function setup(){ document.getElementById('repoTitle').textContent = payload.repo; document.getElementById('subtitle').textContent = 'generated ' + payload.generated_at; document.querySelectorAll('.tabs button').forEach(button => button.addEventListener('click', () => switchTab(button.dataset.tab))); document.getElementById('workflows').addEventListener('mouseover', event => { const mark = event.target.closest('[data-workflow]'); if(mark){ setWorkflowHover(mark.dataset.workflow); } else { clearWorkflowHover(); } }); document.getElementById('workflows').addEventListener('mouseout', event => { const mark = event.target.closest('[data-workflow]'); if(mark && !event.relatedTarget?.closest?.('[data-workflow]')){ clearWorkflowHover(); } }); render(); }
function render(){ renderSummary(); renderCostChart(); renderRankChart(); renderScoredWorkItems(); renderOtherWorkItems(); }
function renderSummary(){ const totalImpact = payload.workflow_ranks.reduce((sum,row)=>sum+row.attributed_impact_score,0); const totalCost = payload.workflow_ranks.reduce((sum,row)=>sum+costValue(row),0); document.getElementById('summary').innerHTML = '<div class="metric"><span class="muted">Total impact score</span><strong>' + fmt(totalImpact,1) + '</strong></div><div class="metric"><span class="muted">' + esc(costSummaryLabel()) + '</span><strong>' + fmt(totalCost,1) + '</strong></div><div class="metric workflow-count"><span class="muted">Workflows</span><strong>' + payload.workflow_ranks.length + '</strong></div>'; }
function renderCostChart(){ const workflowRows = rows().filter(row => row.attributed_impact_score > 0 || costValue(row) > 0 || row.linked_items > 0); const svg = document.getElementById('impactCostChart'); document.getElementById('costChartTitle').textContent = 'Workflow Impact Score / ' + costLabel(); const width = 800, height = 650, left = 72, top = 28, plotW = 680, plotH = 520; const costed = workflowRows.filter(row => costValue(row) > 0); const minCost = Math.max(0.1, (Math.min(...costed.map(row => costValue(row)), 1) || 0.1) * 0.72); const maxCost = Math.max(10, (Math.max(...costed.map(row => costValue(row)), 10) || 10) * 1.35); const maxImpact = Math.max(1, ...workflowRows.map(row => row.attributed_impact_score)) * 1.2; const threshold = thresholds(workflowRows); const xScale = cost => cost <= 0 ? left : left + ((Math.log10(cost)-Math.log10(minCost))/(Math.log10(maxCost)-Math.log10(minCost)))*plotW; const yScale = impact => top + plotH - (Math.max(0,impact)/maxImpact)*plotH; const thresholdX = threshold.cost > 0 ? xScale(threshold.cost) : 0; const thresholdY = threshold.impact > 0 ? yScale(threshold.impact) : 0; let out = '<rect width="100%%" height="100%%" fill="#232925"/><rect x="'+left+'" y="'+top+'" width="'+plotW+'" height="'+plotH+'" rx="6" fill="#101411" stroke="#3f4942"/>'; if(threshold.cost > 0 && threshold.impact > 0){ out += quadrantRect(left, top, thresholdX-left, thresholdY-top, '#0FBF3E') + quadrantRect(thresholdX, top, left+plotW-thresholdX, thresholdY-top, '#DB9D00') + quadrantRect(thresholdX, thresholdY, left+plotW-thresholdX, top+plotH-thresholdY, '#FE4C25') + quadrantRect(left, thresholdY, thresholdX-left, top+plotH-thresholdY, '#909692'); out += quadrantLabel(left, top, thresholdX-left, thresholdY-top, 'Keep / scale', 'high impact, lower cost', '#8CF2A6') + quadrantLabel(thresholdX, top, left+plotW-thresholdX, thresholdY-top, 'Optimize', 'high impact, high cost', '#DB9D00') + quadrantLabel(thresholdX, thresholdY, left+plotW-thresholdX, top+plotH-thresholdY, 'Waste review', 'low impact, high cost', '#FE4C25') + quadrantLabel(left, thresholdY, thresholdX-left, top+plotH-thresholdY, 'Monitor', 'lower impact, lower cost', '#B6BFB8'); } for(let i=0;i<=4;i++){ const x = left + plotW*i/4; const y = top + plotH*i/4; out += '<line x1="'+x+'" y1="'+top+'" x2="'+x+'" y2="'+(top+plotH)+'" stroke="#3f4942"/><line x1="'+left+'" y1="'+y+'" x2="'+(left+plotW)+'" y2="'+y+'" stroke="#3f4942"/><text x="'+(left-12)+'" y="'+(y+5)+'" text-anchor="end" font-size="13" fill="#B6BFB8">'+fmt(maxImpact*(4-i)/4,0)+'</text>'; } for(const tick of logTicks(minCost, maxCost)){ const x = xScale(tick); out += '<text x="'+x+'" y="'+(top+plotH+25)+'" text-anchor="middle" font-size="13" fill="#B6BFB8">'+compactNumber(tick)+'</text>'; } if(threshold.cost > 0){ out += '<line x1="'+thresholdX+'" y1="'+top+'" x2="'+thresholdX+'" y2="'+(top+plotH)+'" stroke="#F2F5F3" stroke-dasharray="6 6" opacity="0.62"/>'; } if(threshold.impact > 0){ out += '<line x1="'+left+'" y1="'+thresholdY+'" x2="'+(left+plotW)+'" y2="'+thresholdY+'" stroke="#F2F5F3" stroke-dasharray="6 6" opacity="0.62"/>'; } for(const row of workflowRows){ const x = costValue(row) > 0 ? xScale(costValue(row)) : left + 10; const y = yScale(row.attributed_impact_score); const radius = Math.min(28, 8 + Math.sqrt(Math.max(row.linked_items,1))*3.1); const color = zones[row.display_zone] || '#909692'; out += '<circle class="workflow-mark workflow-point" data-workflow="'+attr(row.workflow)+'" cx="'+x+'" cy="'+y+'" r="'+radius+'" fill="'+color+'" fill-opacity="0.88" stroke="#101411" stroke-width="2"><title>'+esc(row.workflow)+'\nzone: '+esc(row.display_zone)+'\nimpact score: '+fmt(row.attributed_impact_score)+'\n'+costTooltip(row)+'\nlinked items: '+row.linked_items+'</title></circle>'; } const needsCost = workflowRows.filter(row => row.attributed_impact_score > 0 && costValue(row) <= 0).length; if(needsCost){ out += '<text x="'+(left+14)+'" y="'+(top+plotH-14)+'" font-size="11" font-weight="800" fill="#3094FF">Needs cost: '+needsCost+' workflows with impact but no joined cost</text>'; } out += '<text x="'+(left+plotW/2)+'" y="'+(top+plotH+54)+'" text-anchor="middle" font-size="14" font-weight="700" fill="#E4EBE6">'+esc(costLabel())+'</text><text x="28" y="'+(top+plotH/2)+'" text-anchor="middle" transform="rotate(-90 28 '+(top+plotH/2)+')" font-size="14" font-weight="700" fill="#E4EBE6">Impact score</text>'; svg.innerHTML = out; }
function quadrantRect(x,y,width,height,color){ if(width <= 0 || height <= 0){ return ''; } return '<rect x="'+x+'" y="'+y+'" width="'+width+'" height="'+height+'" fill="'+color+'" fill-opacity="0.08"/>'; }
function quadrantLabel(x,y,width,height,title,subtitle,color){ if(width < 140 || height < 56){ return ''; } const labelX = x + 12; const labelY = y + 22; return '<text x="'+labelX+'" y="'+labelY+'" font-size="12" font-weight="800" fill="'+color+'">'+esc(title)+'</text><text x="'+labelX+'" y="'+(labelY+15)+'" font-size="10" font-weight="700" fill="#B6BFB8">'+esc(subtitle)+'</text>'; }
function renderRankChart(){ const ranked = rows().filter(row => row.attributed_impact_score > 0 || row.linked_items > 0).sort((a,b)=>b.attributed_impact_score-a.attributed_impact_score || b.linked_items-a.linked_items); let chartRows = ranked.slice(0,14); if(hoveredWorkflow && !chartRows.some(row => row.workflow === hoveredWorkflow)){ const hovered = ranked.find(row => row.workflow === hoveredWorkflow); if(hovered){ chartRows = chartRows.slice(0,13).concat(hovered); } } const svg = document.getElementById('impactRankChart'); const width = 800, height = 650, rowH = 38, left = 270, top = 54, plotW = 455; svg.setAttribute('viewBox', '0 0 '+width+' '+height); const maxImpact = Math.max(1, ...chartRows.map(row => row.attributed_impact_score)); let out = '<rect width="100%%" height="100%%" fill="#232925"/>'; chartRows.forEach((row,index) => { const y = top + index*rowH; const w = Math.max(4, row.attributed_impact_score/maxImpact*plotW); const color = zones[row.display_zone] || '#909692'; out += '<text x="'+(left-16)+'" y="'+(y+21)+'" text-anchor="end" font-size="15" font-weight="700" fill="#E4EBE6">'+esc(shortText(row.workflow,25))+'</text><rect x="'+left+'" y="'+(y+2)+'" width="'+plotW+'" height="24" rx="6" fill="#313a34" opacity="0.78"></rect><rect class="workflow-mark workflow-bar" data-workflow="'+attr(row.workflow)+'" x="'+left+'" y="'+(y+2)+'" width="'+w+'" height="24" rx="6" fill="'+color+'" opacity="0.94"><title>'+esc(row.workflow)+'\nzone: '+esc(row.display_zone)+'\nimpact score: '+fmt(row.attributed_impact_score)+'\n'+costTooltip(row)+'\nlinked items: '+row.linked_items+'</title></rect><text x="'+(left+w+10)+'" y="'+(y+21)+'" font-size="14" font-weight="800" fill="#F2F5F3">'+fmt(row.attributed_impact_score,1)+'</text>'; }); svg.innerHTML = out; applyWorkflowHighlight(); }
function scoredWorkItems(){ return [...(payload.work_items || [])].sort((a,b)=>b.last_impact_score-a.last_impact_score || a.number-b.number); }
function renderScoredWorkItems(){ renderItemTable('workflowWorkItems', 'workflowWorkFilter', scoredWorkItems().filter(item => (item.source_workflows || []).length)); }
function renderOtherWorkItems(){ renderItemTable('otherWorkItems', 'otherWorkFilter', scoredWorkItems().filter(item => !(item.source_workflows || []).length)); }
function bindItemFilter(filterId){ const input = document.getElementById(filterId); if(input && !input.dataset.bound){ input.addEventListener('input', () => { renderScoredWorkItems(); renderOtherWorkItems(); }); input.dataset.bound = 'true'; } return input; }
function renderItemTable(containerId, filterId, items){ const container = document.getElementById(containerId); if(!container){ return; } const input = bindItemFilter(filterId); const query = String(input?.value || '').trim().toLowerCase(); const filtered = query ? items.filter(item => itemSearchText(item).includes(query)) : items; if(!filtered.length){ container.innerHTML = '<div class="muted" style="padding:10px">No work item impact data available.</div>'; return; } container.innerHTML = '<table><thead><tr><th style="width:72px">Score</th><th>Work item</th><th style="width:120px">State</th><th>Scoring</th><th>Agentic workflow</th></tr></thead><tbody>' + filtered.map(item => itemTableRow(item)).join('') + '</tbody></table>'; }
function itemStateText(item){ return [item.item_type, item.state, item.state_reason].filter(Boolean).join(' / '); }
function itemTableRow(item){ const source = [item.score_source || 'score', scoreExplanationText(item)].filter(Boolean).join(' / '); const workflows = (item.source_workflows || []).join(', ') || 'no linked agentic workflow'; return '<tr><td><span class="impact-score">' + fmt(item.last_impact_score, 1) + '</span></td><td><a href="' + attr(item.html_url || githubItemURL(item)) + '" target="_blank" rel="noopener noreferrer">' + esc('#' + item.number + ' ' + shortText(item.title, 96)) + '</a></td><td>' + esc(itemStateText(item)) + '</td><td>' + esc(source) + '</td><td>' + esc(workflows) + '</td></tr>'; }
function itemSearchText(item){ return [item.number, item.item_type, item.state, item.state_reason, item.title, item.score_source, scoreExplanationText(item), ...(item.source_workflows || [])].join(' ').toLowerCase(); }
function scoreExplanationText(item){ const explanation = item.score_explanation || {}; const parts = []; if(explanation.policy_path){ let policy = explanation.policy_path; if(explanation.policy_version){ policy += '@v' + explanation.policy_version; } if(explanation.policy_sha256){ policy += '#' + explanation.policy_sha256.slice(0, 12); } parts.push(policy); } if((explanation.matched_rules || []).length){ parts.push('rules: ' + explanation.matched_rules.slice(0, 3).join(', ')); } return parts.join(' / '); }
function logTicks(minValue,maxImpact){ const ticks=[]; const start=Math.floor(Math.log10(minValue)); const end=Math.ceil(Math.log10(maxImpact)); for(let exponent=start; exponent<=end; exponent++){ for(const multiplier of [1,2,5]){ const value=multiplier*Math.pow(10, exponent); if(value>=minValue && value<=maxImpact){ ticks.push(value); } } } return ticks; }
function compactNumber(value){ if(value >= 1000){ return fmt(value/1000,0) + 'k'; } if(value >= 10){ return fmt(value,0); } if(value >= 1){ return fmt(value,1); } return fmt(value,2); }
function shortText(value, limit){ value = String(value || ''); return value.length <= limit ? value : value.slice(0, limit-3) + '...'; }
function githubItemURL(item){ const path = item.item_type === 'pr' ? 'pull' : 'issues'; return payload.repo ? 'https://github.com/' + payload.repo + '/' + path + '/' + item.number : '#'; }
function setWorkflowHover(workflow){ if(!workflow){ return; } const changed = hoveredWorkflow !== workflow; hoveredWorkflow = workflow; if(changed){ renderRankChart(); } applyWorkflowHighlight(); }
function clearWorkflowHover(){ if(!hoveredWorkflow){ return; } hoveredWorkflow = ''; renderRankChart(); applyWorkflowHighlight(); }
function applyWorkflowHighlight(){ const marks = Array.from(document.querySelectorAll('#workflows [data-workflow]')); marks.forEach(mark => { const active = hoveredWorkflow && mark.dataset.workflow === hoveredWorkflow; mark.classList.toggle('workflow-highlight', active); mark.classList.toggle('workflow-dim', Boolean(hoveredWorkflow) && !active); }); }
function switchTab(tab){ document.querySelectorAll('.tabs button,.panel').forEach(el => el.classList.remove('active')); document.querySelector('[data-tab="' + tab + '"]').classList.add('active'); document.getElementById(tab).classList.add('active'); }
setup();
</script>
</body>
</html>`, string(data))
	return os.WriteFile(filepath.Join(outDir, "impact_score_dashboard.html"), html.Bytes(), 0o644)
}

func uiPayloadFromOutput(result output) uiPayload {
	payload := uiPayload{Repo: result.Repo, GeneratedAt: result.GeneratedAt}
	for _, rank := range result.WorkflowRanks {
		payload.WorkflowRanks = append(payload.WorkflowRanks, workflowRankForUI{Workflow: rank.Workflow, AttributedImpactScore: rank.AttributedImpactScore, LinkedItems: rank.LinkedItems, TotalAICCost: rank.TotalAICCost, ActionMinutes: rank.ActionMinutes})
	}
	payload.WorkItems = workItemsForUI(result.ItemRanks, result.Features, result.Repo)
	return payload
}

func workItemsForUI(ranks []impactscore.ItemRank, features []impactscore.ItemFeatures, defaultRepo string) []workItemForUI {
	featuresByItem := map[string]impactscore.ItemFeatures{}
	for _, feature := range features {
		featuresByItem[workItemKey(feature.Item.Type, feature.Item.Number)] = feature
	}
	items := make([]workItemForUI, 0, len(ranks))
	for _, rank := range ranks {
		feature := featuresByItem[workItemKey(rank.ItemType, rank.Number)]
		repo := firstNonEmpty(feature.Item.Repo, defaultRepo)
		items = append(items, workItemForUI{Number: rank.Number, ItemType: rank.ItemType, State: rank.State, StateReason: rank.StateReason, Title: rank.Title, HTMLURL: githubWorkItemURL(repo, rank.ItemType, rank.Number), LastImpactScore: rank.ImpactScore, ScoreSource: rank.ScoreSource, ScoreExplanation: rank.ScoreExplanation, SourceWorkflows: rank.SourceWorkflows})
	}
	return items
}

func githubWorkItemURL(repo, itemType string, number int) string {
	if repo == "" || number == 0 {
		return ""
	}
	path := "issues"
	if itemType == "pr" {
		path = "pull"
	}
	return "https://github.com/" + repo + "/" + path + "/" + strconv.Itoa(number)
}
