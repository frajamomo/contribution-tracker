// ══════════════════════════════════════════════════════
// STATE
// ══════════════════════════════════════════════════════

let session = null;       // { token, roles, user: {id, username, displayName, avatarUrl} }
let reportType = 'activity'; // 'activity' | 'weighted'
let teamData = [];        // array of team objects from API
let currentAbort = null;  // AbortController for in-flight report stream

// ══════════════════════════════════════════════════════
// API HELPERS
// ══════════════════════════════════════════════════════

function getToken() {
  return sessionStorage.getItem('token');
}

function authHeaders() {
  const token = getToken();
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = 'Bearer ' + token;
  return headers;
}

async function apiFetch(url, options = {}) {
  if (!options.headers) options.headers = authHeaders();
  const resp = await fetch(url, options);
  if (resp.status === 401) {
    doLogout();
    throw new Error('Session expired');
  }
  return resp;
}

// ══════════════════════════════════════════════════════
// AUTH
// ══════════════════════════════════════════════════════

async function doLogin() {
  const username = document.getElementById('inp-username').value.trim().toLowerCase();
  const password = document.getElementById('inp-password').value;
  const errEl = document.getElementById('login-error');

  if (!username || !password) {
    errEl.style.display = 'block';
    return;
  }

  try {
    const resp = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });

    if (!resp.ok) {
      errEl.style.display = 'block';
      return;
    }

    const data = await resp.json();
    sessionStorage.setItem('token', data.token);
    session = {
      token: data.token,
      roles: data.roles || [],
      user: data.user,
    };
    errEl.style.display = 'none';
    await mountApp();
  } catch (e) {
    errEl.style.display = 'block';
  }
}

document.addEventListener('DOMContentLoaded', () => {
  ['inp-username', 'inp-password'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.addEventListener('keydown', e => { if (e.key === 'Enter') doLogin(); });
  });
});

function doLogout() {
  session = null;
  sessionStorage.removeItem('token');
  if (currentAbort) { currentAbort.abort(); currentAbort = null; }
  document.getElementById('screen-app').style.display = 'none';
  document.getElementById('screen-login').style.display = 'flex';
  document.getElementById('inp-username').value = '';
  document.getElementById('inp-password').value = '';
  document.getElementById('results-area').innerHTML = '';
}

// ══════════════════════════════════════════════════════
// APP MOUNT
// ══════════════════════════════════════════════════════

