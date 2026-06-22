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

## Styling and Annotations

Use `sns.set_style("whitegrid")`, `sns.set_context("notebook", font_scale=1.2)`, `figsize=(14, 8)`, `dpi=300`, and `bbox_inches='tight'`. Annotate key peaks/troughs with `ax.annotate()` using `arrowprops`.

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
