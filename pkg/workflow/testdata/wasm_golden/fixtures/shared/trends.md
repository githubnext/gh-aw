---
# Trends Visualization Shared Workflow
# Provides guidance for creating trending data charts
#
# Usage:
#   imports:
#     - shared/trends.md
#
# This import provides:
# - Python data visualization environment (via python-dataviz import)
# - Prompts for generating awesome trending charts
# - Best practices for visualizing trends over time
# - Guidelines for creating engaging and informative trend visualizations

imports:
  - shared/python-dataviz.md
---

# Trends Visualization Guide

You are an expert at creating compelling trend visualizations that reveal insights from data over time.

## Trending Chart Best Practices

When generating trending charts, focus on:

### 1. **Time Series Excellence**
- Use line charts for continuous trends over time
- Add trend lines or moving averages to highlight patterns
- Include clear date/time labels on the x-axis
- Show confidence intervals or error bands when relevant

### 2. **Comparative Trends**
- Use multi-line charts to compare multiple trends
- Apply distinct colors for each series with a clear legend
- Consider using area charts for stacked trends
- Highlight key inflection points or anomalies

### 3. **Visual Impact**
- Use vibrant, contrasting colors to make trends stand out
- Add annotations for significant events or milestones
- Include grid lines for easier value reading
- Use appropriate scale (linear vs. logarithmic)

### 4. **Contextual Information**
- Show percentage changes or growth rates
- Include baseline comparisons (year-over-year, month-over-month)
- Add summary statistics (min, max, average, median)
- Highlight recent trends vs. historical patterns

## Example Trend Chart Types

### Temporal Trends
```python
# Line chart with multiple trends
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
for column in data.columns:
    ax.plot(data.index, data[column], marker='o', label=column, linewidth=2)
ax.set_title('Trends Over Time', fontsize=16, fontweight='bold')
ax.set_xlabel('Date', fontsize=12)
ax.set_ylabel('Value', fontsize=12)
ax.legend(loc='best')
ax.grid(True, alpha=0.3)
plt.xticks(rotation=45)
```

### Growth Rates
```python
# Bar chart showing period-over-period growth
fig, ax = plt.subplots(figsize=(10, 6), dpi=300)
growth_data.plot(kind='bar', ax=ax, color=sns.color_palette("husl"))
ax.set_title('Growth Rates by Period', fontsize=16, fontweight='bold')
ax.axhline(y=0, color='black', linestyle='-', linewidth=0.8)
ax.set_ylabel('Growth %', fontsize=12)
```

### Moving Averages
```python
# Trend with moving average overlay
fig, ax = plt.subplots(figsize=(12, 7), dpi=300)
ax.plot(dates, values, label='Actual', alpha=0.5, linewidth=1)
ax.plot(dates, moving_avg, label='7-day Moving Average', linewidth=2.5)
ax.fill_between(dates, values, moving_avg, alpha=0.2)
```

## Data Preparation for Trends

### Time-Based Indexing
```python
# Convert to datetime and set as index
data['date'] = pd.to_datetime(data['date'])
data.set_index('date', inplace=True)
data = data.sort_index()
```

### Resampling and Aggregation
```python
# Resample daily data to weekly
weekly_data = data.resample('W').mean()

# Calculate rolling statistics
data['rolling_mean'] = data['value'].rolling(window=7).mean()
data['rolling_std'] = data['value'].rolling(window=7).std()
```

### Growth Calculations
```python
# Calculate percentage change
data['pct_change'] = data['value'].pct_change() * 100

# Calculate year-over-year growth
data['yoy_growth'] = data['value'].pct_change(periods=365) * 100
```

## Color Palettes for Trends

Use these palettes for impactful trend visualizations:

- **Sequential trends**: `sns.color_palette("viridis", n_colors=5)`
- **Diverging trends**: `sns.color_palette("RdYlGn", n_colors=7)`
- **Multiple series**: `sns.color_palette("husl", n_colors=8)`
- **Categorical**: `sns.color_palette("Set2", n_colors=6)`

## Annotation Best Practices

```python
# Annotate key points
max_idx = data['value'].idxmax()
max_val = data['value'].max()
ax.annotate(f'Peak: {max_val:.2f}',
            xy=(max_idx, max_val),
            xytext=(10, 20),
            textcoords='offset points',
            arrowprops=dict(arrowstyle='->', color='red'),
            fontsize=10,
            fontweight='bold')
```

## Styling for Awesome Charts

```python
import matplotlib.pyplot as plt
import seaborn as sns

# Set professional style
sns.set_style("whitegrid")
sns.set_context("notebook", font_scale=1.2)

# Custom color palette
custom_colors = ["#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A", "#98D8C8"]
sns.set_palette(custom_colors)

# Figure with optimal dimensions
fig, ax = plt.subplots(figsize=(14, 8), dpi=300)

# ... your plotting code ...

# Tight layout for clean appearance
plt.tight_layout()

# Save with high quality
plt.savefig('/tmp/gh-aw/python/charts/trend_chart.png',
            dpi=300,
            bbox_inches='tight',
            facecolor='white',
            edgecolor='none')
```

## Tips for Trending Charts

1. **Start with the story**: What trend are you trying to show?
2. **Choose the right timeframe**: Match granularity to the pattern
3. **Smooth noise**: Use moving averages for volatile data
4. **Show context**: Include historical baselines or benchmarks
5. **Highlight insights**: Use annotations to draw attention
6. **Test readability**: Ensure labels and legends are clear
7. **Optimize colors**: Use colorblind-friendly palettes
8. **Export high quality**: Always use DPI 300+ for presentations

## Common Trend Patterns to Visualize

- **Seasonal patterns**: Monthly or quarterly cycles
- **Long-term growth**: Exponential or linear trends
- **Volatility changes**: Periods of stability vs. fluctuation
- **Correlations**: How multiple trends relate
- **Anomalies**: Outliers or unusual events
- **Forecasts**: Projected future trends with uncertainty

Remember: The best trending charts tell a clear story, make patterns obvious, and inspire action based on the insights revealed.