async function mountApp() {
  document.getElementById('screen-login').style.display = 'none';
  document.getElementById('screen-app').style.display = 'flex';

  const user = session.user;
  const roles = session.roles;
  const isAdmin = roles.includes('ADMIN');
  const isTeamLeader = roles.includes('TEAM_LEADER');

  // Navbar
  const av = document.getElementById('nav-avatar');
  av.textContent = (user.displayName || user.username)[0].toUpperCase();
  av.style.background = isAdmin ? '#cf222e' : isTeamLeader ? '#8250df' : '#0969da';
  document.getElementById('nav-name').textContent = user.displayName || user.username;
  const badge = document.getElementById('nav-role-badge');
  const primaryRole = isAdmin ? 'ADMIN' : isTeamLeader ? 'TEAM_LEADER' : 'TEAM_MEMBER';
  badge.textContent = primaryRole;
  const roleClass = { TEAM_MEMBER: 'role-user', TEAM_LEADER: 'role-manager', ADMIN: 'role-admin' };
  badge.className = 'role-badge ' + (roleClass[primaryRole] || 'role-user');

  // Show/hide sections by role
  document.querySelector('.report-form').classList.toggle('hidden', isAdmin);
  document.getElementById('results-area').classList.toggle('hidden', isAdmin);
  document.getElementById('admin-panel').classList.toggle('hidden', !isAdmin);
  document.querySelector('.sidebar').classList.toggle('hidden', isAdmin);

  if (isAdmin) {
    document.getElementById('page-title').textContent = 'Administration';
    document.getElementById('page-subtitle').textContent = 'Manage application data backups and restores.';
    document.getElementById('report-type-row').classList.add('hidden');
    return;
  }

  // Fetch teams data
  try {
    const resp = await apiFetch('/api/teams');
    if (resp.ok) {
      teamData = await resp.json();
      if (!Array.isArray(teamData)) teamData = [];
    }
  } catch (e) {
    teamData = [];
  }

  // Page header
  if (isTeamLeader) {
    document.getElementById('page-title').textContent = 'Team Report';
    document.getElementById('page-subtitle').textContent = 'View contribution activity across your entire team.';
    document.getElementById('field-member').classList.remove('hidden');
    document.getElementById('report-type-row').classList.remove('hidden');
    document.getElementById('add-repo-btn').classList.remove('hidden');
    document.getElementById('repo-managed-note').classList.add('hidden');
    setReportType('activity');
    document.getElementById('form-hint').textContent = 'Results stream in per member as data is fetched from each source.';
  } else {
    document.getElementById('page-title').textContent = 'My Activity';
    document.getElementById('page-subtitle').textContent = 'View your own contribution activity across your team\'s repositories.';
    document.getElementById('field-member').classList.add('hidden');
    document.getElementById('report-type-row').classList.add('hidden');
    document.getElementById('add-repo-btn').classList.add('hidden');
    document.getElementById('repo-managed-note').classList.remove('hidden');
    document.getElementById('form-hint').textContent = 'Repositories are configured by your team leader.';
  }

  document.getElementById('filter-period').value = 'last30';
  onPeriodChange();
  renderRepos();
  populateTeamMembers();
  document.getElementById('results-area').innerHTML = '';
}

// ══════════════════════════════════════════════════════
// TEAM MEMBERS (for filter dropdown)
// ══════════════════════════════════════════════════════

function populateTeamMembers() {
  // Note: the team member list comes from teamData. Members are IDs, but
  // for the dropdown we display them. The API returns teams with MemberIDs;
  // the display names come in the report stream itself.
  // For the filter dropdown, we show member IDs until the user generates a report.
  const sel = document.getElementById('filter-member');
  sel.innerHTML = '<option value="">All members</option>';
  // We don't have displayNames for members yet, but we can show the IDs.
  // The actual names will appear in report cards from the SSE stream.
}

// ══════════════════════════════════════════════════════
// REPOSITORIES
// ══════════════════════════════════════════════════════

function getTeam() {
  return teamData.length > 0 ? teamData[0] : null;
}

function renderRepos() {
  const team = getTeam();
  const repos = (team && team.RepositoryIDs) || [];
  const isTeamLeader = session.roles.includes('TEAM_LEADER');
  const list = document.getElementById('repo-list');
  if (repos.length === 0) {
    list.innerHTML = '<div class="no-repos">No repositories configured.</div>';
    return;
  }
  list.innerHTML = repos.map((repoId, i) => `
    <div class="repo-item">
      <span class="repo-icon">📦</span>
      <div class="repo-info">
        <div class="repo-name" title="${escapeHtml(repoId)}">${escapeHtml(repoId)}</div>
      </div>
      ${isTeamLeader ? `<button class="repo-remove" onclick="removeRepo('${escapeHtml(repoId)}')" title="Remove">✕</button>` : ''}
    </div>
  `).join('');
}

function toggleAddRepo() {
  const form = document.getElementById('add-repo-form');
  const open = form.style.display === 'block';
  form.style.display = open ? 'none' : 'block';
  if (!open) document.getElementById('new-repo-name').focus();
}

