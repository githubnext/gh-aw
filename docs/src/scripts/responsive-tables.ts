/**
 * Responsive Tables Enhancement
 * 
 * Adds data-label attributes to all table cells in markdown content
 * to enable CSS-only responsive card layout on mobile devices.
 * 
 * This script runs on page load and enhances standard markdown tables
 * with the data needed for mobile transformation.
 */

function enhanceResponsiveTables() {
  // Find all tables in markdown content
  const tables = document.querySelectorAll('.sl-markdown-content table');
  
  tables.forEach(table => {
    const headers: string[] = [];
    
    // Extract header text from thead
    const headerCells = table.querySelectorAll('thead th');
    headerCells.forEach(th => {
      headers.push(th.textContent?.trim() || '');
    });
    
    // Add data-label attribute to each td based on column position
    const rows = table.querySelectorAll('tbody tr');
    rows.forEach(row => {
      const cells = row.querySelectorAll('td');
      cells.forEach((cell, index) => {
        if (headers[index]) {
          cell.setAttribute('data-label', headers[index]);
        }
      });
    });
  });
}

// Run on initial page load
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', enhanceResponsiveTables);
} else {
  enhanceResponsiveTables();
}

// Re-run when Astro navigates to a new page (for SPA-like navigation)
document.addEventListener('astro:page-load', enhanceResponsiveTables);
