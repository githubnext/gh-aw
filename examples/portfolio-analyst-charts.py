#!/usr/bin/env python3
"""
Example chart generation script for Portfolio Analyst workflow.
This demonstrates how to create the 4 required dashboard charts.
"""

import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
import os

# Configuration
SUMMARY_FILE = '/tmp/portfolio-logs/summary.json'
DATA_DIR = '/tmp/gh-aw/python/data'
CHARTS_DIR = '/tmp/gh-aw/python/charts'

# Ensure directories exist
os.makedirs(DATA_DIR, exist_ok=True)
os.makedirs(CHARTS_DIR, exist_ok=True)

# Set professional style
sns.set_style("whitegrid")
sns.set_palette("husl")

# Load summary data
print("Loading workflow data from summary.json...")
with open(SUMMARY_FILE, 'r') as f:
    data = json.load(f)

# Prepare dataframe
runs_df = pd.DataFrame(data['runs'])
runs_df['date'] = pd.to_datetime(runs_df['created_at']).dt.date
runs_df['cost'] = runs_df['estimated_cost']

# Chart 1: Cost Trends Over Time
print("Generating cost trends chart...")
daily_costs = runs_df.groupby('date')['cost'].sum().reset_index()
daily_costs['rolling_avg'] = daily_costs['cost'].rolling(window=7, min_periods=1).mean()

fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
ax.plot(daily_costs['date'], daily_costs['cost'], 
        label='Daily Cost', marker='o', alpha=0.6, linewidth=1.5)
ax.plot(daily_costs['date'], daily_costs['rolling_avg'], 
        label='7-day Moving Average', linewidth=2.5, color='#FF6B6B')
ax.fill_between(daily_costs['date'], 0, daily_costs['cost'], alpha=0.2)

ax.set_title('Workflow Costs - Last 30 Days', fontsize=16, fontweight='bold')
ax.set_xlabel('Date', fontsize=12)
ax.set_ylabel('Cost ($)', fontsize=12)
ax.legend(loc='best', fontsize=10)
ax.grid(True, alpha=0.3)
plt.xticks(rotation=45)
plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/cost_trends.png', dpi=300, bbox_inches='tight', facecolor='white')
plt.close()

# Chart 2: Top 10 Workflows by Cost
print("Generating top spenders chart...")
workflow_costs = runs_df.groupby('workflow_name')['cost'].sum().sort_values(ascending=True).tail(10)

fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
colors = sns.color_palette("RdYlGn_r", len(workflow_costs))
workflow_costs.plot(kind='barh', ax=ax, color=colors)

ax.set_title('Top 10 Workflows by Total Cost', fontsize=16, fontweight='bold')
ax.set_xlabel('Total Cost ($)', fontsize=12)
ax.set_ylabel('Workflow', fontsize=12)
ax.grid(True, alpha=0.3, axis='x')

# Add cost labels on bars
for i, (idx, value) in enumerate(workflow_costs.items()):
    ax.text(value, i, f' ${value:.2f}', va='center', fontsize=9, fontweight='bold')

plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/top_spenders.png', dpi=300, bbox_inches='tight', facecolor='white')
plt.close()

# Chart 3: Workflows with High Failure Rates
print("Generating failure rates chart...")
workflow_stats = runs_df.groupby('workflow_name').agg({
    'conclusion': 'count',
    'cost': 'sum'
}).rename(columns={'conclusion': 'total_runs'})

failure_counts = runs_df[runs_df['conclusion'] == 'failure'].groupby('workflow_name').size()
workflow_stats['failures'] = failure_counts
workflow_stats['failures'].fillna(0, inplace=True)
workflow_stats['failure_rate'] = (workflow_stats['failures'] / workflow_stats['total_runs']) * 100
workflow_stats['wasted_cost'] = runs_df[runs_df['conclusion'] == 'failure'].groupby('workflow_name')['cost'].sum()
workflow_stats['wasted_cost'].fillna(0, inplace=True)

# Filter workflows with >30% failure rate
high_failure = workflow_stats[workflow_stats['failure_rate'] > 30].sort_values('wasted_cost', ascending=True).tail(10)

if len(high_failure) > 0:
    fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
    
    # Create bars with color gradient based on failure rate
    colors = plt.cm.Reds(high_failure['failure_rate'] / 100)
    high_failure['failure_rate'].plot(kind='barh', ax=ax, color=colors)
    
    ax.set_title('Workflows with High Failure Rates (>30%)', fontsize=16, fontweight='bold')
    ax.set_xlabel('Failure Rate (%)', fontsize=12)
    ax.set_ylabel('Workflow', fontsize=12)
    ax.grid(True, alpha=0.3, axis='x')
    
    # Add labels with failure rate and wasted cost
    for i, (idx, row) in enumerate(high_failure.iterrows()):
        ax.text(row['failure_rate'], i, 
                f" {row['failure_rate']:.1f}% (${row['wasted_cost']:.2f} wasted)", 
                va='center', fontsize=9)
    
    plt.tight_layout()
    plt.savefig(f'{CHARTS_DIR}/failure_rates.png', dpi=300, bbox_inches='tight', facecolor='white')
    plt.close()
else:
    print("No workflows with >30% failure rate found - skipping chart")

# Chart 4: Overall Success Rate Distribution
print("Generating success overview chart...")
conclusion_counts = runs_df['conclusion'].value_counts()

fig, ax = plt.subplots(figsize=(10, 7), dpi=300)
colors_map = {
    'success': '#4ECDC4',
    'failure': '#FF6B6B',
    'cancelled': '#FFA07A'
}
colors = [colors_map.get(c, '#95A5A6') for c in conclusion_counts.index]

wedges, texts, autotexts = ax.pie(conclusion_counts.values, 
                                    labels=conclusion_counts.index, 
                                    autopct='%1.1f%%',
                                    colors=colors,
                                    startangle=90,
                                    textprops={'fontsize': 12, 'fontweight': 'bold'})

# Add count to labels
for i, (label, count) in enumerate(zip(conclusion_counts.index, conclusion_counts.values)):
    texts[i].set_text(f'{label.title()}\n({count} runs)')

ax.set_title('Workflow Run Status Distribution', fontsize=16, fontweight='bold', pad=20)
plt.tight_layout()
plt.savefig(f'{CHARTS_DIR}/success_overview.png', dpi=300, bbox_inches='tight', facecolor='white')
plt.close()

print("\n✅ All 4 charts generated successfully!")
print(f"📊 Charts saved to: {CHARTS_DIR}/")
print("   - cost_trends.png")
print("   - top_spenders.png")
print("   - failure_rates.png")
print("   - success_overview.png")
print("\nNext steps:")
print("1. Upload each chart using 'upload asset' tool")
print("2. Use returned URLs to embed in the dashboard report")
