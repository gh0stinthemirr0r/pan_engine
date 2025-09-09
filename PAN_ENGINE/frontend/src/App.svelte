<script>
  import { onMount } from 'svelte';
  import { 
    Greet, 
    GenerateReport,
    SaveAPISettings,
    GetAPISettings,
    TestAPIConnection,
    ExportToCSV,
    ExportToPDF,
    ListReports,
    OpenReport,
    DeleteReport,
    GetSupportedReportTypes,
    GetReportCategories,
    BatchExportReports,
    FilterReportData,
    SearchAllReports,
    GetReportHistory
  } from '../wailsjs/go/main/App';

  // Active view
  let activeView = 'generate';
  let previousView = '';
  
  // API settings
  let apiSettings = { url: '', key: '', status: 'unknown' };
  let showSettings = false;
  let apiConnectionStatus = null;
  let inputType = 'password';
  let isTestingConnection = false;
  
  // Report generation
  let reportType = '';
  let reportsByCategory = {};
  let reportCategories = [];
  let selectedCategory = 'All';
  let startDate = '';
  let endDate = '';
  let loading = false;
  let error = '';
  let responseData = null;
  let selectedReportFormat = 'json';
  let reportData = null;
  
  // Batch export
  let selectedReports = [];
  let batchFormat = 'pdf';
  let batchStartDate = '';
  let batchEndDate = '';
  let batchResults = null;
  
  // Report management
  let reports = [];
  let loadingReports = false;
  
  // Filtering and search
  let searchTerm = '';
  let filters = {};
  let searchResults = null;
  let filteredData = null;
  
  // Configuration
  let maxRows = 1000;
  let reportFormat = 'standard';
  
  // Initialize application
  onMount(async () => {
    try {
      // Get report categories
      reportCategories = await GetReportCategories();
      reportCategories.unshift('All');
      
      // Get report types
      const reportTypes = await GetSupportedReportTypes();
      
      // Sort reports by category
      reportsByCategory = { 'All': [] };
      
      for (const report of reportTypes) {
        if (!reportsByCategory[report.category]) {
          reportsByCategory[report.category] = [];
        }
        
        reportsByCategory[report.category].push(report);
        reportsByCategory['All'].push(report);
      }
      
      // Get API settings
      const settings = await GetAPISettings();
      apiSettings = settings;
      
      // Load reports
      await loadReports();
      
    } catch (err) {
      error = `Error initializing application: ${err.message}`;
    }
  });
  
  // Navigation functions
  function navigateTo(view) {
    previousView = activeView;
    activeView = view;
    error = '';
  }
  
  function goBack() {
    if (previousView) {
      activeView = previousView;
      previousView = '';
    } else {
      activeView = 'generate';
    }
    error = '';
  }
  
  // API settings functions
  async function saveSettings() {
    try {
      const result = await SaveAPISettings(apiSettings.url, apiSettings.key);
      showSettings = false;
      await testConnection();
    } catch (e) {
      error = e.message || 'An error occurred while saving API settings';
    }
  }
  
  async function testConnection() {
    isTestingConnection = true;
    apiConnectionStatus = null;
    
    try {
      const result = await TestAPIConnection();
      apiConnectionStatus = result;
      apiSettings.status = result.status;
    } catch (e) {
      apiConnectionStatus = {
        status: 'error',
        message: e.message || 'Connection test failed'
      };
    } finally {
      isTestingConnection = false;
    }
  }
  
  function togglePasswordVisibility() {
    inputType = inputType === 'password' ? 'text' : 'password';
  }
  
  // Report generation functions
  async function generateReport() {
    if (!reportType) {
      error = 'Please select a report type';
      return;
    }
    
    loading = true;
    error = '';
    responseData = null;
    
    try {
      // Get dates if we have them (optional for some reports)
      const start = startDate || '';
      const end = endDate || '';
      
      // Generate the report
      responseData = await GenerateReport(reportType, start, end);
      
      // Handle export if requested
      if (selectedReportFormat !== 'json') {
        await exportReport();
      }
    } catch (e) {
      error = e.message || 'An error occurred while generating the report';
    } finally {
      loading = false;
    }
  }
  
  async function exportReport() {
    if (!reportType) {
      error = 'No report data available to export';
      return;
    }
    
    try {
      let exportPath = '';
      
      if (selectedReportFormat === 'csv') {
        exportPath = await ExportToCSV(reportType);
      } else if (selectedReportFormat === 'pdf') {
        exportPath = await ExportToPDF(reportType);
      }
      
      if (exportPath) {
        // Add success message
        exportSuccess = `Report exported successfully to ${exportPath}`;
        
        // Reload reports list
        await loadReports();
      }
    } catch (e) {
      error = e.message || 'An error occurred while exporting the report';
    }
  }
  
  async function loadReports() {
    loadingReports = true;
    
    try {
      reports = await ListReports();
      // Sort by date, newest first
      reports.sort((a, b) => new Date(b.modified) - new Date(a.modified));
    } catch (e) {
      error = e.message || 'An error occurred while loading reports';
    } finally {
      loadingReports = false;
    }
  }
  
  async function openReport(path) {
    try {
      await OpenReport(path);
    } catch (e) {
      error = e.message || 'An error occurred while opening the report';
    }
  }
  
  async function deleteReport(path) {
    if (!confirm('Are you sure you want to delete this report?')) {
      return;
    }
    
    try {
      await DeleteReport(path);
      await loadReports();
    } catch (e) {
      error = e.message || 'An error occurred while deleting the report';
    }
  }
  
  // Batch export functions
  async function executeBatchExport() {
    if (selectedReports.length === 0) {
      error = 'Please select at least one report type';
      return;
    }
    
    loading = true;
    error = '';
    batchResults = null;
    
    try {
      batchResults = await BatchExportReports(
        selectedReports,
        batchFormat,
        batchStartDate || '',
        batchEndDate || ''
      );
      
      // Reload reports after batch export
      await loadReports();
    } catch (e) {
      error = e.message || 'An error occurred during batch export';
    } finally {
      loading = false;
    }
  }
  
  function toggleReportSelection(reportValue) {
    const index = selectedReports.indexOf(reportValue);
    if (index === -1) {
      selectedReports = [...selectedReports, reportValue];
    } else {
      selectedReports = selectedReports.filter(v => v !== reportValue);
    }
  }
  
  function selectAllReportsInCategory(category) {
    const categoryReports = reportsByCategory[category] || [];
    const categoryValues = categoryReports.map(r => r.value);
    
    // If all reports in this category are already selected, deselect them
    const allSelected = categoryValues.every(v => selectedReports.includes(v));
    
    if (allSelected) {
      selectedReports = selectedReports.filter(v => !categoryValues.includes(v));
    } else {
      // Add all reports from this category that aren't already selected
      const newSelections = categoryValues.filter(v => !selectedReports.includes(v));
      selectedReports = [...selectedReports, ...newSelections];
    }
  }
  
  // Search and filter functions
  async function searchReports() {
    if (!searchTerm) {
      error = 'Please enter a search term';
      return;
    }
    
    loading = true;
    error = '';
    searchResults = null;
    
    try {
      searchResults = await SearchAllReports(searchTerm);
    } catch (e) {
      error = e.message || 'An error occurred while searching reports';
    } finally {
      loading = false;
    }
  }
  
  async function applyFilters() {
    if (!reportType) {
      error = 'No report selected for filtering';
      return;
    }
    
    // Remove empty filters
    const nonEmptyFilters = {};
    for (const [key, value] of Object.entries(filters)) {
      if (value.trim() !== '') {
        nonEmptyFilters[key] = value.trim();
      }
    }
    
    if (Object.keys(nonEmptyFilters).length === 0) {
      error = 'No filters specified';
      return;
    }
    
    loading = true;
    error = '';
    filteredData = null;
    
    try {
      filteredData = await FilterReportData(reportType, nonEmptyFilters);
    } catch (e) {
      error = e.message || 'An error occurred while filtering report data';
    } finally {
      loading = false;
    }
  }
  
  // Utility functions
  function formatDate(dateStr) {
    const date = new Date(dateStr);
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(date);
  }
  
  function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }
  
  let exportSuccess = '';