async function addRepo() {
  const name = document.getElementById('new-repo-name').value.trim();
  const team = getTeam();
  if (!name || !team) return;

  try {
    const resp = await apiFetch(`/api/teams/${encodeURIComponent(team.ID)}/repositories`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ repoId: name }),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Failed to add repository: ' + (err.error || resp.statusText));
      return;
    }

    document.getElementById('new-repo-name').value = '';
    document.getElementById('add-repo-form').style.display = 'none';

    // Refresh teams
    await refreshTeams();
    renderRepos();
  } catch (e) {
    alert('Failed to add repository: ' + e.message);
  }
}

async function removeRepo(repoId) {
  const team = getTeam();
  if (!team) return;

  try {
    const resp = await apiFetch(`/api/teams/${encodeURIComponent(team.ID)}/repositories/${encodeURIComponent(repoId)}`, {
      method: 'DELETE',
      headers: authHeaders(),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Failed to remove repository: ' + (err.error || resp.statusText));
      return;
    }

    await refreshTeams();
    renderRepos();
  } catch (e) {
    alert('Failed to remove repository: ' + e.message);
  }
}

async function refreshTeams() {
  try {
    const resp = await apiFetch('/api/teams');
    if (resp.ok) {
      teamData = await resp.json();
      if (!Array.isArray(teamData)) teamData = [];
    }
  } catch (e) {
    // keep current data
  }
}

// ══════════════════════════════════════════════════════
// PERIOD RESOLUTION
// ══════════════════════════════════════════════════════

function onPeriodChange() {
  const period = document.getElementById('filter-period').value;
  const customEl = document.getElementById('custom-range');
  if (period === 'custom') {
    customEl.classList.add('visible');
    return;
  }
  customEl.classList.remove('visible');
  const { since, until } = resolvePeriod(period);
  document.getElementById('filter-since').value = since;
  document.getElementById('filter-until').value = until;
}

function resolvePeriod(period) {
  const today = new Date();
  const fmt = d => d.toISOString().slice(0, 10);

  const startOfQuarter = d => {
    const q = Math.floor(d.getMonth() / 3);
    return new Date(d.getFullYear(), q * 3, 1);
  };

  switch (period) {
    case 'last7': {
      const s = new Date(today); s.setDate(s.getDate() - 6);
      return { since: fmt(s), until: fmt(today) };
    }
    case 'last30': {
      const s = new Date(today); s.setDate(s.getDate() - 29);
      return { since: fmt(s), until: fmt(today) };
    }
    case 'currentQuarter': {
      return { since: fmt(startOfQuarter(today)), until: fmt(today) };
    }
    case 'lastQuarter': {
      const sq = startOfQuarter(today);
      const end = new Date(sq); end.setDate(end.getDate() - 1);
      return { since: fmt(startOfQuarter(end)), until: fmt(end) };
    }
    case 'currentYear': {
      return { since: `${today.getFullYear()}-01-01`, until: fmt(today) };
    }
    case 'lastYear': {
      const y = today.getFullYear() - 1;
      return { since: `${y}-01-01`, until: `${y}-12-31` };
    }
    default:
      return { since: fmt(today), until: fmt(today) };
  }
}

// ══════════════════════════════════════════════════════
// REPORT TYPE
// ══════════════════════════════════════════════════════

function setReportType(type) {
  reportType = type;
  document.getElementById('tab-activity').classList.toggle('active', type === 'activity');
  document.getElementById('tab-weighted').classList.toggle('active', type === 'weighted');
  document.getElementById('weights-panel').classList.toggle('visible', type === 'weighted');
  document.getElementById('form-hint').textContent = type === 'weighted'
    ? 'Score = commits x weight + issues x weight + reviews x weight, per member.'
    : 'Results stream in per member as data is fetched from each source.';
}

// ══════════════════════════════════════════════════════
// REPORT GENERATION (real SSE stream via POST + fetch)
// ══════════════════════════════════════════════════════

async function generateReport() {
  const area = document.getElementById('results-area');
  area.innerHTML = '';

  // Abort any in-flight stream
  if (currentAbort) { currentAbort.abort(); currentAbort = null; }

  const period = document.getElementById('filter-period').value;
  const { since, until } = period === 'custom'
    ? { since: document.getElementById('filter-since').value, until: document.getElementById('filter-until').value }
    : resolvePeriod(period);

  if (!since || !until) {
    alert('Please select a valid date range.');
    return;
  }

  const types = [];
  if (document.getElementById('chk-commit').checked) types.push('COMMIT');
  if (document.getElementById('chk-issue').checked) types.push('ISSUE');
  if (document.getElementById('chk-review').checked) types.push('PR_REVIEW');

  const team = getTeam();
  const teamId = team ? team.ID : '';

  const weights = {
    commit: parseFloat(document.getElementById('w-commit').value) || 1,
    issue:  parseFloat(document.getElementById('w-issue').value)  || 0.5,
    review: parseFloat(document.getElementById('w-review').value) || 0.8,
  };

  // Status bar
  const statusEl = document.createElement('div');
  statusEl.className = 'stream-status';
  statusEl.innerHTML = '<div class="pulse"></div><span id="stream-msg">Opening stream...</span>';
  area.appendChild(statusEl);

  const abortController = new AbortController();
  currentAbort = abortController;

  const allReports = []; // collect UserReportDTOs for summary

  try {
    const resp = await fetch('/api/reports/stream', {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({
        teamId: teamId,
        since: since,
        until: until,
        types: types,
        reportType: reportType,
      }),
      signal: abortController.signal,
    });

    if (resp.status === 401) {
      doLogout();
      return;
    }

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      statusEl.className = 'stream-status error';
      document.getElementById('stream-msg').textContent = 'Error: ' + (err.error || resp.statusText);
      return;
    }

    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // Parse SSE events from buffer
      // SSE format: "event: TYPE\ndata: JSON\n\n"
      const events = [];
      while (true) {
        const eventEnd = buffer.indexOf('\n\n');
        if (eventEnd === -1) break;

        const rawEvent = buffer.substring(0, eventEnd);
        buffer = buffer.substring(eventEnd + 2);

        let eventType = '';
        let eventData = '';

        for (const line of rawEvent.split('\n')) {
          if (line.startsWith('event: ')) {
            eventType = line.substring(7).trim();
          } else if (line.startsWith('data: ')) {
            eventData = line.substring(6);
          }
        }

        if (eventType && eventData) {
          events.push({ type: eventType, data: eventData });
        }
      }

      for (const evt of events) {
        try {
          const parsed = JSON.parse(evt.data);

          switch (evt.type) {
            case 'USER_REPORT': {
              const report = parsed.report;
              if (!report) break;
              allReports.push(report);
              const streamMsg = document.getElementById('stream-msg');
              if (streamMsg) {
                streamMsg.textContent = `Received data for ${report.user.displayName || report.user.username}...`;
              }
              const card = buildUserCard(report, since, until, weights);
              area.insertBefore(card, statusEl);
              break;
            }
            case 'COMPLETE': {
              const summaryCard = buildSummaryCard(allReports, weights);
              area.insertBefore(summaryCard, statusEl);
              statusEl.className = 'stream-status done';
              const streamMsg = document.getElementById('stream-msg');
              if (streamMsg) {
                const totalActivities = allReports.reduce((sum, r) => sum + (r.activities ? r.activities.length : 0), 0);
                streamMsg.textContent = `Report complete -- ${totalActivities} activities across ${allReports.length} ${allReports.length === 1 ? 'member' : 'members'}.`;
              }
              break;
            }
            case 'ERROR': {
              statusEl.className = 'stream-status error';
              const streamMsg = document.getElementById('stream-msg');
              if (streamMsg) {
                streamMsg.textContent = 'Error: ' + (parsed.error || 'Unknown error');
              }
              break;
            }
          }
        } catch (parseErr) {
          // skip malformed events
        }
      }
    }
  } catch (e) {
    if (e.name === 'AbortError') return; // user cancelled
    statusEl.className = 'stream-status error';
    const streamMsg = document.getElementById('stream-msg');
    if (streamMsg) {
      streamMsg.textContent = 'Connection error: ' + e.message;
    }
  } finally {
    if (currentAbort === abortController) currentAbort = null;
  }
}