</script>

<main>
  <header>
    <h1>PAN_ENGINE</h1>
    <nav>
      <ul>
        <li class:active={activeView === 'generate'}>
          <button on:click={() => navigateTo('generate')}>Generate Report</button>
        </li>
        <li class:active={activeView === 'batch'}>
          <button on:click={() => navigateTo('batch')}>Batch Export</button>
        </li>
        <li class:active={activeView === 'reports'}>
          <button on:click={() => navigateTo('reports')}>View Reports</button>
        </li>
        <li class:active={activeView === 'search'}>
          <button on:click={() => navigateTo('search')}>Search</button>
        </li>
      </ul>
    </nav>
    <div class="api-status">
      <div class="status-indicator {apiSettings.status}"></div>
      <button on:click={() => { showSettings = true; }}>API Settings</button>
    </div>
  </header>

  <div class="content">
    {#if error}
      <div class="error">
        <span class="close-error" on:click={() => error = ''}>×</span>
        {error}
      </div>
    {/if}
    
    {#if exportSuccess}
      <div class="success">
        <span class="close-success" on:click={() => exportSuccess = ''}>×</span>
        {exportSuccess}
      </div>
    {/if}

    <!-- Generate Report View -->
    {#if activeView === 'generate'}
      <div class="panel">
        <h2>Generate Report</h2>
        
        <form on:submit|preventDefault={generateReport}>
          <div class="grid-form">
            <div class="form-group">
              <label for="category">Category:</label>
              <select id="category" bind:value={selectedCategory}>
                {#each reportCategories as category}
                  <option value={category}>{category}</option>
                {/each}
              </select>
            </div>
            
            <div class="form-group">
              <label for="reportType">Report Type:</label>
              <select id="reportType" bind:value={reportType}>
                <option value="">Select a report type</option>
                {#if reportsByCategory[selectedCategory]}
                  {#each reportsByCategory[selectedCategory] as report}
                    <option value={report.value}>{report.label}</option>
                  {/each}
                {/if}
              </select>
            </div>
            
            <div class="form-group">
              <label for="startDate">Start Date:</label>
              <input type="date" id="startDate" bind:value={startDate} />
            </div>
            
            <div class="form-group">
              <label for="endDate">End Date:</label>
              <input type="date" id="endDate" bind:value={endDate} />
            </div>
            
            <div class="form-group">
              <label for="format">Output Format:</label>
              <select id="format" bind:value={selectedReportFormat}>
                <option value="json">JSON (View in App)</option>
                <option value="csv">CSV Export</option>
                <option value="pdf">PDF Export</option>
              </select>
            </div>
            
            <div class="form-actions">
              <button type="submit" disabled={loading} class="primary">
                {loading ? 'Generating...' : 'Generate Report'}
              </button>
            </div>
          </div>
        </form>
      </div>
      
      {#if responseData}
        <div class="panel results">
          <div class="results-header">
            <h3>Report Results</h3>
            <div class="export-buttons">
              <button on:click={() => { selectedReportFormat = 'csv'; exportReport(); }}>
                Export to CSV
              </button>
              <button on:click={() => { selectedReportFormat = 'pdf'; exportReport(); }}>
                Export to PDF
              </button>
            </div>
          </div>
          
          <div class="results-content">
            {#if Array.isArray(responseData)}
              <table>
                <thead>
                  {#if responseData.length > 0 && typeof responseData[0] === 'object'}
                    <tr>
                      {#each Object.keys(responseData[0]) as header}
                        <th>{header}</th>
                      {/each}
                    </tr>
                  {/if}
                </thead>
                <tbody>
                  {#each responseData as item, i}
                    {#if i < 100} <!-- Limit display rows -->
                      <tr>
                        {#each Object.values(item) as value}
                          <td>{typeof value === 'object' ? JSON.stringify(value) : value}</td>
                        {/each}
                      </tr>
                    {/if}
                  {/each}
                </tbody>
              </table>
              {#if responseData.length > 100}
                <div class="more-rows">
                  Showing 100 of {responseData.length} rows. Export to see all data.
                </div>
              {/if}
            {:else if typeof responseData === 'object'}
              <table>
                <tbody>
                  {#each Object.entries(responseData) as [key, value]}
                    <tr>
                      <th>{key}</th>
                      <td>{typeof value === 'object' ? JSON.stringify(value) : value}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {:else}
              <pre>{JSON.stringify(responseData, null, 2)}</pre>
            {/if}
          </div>
        </div>
      {/if}
    {/if}
    
    <!-- Batch Export View -->
    {#if activeView === 'batch'}
      <div class="panel">
        <h2>Batch Export</h2>
        
        <form on:submit|preventDefault={executeBatchExport}>
          <div class="form-group">
            <label>Select Reports to Export:</label>
            
            <div class="category-selection">
              {#each Object.entries(reportsByCategory).filter(([cat]) => cat !== 'All') as [category, reports]}
                <div class="category-container">
                  <div class="category-header">
                    <label class="category-name">
                      <input 
                        type="checkbox" 
                        on:change={() => selectAllReportsInCategory(category)}
                        checked={reports.every(r => selectedReports.includes(r.value))} 
                      />
                      {category}
                    </label>
                  </div>
                  
                  <div class="report-items">
                    {#each reports as report}
                      <label class="report-item">
                        <input 
                          type="checkbox" 
                          checked={selectedReports.includes(report.value)} 
                          on:change={() => toggleReportSelection(report.value)}
                        />
                        {report.label}
                      </label>
                    {/each}
                  </div>
                </div>
              {/each}
            </div>
          </div>
          
          <div class="grid-form">
            <div class="form-group">
              <label for="batchFormat">Export Format:</label>
              <select id="batchFormat" bind:value={batchFormat}>
                <option value="pdf">PDF</option>
                <option value="csv">CSV</option>
              </select>
            </div>
            
            <div class="form-group">
              <label for="batchStartDate">Start Date:</label>
              <input type="date" id="batchStartDate" bind:value={batchStartDate} />
            </div>
            
            <div class="form-group">
              <label for="batchEndDate">End Date:</label>
              <input type="date" id="batchEndDate" bind:value={batchEndDate} />
            </div>
            
            <div class="form-actions">
              <button type="submit" disabled={loading || selectedReports.length === 0} class="primary">
                {loading ? 'Exporting...' : 'Export Reports'}
              </button>
            </div>
          </div>
        </form>
        
        {#if batchResults}
          <div class="batch-results">
            <h3>Batch Export Results</h3>
            <div class="summary">
              <p>Successfully exported {batchResults._summary.successful} of {batchResults._summary.total} reports.</p>
              {#if batchResults._summary.failed > 0}
                <p class="batch-error">{batchResults._summary.failed} reports failed to export.</p>
              {/if}
            </div>
            
            <table>
              <thead>
                <tr>
                  <th>Report</th>
                  <th>Status</th>
                  <th>File</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {#each Object.entries(batchResults).filter(([key]) => key !== '_summary') as [reportType, result]}
                  <tr class={result.success ? 'success' : 'error'}>
                    <td>{reportType}</td>
                    <td>{result.success ? 'Success' : 'Failed'}</td>
                    <td>{result.success ? result.path : result.error}</td>
                    <td>
                      {#if result.success}
                        <button on:click={() => openReport(result.path)}>Open</button>
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    {/if}
    
    <!-- View Reports View -->
    {#if activeView === 'reports'}
      <div class="panel">
        <h2>Reports</h2>
        
        <div class="reports-controls">
          <button on:click={loadReports} disabled={loadingReports}>
            {loadingReports ? 'Loading...' : 'Refresh'}
          </button>
        </div>
        
        {#if reports.length > 0}
          <table class="reports-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Size</th>
                <th>Date Modified</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each reports as report}
                <tr>
                  <td>{report.name}</td>
                  <td>{report.type.toUpperCase()}</td>
                  <td>{formatBytes(parseInt(report.size))}</td>
                  <td>{formatDate(report.modified)}</td>
                  <td class="actions">
                    <button on:click={() => openReport(report.path)}>Open</button>
                    <button class="delete" on:click={() => deleteReport(report.path)}>Delete</button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {:else if loadingReports}
          <div class="loading-message">Loading reports...</div>
        {:else}
          <div class="empty-message">No reports found. Generate some reports first.</div>
        {/if}
      </div>
    {/if}
    
    <!-- Search View -->
    {#if activeView === 'search'}
      <div class="panel">
        <h2>Search Reports</h2>
        
        <form on:submit|preventDefault={searchReports}>
          <div class="search-form">
            <div class="form-group">
              <label for="searchTerm">Search Term:</label>
              <input type="text" id="searchTerm" bind:value={searchTerm} placeholder="Enter search term..." />
            </div>
            
            <div class="form-actions">
              <button type="submit" disabled={loading || !searchTerm} class="primary">
                {loading ? 'Searching...' : 'Search'}
              </button>
            </div>
          </div>
        </form>
        
        {#if searchResults}
          <div class="search-results">
            <h3>Search Results</h3>
            
            {#if searchResults.found}
              <p>Found matches in {searchResults.reports_with_matches} of {searchResults.reports_searched} reports.</p>
              
              <div class="accordion-results">
                {#each Object.entries(searchResults.results) as [reportType, result]}
                  <div class="accordion-item">
                    <div class="accordion-header">
                      <h4>{reportType} ({result.count} matches)</h4>
                      <button>View</button>
                    </div>
                    <div class="accordion-content">
                      {#if Array.isArray(result.matches)}
                        <table>
                          <thead>
                            {#if result.matches.length > 0 && typeof result.matches[0] === 'object'}
                              <tr>
                                {#each Object.keys(result.matches[0]) as header}
                                  <th>{header}</th>
                                {/each}
                              </tr>
                            {/if}
                          </thead>
                          <tbody>
                            {#each result.matches as item, i}
                              {#if i < 10} <!-- Limit display rows -->
                                <tr>
                                  {#each Object.values(item) as value}
                                    <td>{typeof value === 'object' ? JSON.stringify(value) : value}</td>
                                  {/each}
                                </tr>
                              {/if}
                            {/each}
                          </tbody>
                        </table>
                        {#if result.matches.length > 10}
                          <div class="more-rows">
                            Showing 10 of {result.matches.length} matches.
                          </div>
                        {/if}
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            {:else}
              <p>No matches found in {searchResults.reports_searched} reports.</p>
            {/if}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  {#if showSettings}
    <div class="modal">
      <div class="modal-content">
        <h2>API Settings</h2>
        
        <div class="form-group">
          <label for="apiUrl">API URL:</label>
          <input 
            type="text" 
            id="apiUrl" 
            bind:value={apiSettings.url} 
            placeholder="https://firewall.example.com" 
          />
        </div>

        <div class="form-group">
          <label for="apiKey">API Key:</label>
          <div class="password-input">
            <input 
              type="password"
              id="apiKey" 
              bind:value={apiSettings.key} 
              placeholder="Enter your API key" 
            />
          </div>
        </div>
        
        {#if apiConnectionStatus}
          <div class="connection-status {apiConnectionStatus.status}">
            <p>{apiConnectionStatus.message}</p>
          </div>
        {/if}

        <div class="modal-buttons">
          <button on:click={testConnection} disabled={isTestingConnection}>
            {isTestingConnection ? 'Testing...' : 'Test Connection'}
          </button>
          <button class="primary" on:click={saveSettings}>Save</button>
          <button class="cancel" on:click={() => showSettings = false}>Cancel</button>
        </div>
      </div>
    </div>
  {/if}

  <footer>
    <p>© Aaron Stovall</p>
  </footer>
</main>

<style>
  :global(body) {
    margin: 0;
    font-family: 'Nunito', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
    background-color: #1a1a1a;
    color: #e0e0e0;
  }

  main {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }

  header {
    background-color: #2a2a2a;
    padding: 0.75rem 1.5rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
    border-bottom: 2px solid #a8ff00;
  }

  header h1 {
    color: #a8ff00;
    margin: 0;
    font-size: 1.75rem;
    text-shadow: 0 0 10px rgba(168, 255, 0, 0.5);
  }

  nav {
    flex: 1;
    display: flex;
    justify-content: center;
  }

  nav ul {
    display: flex;
    list-style: none;
    margin: 0;
    padding: 0;
    gap: 0.5rem;
  }

  nav li {
    margin: 0;
  }

  nav button {
    background: transparent;
    color: #e0e0e0;
    border: none;
    padding: 0.5rem 1rem;
    cursor: pointer;
    font-size: 0.9rem;
    border-radius: 4px;
    transition: all 0.2s;
  }

  nav li.active button {
    background-color: rgba(168, 255, 0, 0.2);
    color: #a8ff00;
    font-weight: bold;
  }

  nav button:hover {
    background-color: rgba(168, 255, 0, 0.1);
    color: #a8ff00;
  }

  .api-status {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .status-indicator {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background-color: #666;
  }

  .status-indicator.connected {
    background-color: #a8ff00;
    box-shadow: 0 0 5px rgba(168, 255, 0, 0.8);
  }

  .status-indicator.error {
    background-color: #ff4444;
    box-shadow: 0 0 5px rgba(255, 68, 68, 0.8);
  }

  .status-indicator.unknown {
    background-color: #ffaa00;
    box-shadow: 0 0 5px rgba(255, 170, 0, 0.8);
  }

  .content {
    flex: 1;
    padding: 1.5rem;
    max-width: 1200px;
    margin: 0 auto;
    width: 100%;
    box-sizing: border-box;
    overflow-y: auto;
  }

  .panel {
    background: #2a2a2a;
    padding: 1.5rem;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
    border: 1px solid #3a3a3a;
    margin-bottom: 1.5rem;
  }

  .panel h2 {
    margin-top: 0;
    color: #a8ff00;
    border-bottom: 1px solid #3a3a3a;
    padding-bottom: 0.75rem;
    margin-bottom: 1.25rem;
  }

  .form-group {
    margin-bottom: 1.25rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    color: #a8ff00;
    font-weight: bold;
  }

  input, select {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #3a3a3a;
    border-radius: 4px;
    font-size: 1rem;
    background-color: #1a1a1a;
    color: #e0e0e0;
    font-family: inherit;
  }

  input:focus, select:focus {
    outline: none;
    border-color: #a8ff00;
    box-shadow: 0 0 5px rgba(168, 255, 0, 0.3);
  }

  button {
    background-color: #3a3a3a;
    color: #e0e0e0;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: bold;
    transition: all 0.2s;
    font-family: inherit;
  }

  button.primary {
    background-color: #a8ff00;
    color: #1a1a1a;
  }

  button:hover {
    background-color: #4a4a4a;
  }

  button.primary:hover {
    background-color: #c4ff40;
    box-shadow: 0 0 10px rgba(168, 255, 0, 0.5);
  }

  button:disabled {
    background-color: #3a3a3a;
    color: #666;
    cursor: not-allowed;
    box-shadow: none;
  }

  .grid-form {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    grid-gap: 1rem;
  }

  .form-actions {
    grid-column: 1 / -1;
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    margin-top: 0.5rem;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 1rem;
    font-size: 0.9rem;
  }

  th, td {
    padding: 0.6rem 0.75rem;
    text-align: left;
    border-bottom: 1px solid #3a3a3a;
  }

  th {
    background-color: #1a1a1a;
    font-weight: bold;
    color: #a8ff00;
  }

  tbody tr:hover {
    background-color: rgba(168, 255, 0, 0.05);
  }

  .category-selection {
    max-height: 400px;
    overflow-y: auto;
    padding: 0.5rem;
    background-color: #1a1a1a;
    border-radius: 4px;
    border: 1px solid #3a3a3a;
  }

  .category-container {
    margin-bottom: 1rem;
    border-bottom: 1px solid #3a3a3a;
    padding-bottom: 0.75rem;
  }

  .category-container:last-child {
    margin-bottom: 0;
    border-bottom: none;
  }

  .category-header {
    margin-bottom: 0.5rem;
  }

  .category-name {
    font-weight: bold;
    color: #a8ff00;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .report-items {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 0.5rem;
    padding-left: 1.5rem;
  }

  .report-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: normal;
    color: #e0e0e0;
  }

  input[type="checkbox"] {
    width: auto;
  }

  .error {
    color: #ff4444;
    margin: 0 0 1.5rem 0;
    padding: 1rem;
    background-color: rgba(255, 68, 68, 0.1);
    border-radius: 4px;
    border: 1px solid rgba(255, 68, 68, 0.3);
  }

  .success {
    color: #a8ff00;
    margin: 0 0 1.5rem 0;
    padding: 1rem;
    background-color: rgba(168, 255, 0, 0.1);
    border-radius: 4px;
    border: 1px solid rgba(168, 255, 0, 0.3);
  }

  .modal {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.8);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
  }

  .modal-content {
    background: #2a2a2a;
    padding: 2rem;
    border-radius: 8px;
    width: 90%;
    max-width: 500px;
    border: 1px solid #3a3a3a;
    box-shadow: 0 0 20px rgba(0, 0, 0, 0.5);
  }

  .modal h2 {
    margin-top: 0;
    color: #a8ff00;
    border-bottom: 1px solid #3a3a3a;
    padding-bottom: 0.75rem;
    margin-bottom: 1.25rem;
  }

  .modal-buttons {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 2rem;
  }

  .search-form {
    display: flex;
    gap: 1rem;
    align-items: flex-end;
  }

  .search-form .form-group {
    flex: 1;
    margin-bottom: 0;
  }

  .accordion-item {
    border: 1px solid #3a3a3a;
    border-radius: 4px;
    margin-bottom: 0.75rem;
    overflow: hidden;
  }

  .accordion-header {
    background-color: #1a1a1a;
    padding: 0.75rem 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    cursor: pointer;
  }

  .accordion-header h4 {
    margin: 0;
    font-size: 1rem;
    color: #a8ff00;
  }

  .accordion-content {
    padding: 1rem;
    background-color: #2a2a2a;
    max-height: 300px;
    overflow-y: auto;
  }

  footer {
    background-color: #2a2a2a;
    padding: 1rem;
    text-align: center;
    border-top: 2px solid #a8ff00;
    margin-top: auto;
  }

  footer p {
    margin: 0;
    color: #e0e0e0;
    font-size: 0.9rem;
  }
</style>