// ══════════════════════════════════════════════════════
// BACKUP / RESTORE (admin)
// ══════════════════════════════════════════════════════

let selectedBackupFile = null;

async function exportBackup() {
  const btn = document.getElementById('export-btn');
  btn.textContent = 'Generating...';
  btn.disabled = true;

  try {
    const resp = await apiFetch('/api/admin/backup');

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Export failed: ' + (err.error || resp.statusText));
      btn.textContent = '↓ Download backup.json';
      btn.disabled = false;
      return;
    }

    const data = await resp.json();
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url  = URL.createObjectURL(blob);
    const a    = document.createElement('a');
    a.href     = url;
    a.download = `backup-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
  } catch (e) {
    alert('Export failed: ' + e.message);
  } finally {
    btn.textContent = '↓ Download backup.json';
    btn.disabled = false;
  }
}

function onFileSelected(event) {
  const file = event.target.files[0];
  if (file) setSelectedFile(file);
}

function onDragOver(event) {
  event.preventDefault();
  document.getElementById('drop-zone').classList.add('drag-over');
}

function onDragLeave(event) {
  document.getElementById('drop-zone').classList.remove('drag-over');
}

function onDrop(event) {
  event.preventDefault();
  document.getElementById('drop-zone').classList.remove('drag-over');
  const file = event.dataTransfer.files[0];
  if (file) setSelectedFile(file);
}

function setSelectedFile(file) {
  selectedBackupFile = file;
  document.getElementById('selected-file-name').textContent = file.name;
  document.getElementById('selected-file').classList.add('visible');
  document.getElementById('restore-btn').disabled = false;
}

function clearFile() {
  selectedBackupFile = null;
  document.getElementById('selected-file').classList.remove('visible');
  document.getElementById('restore-btn').disabled = true;
  document.getElementById('backup-file-input').value = '';
}

function openRestore() {
  if (!selectedBackupFile) return;
  document.querySelector('.restore-dialog p').textContent =
    `This will import "${selectedBackupFile.name}" and overwrite all current application data. This action cannot be undone.`;
  document.getElementById('restore-overlay').classList.add('visible');
}

function closeRestore() {
  document.getElementById('restore-overlay').classList.remove('visible');
}

async function confirmRestore() {
  const btn = document.getElementById('confirm-restore-btn');
  btn.textContent = 'Restoring...';
  btn.disabled = true;

  try {
    const text = await readFileAsText(selectedBackupFile);
    const data = JSON.parse(text);

    const resp = await apiFetch('/api/admin/restore', {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify(data),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Restore failed: ' + (err.error || resp.statusText));
    } else {
      closeRestore();
      clearFile();
      alert('Restore complete. The application data has been updated.');
    }
  } catch (e) {
    if (e instanceof SyntaxError) {
      alert('Invalid backup file. Please upload a valid JSON export.');
    } else {
      alert('Restore failed: ' + e.message);
    }
  } finally {
    btn.textContent = 'Restore';
    btn.disabled = false;
  }
}

function readFileAsText(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = e => resolve(e.target.result);
    reader.onerror = () => reject(new Error('Failed to read file'));
    reader.readAsText(file);
  });
}

// ══════════════════════════════════════════════════════
// SCORING
// ══════════════════════════════════════════════════════

function computeScore(report, weights) {
  if (!report.counts) return 0;
  let score = 0;
  for (const c of report.counts) {
    switch (c.type) {
      case 'COMMIT':    score += c.count * weights.commit; break;
      case 'ISSUE':     score += c.count * weights.issue;  break;
      case 'PR_REVIEW': score += c.count * weights.review; break;
      case 'MERGED_PR': score += c.count * weights.review; break;
    }
  }
  return Math.round(score * 10) / 10;
}

function getCountByType(report, type) {
  if (!report.counts) return 0;
  const c = report.counts.find(c => c.type === type);
  return c ? c.count : 0;
}

// ══════════════════════════════════════════════════════
// BUILD USER REPORT CARD
// ══════════════════════════════════════════════════════

function buildUserCard(report, since, until, weights) {
  const user = report.user;
  const activities = report.activities || [];
  const counts = report.counts || [];
  const isWeighted = reportType === 'weighted';
  const score = computeScore(report, weights);

  const commitCount = getCountByType(report, 'COMMIT');
  const issueCount = getCountByType(report, 'ISSUE');
  const reviewCount = getCountByType(report, 'PR_REVIEW') + getCountByType(report, 'MERGED_PR');

  const color = stringToColor(user.username || user.id);

  const card = document.createElement('div');
  card.className = 'user-report-card';
  card.innerHTML = `
    <div class="card-header">
      <div class="avatar" style="background:${color}">${(user.displayName || user.username || '?')[0].toUpperCase()}</div>
      <div class="card-user-info">
        <div class="card-user-name">${escapeHtml(user.displayName || user.username)}</div>
        <div class="card-user-meta">@${escapeHtml(user.username)} &nbsp;&middot;&nbsp; ${escapeHtml(since)} &rarr; ${escapeHtml(until)}</div>
      </div>
      <div class="stat-pills">
        <span class="stat-pill pill-commit">&#x2B21; ${commitCount} commits</span>
        <span class="stat-pill pill-issue">&#x25CE; ${issueCount} issues</span>
        <span class="stat-pill pill-review">&#x1F441; ${reviewCount} reviews</span>
      </div>
      ${isWeighted ? `
      <div class="score-badge">
        <div class="score-value">${score}</div>
        <div class="score-label">Score</div>
      </div>` : ''}
    </div>
    <div class="card-body">
      ${activities.length === 0
        ? '<div style="padding:20px;text-align:center;color:var(--text-muted);font-size:13px">No matching activity in this period.</div>'
        : `<div class="activity-list">${activities.map(a => `
            <div class="activity-item">
              <span class="activity-icon">${activityIcon(a.type)}</span>
              <div class="activity-content">
                <div class="activity-title">${a.url ? `<a href="${escapeHtml(a.url)}" target="_blank" style="color:inherit;text-decoration:none">${escapeHtml(a.title)}</a>` : escapeHtml(a.title)}</div>
                <div class="activity-meta">
                  <span>${escapeHtml(a.displayName || a.type)}</span>
                  ${a.summary ? `<span>${escapeHtml(a.summary)}</span>` : ''}
                </div>
              </div>
              <div class="activity-date">${formatDate(a.createdAt)}</div>
            </div>`).join('')}
          </div>`
      }
    </div>`;
  return card;
}

// ══════════════════════════════════════════════════════
// BUILD SUMMARY CARD
// ══════════════════════════════════════════════════════

function buildSummaryCard(reports, weights) {
  const isWeighted = reportType === 'weighted';
  let totalActivities = 0;
  let totalCommits = 0;
  let totalIssues = 0;
  let totalReviews = 0;

  const userResults = reports.map(r => {
    const commits = getCountByType(r, 'COMMIT');
    const issues = getCountByType(r, 'ISSUE');
    const reviews = getCountByType(r, 'PR_REVIEW') + getCountByType(r, 'MERGED_PR');
    const actCount = (r.activities || []).length;
    totalActivities += actCount;
    totalCommits += commits;
    totalIssues += issues;
    totalReviews += reviews;
    return {
      user: r.user,
      commits, issues, reviews,
      score: computeScore(r, weights),
    };
  });

  const card = document.createElement('div');
  card.className = 'summary-card';

  const statsHtml = `
    <div class="summary-grid">
      <div class="summary-stat stat-blue">
        <div class="stat-value">${totalActivities}</div>
        <div class="stat-label">Total activities</div>
      </div>
      <div class="summary-stat stat-blue">
        <div class="stat-value">${totalCommits}</div>
        <div class="stat-label">Commits</div>
      </div>
      <div class="summary-stat stat-green">
        <div class="stat-value">${totalIssues}</div>
        <div class="stat-label">Issues</div>
      </div>
      <div class="summary-stat stat-purple">
        <div class="stat-value">${totalReviews}</div>
        <div class="stat-label">PR Reviews</div>
      </div>
    </div>`;

  let leaderboardHtml = '';
  if (isWeighted && userResults.length > 1) {
    const sorted = [...userResults].sort((a, b) => b.score - a.score);
    leaderboardHtml = `
      <div style="margin-top:20px">
        <div style="font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:0.4px;color:var(--text-muted);margin-bottom:10px">Team Ranking</div>
        <table class="leaderboard">
          <thead>
            <tr>
              <th style="width:40px">#</th>
              <th>Member</th>
              <th class="right">Commits</th>
              <th class="right">Issues</th>
              <th class="right">Reviews</th>
              <th class="right">Score</th>
            </tr>
          </thead>
          <tbody>
            ${sorted.map((r, i) => {
              const color = stringToColor(r.user.username || r.user.id);
              return `
              <tr class="rank-${i + 1}">
                <td><span class="rank-num">${i + 1}</span></td>
                <td>
                  <div class="lb-member">
                    <div class="avatar" style="background:${color};width:24px;height:24px;font-size:11px">${(r.user.displayName || r.user.username || '?')[0].toUpperCase()}</div>
                    ${escapeHtml(r.user.displayName || r.user.username)}
                  </div>
                </td>
                <td class="right lb-breakdown">${r.commits} &times; ${weights.commit}</td>
                <td class="right lb-breakdown">${r.issues} &times; ${weights.issue}</td>
                <td class="right lb-breakdown">${r.reviews} &times; ${weights.review}</td>
                <td class="right"><span class="lb-score">${r.score}</span></td>
              </tr>`;
            }).join('')}
          </tbody>
        </table>
      </div>`;
  }

  card.innerHTML = `
    <h3>Report Summary &middot; ${reports.length} ${reports.length === 1 ? 'member' : 'members'}</h3>
    ${statsHtml}
    ${leaderboardHtml}`;
  return card;
}

// ══════════════════════════════════════════════════════
// UTILITY FUNCTIONS
// ══════════════════════════════════════════════════════

function activityIcon(type) {
  switch (type) {
    case 'COMMIT':    return '&#x2B21;';  // hexagon
    case 'ISSUE':     return '&#x25CE;';  // bullseye
    case 'PR_REVIEW': return '&#x1F441;'; // eye
    case 'MERGED_PR': return '&#x1F500;'; // merge arrows
    default:          return '&#x25CF;';  // circle
  }
}

function formatDate(isoStr) {
  if (!isoStr) return '';
  try {
    const d = new Date(isoStr);
    const now = new Date();
    const diffMs = now - d;
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    if (diffDays === 0) return 'today';
    if (diffDays === 1) return '1 day ago';
    if (diffDays < 30) return `${diffDays} days ago`;
    return d.toISOString().slice(0, 10);
  } catch {
    return isoStr;
  }
}

function escapeHtml(str) {
  if (!str) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function stringToColor(str) {
  // Deterministic color from a string
  const colors = ['#0969da', '#1a7f37', '#8250df', '#9a6700', '#cf222e', '#0550ae', '#116329', '#6639ba'];
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  return colors[Math.abs(hash) % colors.length];
}
