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

  document.getElementById('new-repo-platform').addEventListener('change', updateTokenField);

  const tokenInput = document.getElementById('new-repo-token');
  tokenInput.addEventListener('focus', () => {
    if (tokenInput.value === TOKEN_SENTINEL) tokenInput.select();
  });
  tokenInput.addEventListener('input', () => {
    const wrapper = tokenInput.closest('.token-field-wrapper');
    const hint = document.getElementById('token-required-hint');
    if (tokenInput.value.length > 0) {
      wrapper.classList.remove('required');
      hint.classList.remove('visible');
    } else if (!platformHasToken(document.getElementById('new-repo-platform').value)) {
      wrapper.classList.add('required');
      hint.classList.add('visible');
    }
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
    document.getElementById('page-subtitle').textContent = 'Manage users, teams, backups, and configuration.';
    document.getElementById('report-type-row').classList.add('hidden');
    loadAdminUsers();
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
    setReportType('activity');
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
  const team = getTeam();
  if (team && team.Members) {
    team.Members.forEach(m => {
      const opt = document.createElement('option');
      opt.value = m.id;
      opt.textContent = m.displayName || m.username;
      sel.appendChild(opt);
    });
  }
}

// ══════════════════════════════════════════════════════
// REPOSITORIES
// ══════════════════════════════════════════════════════

function getTeam() {
  return teamData.length > 0 ? teamData[0] : null;
}

function renderRepos() {
  const team = getTeam();
  const repos = (team && team.Repositories) || [];
  const isTeamLeader = session.roles.includes('TEAM_LEADER');
  const list = document.getElementById('repo-list');
  if (repos.length === 0) {
    list.innerHTML = '<div class="no-repos">No repositories configured.</div>';
    return;
  }
  list.innerHTML = repos.map(repo => `
    <div class="repo-item">
      <span class="repo-icon">📦</span>
      <div class="repo-info">
        <div class="repo-name" title="${escapeHtml(repo.fullName)}">${escapeHtml(repo.fullName)}</div>
      </div>
      ${isTeamLeader ? `<button class="repo-remove" onclick="removeRepo('${escapeHtml(repo.id)}')" title="Remove">✕</button>` : ''}
    </div>
  `).join('');
}

const TOKEN_SENTINEL = '••••••••••••';

function platformHasToken(platform) {
  const team = getTeam();
  if (!team || !team.Repositories) return false;
  return team.Repositories.some(r => r.platform.toUpperCase() === platform.toUpperCase());
}

function updateTokenField() {
  const platform = document.getElementById('new-repo-platform').value;
  const tokenInput = document.getElementById('new-repo-token');
  const wrapper = tokenInput.closest('.token-field-wrapper');
  const hint = document.getElementById('token-required-hint');
  if (platformHasToken(platform)) {
    tokenInput.value = TOKEN_SENTINEL;
    tokenInput.placeholder = 'Using existing token';
    wrapper.classList.remove('required');
    hint.classList.remove('visible');
  } else {
    tokenInput.value = '';
    tokenInput.placeholder = 'API token (PAT)';
    wrapper.classList.add('required');
    hint.classList.add('visible');
  }
}

function toggleAddRepo() {
  const form = document.getElementById('add-repo-form');
  const open = form.style.display === 'block';
  form.style.display = open ? 'none' : 'block';
  if (!open) {
    updateTokenField();
    document.getElementById('new-repo-name').focus();
  }
}

async function addRepo() {
  const name = document.getElementById('new-repo-name').value.trim();
  const platform = document.getElementById('new-repo-platform').value;
  const rawToken = document.getElementById('new-repo-token').value;
  const token = rawToken === TOKEN_SENTINEL ? '' : rawToken.trim();
  const team = getTeam();
  if (!name || !team) return;
  if (!token && !platformHasToken(platform)) { alert('API token is required to access repository data.'); return; }

  try {
    const resp = await apiFetch(`/api/teams/${encodeURIComponent(team.ID)}/repositories`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ fullName: name, platform: platform, apiToken: token }),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Failed to add repository: ' + (err.error || resp.statusText));
      return;
    }

    document.getElementById('new-repo-name').value = '';
    document.getElementById('new-repo-token').value = '';
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
// PROFILE — Platform Usernames
// ══════════════════════════════════════════════════════

async function showProfile() {
  document.getElementById('screen-app').style.display = 'none';
  document.getElementById('screen-profile').style.display = 'flex';

  const container = document.getElementById('profile-platforms');
  container.innerHTML = '<div style="color:var(--text-muted)">Loading…</div>';
  document.getElementById('profile-status').textContent = '';

  try {
    const resp = await apiFetch('/api/profile');
    if (!resp.ok) throw new Error('Failed to load profile');
    const profile = await resp.json();

    const team = getTeam();
    const repos = (team && team.Repositories) || [];
    const platforms = [...new Set(repos.map(r => r.platform))];

    if (platforms.length === 0) {
      container.innerHTML = '<div style="color:var(--text-muted)">No repositories configured for your team yet.</div>';
      return;
    }

    const currentUsernames = profile.platformUsernames || {};

    container.innerHTML = platforms.map(p => {
      const current = currentUsernames[p] || '';
      const displayName = p === 'GITHUB' ? 'GitHub' : p === 'GITLAB' ? 'GitLab' : p;
      return `
        <div class="profile-platform-row">
          <div class="profile-platform-label">${escapeHtml(displayName)}</div>
          <div class="profile-platform-input">
            <input type="text" id="profile-username-${escapeHtml(p)}"
                   value="${escapeHtml(current)}"
                   placeholder="Your ${escapeHtml(displayName)} username"
                   data-platform="${escapeHtml(p)}">
            <button class="btn btn-primary btn-sm" onclick="savePlatformUsername('${escapeHtml(p)}')">Save</button>
          </div>
        </div>`;
    }).join('');
  } catch (e) {
    container.innerHTML = '<div style="color:#cf222e">Failed to load profile.</div>';
  }
}

async function savePlatformUsername(platform) {
  const input = document.getElementById('profile-username-' + platform);
  const username = input.value.trim();
  const statusEl = document.getElementById('profile-status');

  if (!username) {
    statusEl.textContent = 'Username cannot be empty.';
    statusEl.style.color = '#cf222e';
    return;
  }

  try {
    const resp = await apiFetch('/api/profile/platform-username', {
      method: 'PUT',
      headers: authHeaders(),
      body: JSON.stringify({ platform: platform, username: username }),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      throw new Error(err.error || resp.statusText);
    }

    statusEl.textContent = 'Saved!';
    statusEl.style.color = '#1a7f37';
    setTimeout(() => { statusEl.textContent = ''; }, 2000);
  } catch (e) {
    statusEl.textContent = 'Failed to save: ' + e.message;
    statusEl.style.color = '#cf222e';
  }
}

function backToApp() {
  document.getElementById('screen-profile').style.display = 'none';
  document.getElementById('screen-app').style.display = 'flex';
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
  if (document.getElementById('chk-merged').checked) types.push('MERGED_PR');
  if (document.getElementById('chk-issue').checked) types.push('ISSUE');
  if (document.getElementById('chk-review').checked) types.push('PR_REVIEW');

  const team = getTeam();
  const teamId = team ? team.ID : '';
  const memberId = document.getElementById('filter-member').value;

  const weights = {
    commit: parseFloat(document.getElementById('w-commit').value) || 1,
    merged: parseFloat(document.getElementById('w-merged').value) || 1.5,
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
        memberId: memberId,
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
// ADMIN: USER & TEAM MANAGEMENT
// ══════════════════════════════════════════════════════

let adminUsers = [];
let allTeams = [];

async function loadAdminUsers() {
  try {
    const [usersResp, teamsResp] = await Promise.all([
      apiFetch('/api/admin/users'),
      apiFetch('/api/teams'),
    ]);
    if (usersResp.ok) adminUsers = await usersResp.json();
    if (teamsResp.ok) allTeams = await teamsResp.json();
    if (!Array.isArray(allTeams)) allTeams = [];
  } catch (e) {
    adminUsers = [];
    allTeams = [];
  }
  renderAdminUsers();
  renderAdminTeams();
}

function renderAdminUsers() {
  const tbody = document.getElementById('admin-user-tbody');
  if (!adminUsers || adminUsers.length === 0) {
    tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:var(--text-muted)">No users found.</td></tr>';
    return;
  }

  tbody.innerHTML = adminUsers.map(u => {
    const rolePills = (u.roles || []).map(r => {
      const cls = r === 'ADMIN' ? 'role-ad' : r === 'TEAM_LEADER' ? 'role-tl' : 'role-tm';
      const label = r === 'ADMIN' ? 'Admin' : r === 'TEAM_LEADER' ? 'Leader' : 'Member';
      return `<span class="user-role-pill ${cls}">${label}</span>`;
    }).join('');

    const teamTags = (u.teams || []).map(t =>
      `<span class="team-tag">${escapeHtml(t.name)}<button class="remove-team" onclick="removeMemberFromTeam('${escapeHtml(t.id)}','${escapeHtml(u.id)}')" title="Remove from team">&times;</button></span>`
    ).join('');

    const availableTeams = allTeams.filter(t => !(u.teams || []).some(ut => ut.id === t.ID));
    const addTeamSelect = availableTeams.length > 0
      ? `<select class="add-team-select" onchange="addMemberToTeam(this.value,'${escapeHtml(u.id)}');this.value=''">
           <option value="">+ team</option>
           ${availableTeams.map(t => `<option value="${escapeHtml(t.ID)}">${escapeHtml(t.Name)}</option>`).join('')}
         </select>`
      : '';

    return `<tr>
      <td>${escapeHtml(u.username)}</td>
      <td>${escapeHtml(u.displayName)}</td>
      <td>${escapeHtml(u.email || '')}</td>
      <td>${rolePills}</td>
      <td>${teamTags}${addTeamSelect}</td>
      <td><button class="btn-danger" onclick="deleteUser('${escapeHtml(u.id)}','${escapeHtml(u.username)}')" title="Delete user">&#x1F5D1;</button></td>
    </tr>`;
  }).join('');
}

function toggleCreateUserForm() {
  const form = document.getElementById('create-user-form');
  form.classList.toggle('visible');
  if (form.classList.contains('visible')) {
    document.getElementById('new-user-username').focus();
  }
  document.getElementById('create-user-status').textContent = '';
}

async function createUser() {
  const username = document.getElementById('new-user-username').value.trim().toLowerCase();
  const displayName = document.getElementById('new-user-display').value.trim();
  const email = document.getElementById('new-user-email').value.trim();
  const password = document.getElementById('new-user-password').value;
  const statusEl = document.getElementById('create-user-status');

  if (!username || !displayName || !password) {
    statusEl.textContent = 'Username, display name, and password are required.';
    statusEl.style.color = '#cf222e';
    return;
  }

  const roles = [];
  if (document.getElementById('role-team-member').checked) roles.push('TEAM_MEMBER');
  if (document.getElementById('role-team-leader').checked) roles.push('TEAM_LEADER');
  if (document.getElementById('role-admin').checked) roles.push('ADMIN');

  try {
    const resp = await apiFetch('/api/admin/users', {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ username, displayName, email, password, roles }),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      statusEl.textContent = 'Failed: ' + (err.error || resp.statusText);
      statusEl.style.color = '#cf222e';
      return;
    }

    document.getElementById('new-user-username').value = '';
    document.getElementById('new-user-display').value = '';
    document.getElementById('new-user-email').value = '';
    document.getElementById('new-user-password').value = '';
    document.getElementById('role-team-member').checked = true;
    document.getElementById('role-team-leader').checked = false;
    document.getElementById('role-admin').checked = false;
    document.getElementById('create-user-form').classList.remove('visible');
    statusEl.textContent = '';

    await loadAdminUsers();
  } catch (e) {
    statusEl.textContent = 'Failed: ' + e.message;
    statusEl.style.color = '#cf222e';
  }
}

async function deleteUser(userId, username) {
  if (!confirm(`Delete user "${username}"? This cannot be undone.`)) return;

  try {
    const resp = await apiFetch(`/api/admin/users/${encodeURIComponent(userId)}`, {
      method: 'DELETE',
      headers: authHeaders(),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Failed to delete user: ' + (err.error || resp.statusText));
      return;
    }

    await loadAdminUsers();
  } catch (e) {
    alert('Failed to delete user: ' + e.message);
  }
}

async function addMemberToTeam(teamId, userId) {
  if (!teamId) return;
  try {
    const resp = await apiFetch(`/api/teams/${encodeURIComponent(teamId)}/members`, {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ userId }),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Failed to add member: ' + (err.error || resp.statusText));
      return;
    }

    await loadAdminUsers();
  } catch (e) {
    alert('Failed to add member: ' + e.message);
  }
}

async function removeMemberFromTeam(teamId, userId) {
  try {
    const resp = await apiFetch(`/api/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(userId)}`, {
      method: 'DELETE',
      headers: authHeaders(),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Failed to remove member: ' + (err.error || resp.statusText));
      return;
    }

    await loadAdminUsers();
  } catch (e) {
    alert('Failed to remove member: ' + e.message);
  }
}

// ── TEAM CRUD ──

function renderAdminTeams() {
  const container = document.getElementById('admin-team-list');
  if (!allTeams || allTeams.length === 0) {
    container.innerHTML = '<div style="padding:10px;text-align:center;color:var(--text-muted);font-size:13px">No teams yet.</div>';
    return;
  }
  container.innerHTML = allTeams.map(t => {
    const memberCount = (t.MemberIDs || []).length;
    return `<div class="admin-team-item">
      <div>
        <span style="font-weight:500">${escapeHtml(t.Name)}</span>
        <span style="color:var(--text-muted);font-size:11px;margin-left:8px">${memberCount} member${memberCount !== 1 ? 's' : ''}</span>
      </div>
      <button class="btn-danger" onclick="deleteTeam('${escapeHtml(t.ID)}','${escapeHtml(t.Name)}')" title="Delete team">&#x1F5D1;</button>
    </div>`;
  }).join('');
}

function toggleCreateTeamForm() {
  const form = document.getElementById('create-team-form');
  form.classList.toggle('visible');
  if (form.classList.contains('visible')) {
    document.getElementById('new-team-name').focus();
  }
  document.getElementById('create-team-status').textContent = '';
}

async function createTeam() {
  const name = document.getElementById('new-team-name').value.trim();
  const statusEl = document.getElementById('create-team-status');

  if (!name) {
    statusEl.textContent = 'Team name is required.';
    statusEl.style.color = '#cf222e';
    return;
  }

  try {
    const resp = await apiFetch('/api/admin/teams', {
      method: 'POST',
      headers: authHeaders(),
      body: JSON.stringify({ name }),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      statusEl.textContent = 'Failed: ' + (err.error || resp.statusText);
      statusEl.style.color = '#cf222e';
      return;
    }

    document.getElementById('new-team-name').value = '';
    document.getElementById('create-team-form').classList.remove('visible');
    statusEl.textContent = '';

    await loadAdminUsers();
  } catch (e) {
    statusEl.textContent = 'Failed: ' + e.message;
    statusEl.style.color = '#cf222e';
  }
}

async function deleteTeam(teamId, teamName) {
  if (!confirm(`Delete team "${teamName}"? Members will be unassigned but not deleted.`)) return;

  try {
    const resp = await apiFetch(`/api/admin/teams/${encodeURIComponent(teamId)}`, {
      method: 'DELETE',
      headers: authHeaders(),
    });

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}));
      alert('Failed to delete team: ' + (err.error || resp.statusText));
      return;
    }

    await loadAdminUsers();
  } catch (e) {
    alert('Failed to delete team: ' + e.message);
  }
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
      case 'MERGED_PR': score += c.count * weights.merged; break;
      case 'ISSUE':     score += c.count * weights.issue;  break;
      case 'PR_REVIEW': score += c.count * weights.review; break;
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
  const mergedCount = getCountByType(report, 'MERGED_PR');
  const issueCount = getCountByType(report, 'ISSUE');
  const reviewCount = getCountByType(report, 'PR_REVIEW');

  const color = stringToColor(user.username || user.id);

  const card = document.createElement('div');
  card.className = 'user-report-card';
  card.innerHTML = `
    <div class="card-header" role="button" tabindex="0">
      <div class="avatar" style="background:${color}">${(user.displayName || user.username || '?')[0].toUpperCase()}</div>
      <div class="card-user-info">
        <div class="card-user-name">${escapeHtml(user.displayName || user.username)}</div>
        <div class="card-user-meta">@${escapeHtml(user.username)} &nbsp;&middot;&nbsp; ${escapeHtml(since)} &rarr; ${escapeHtml(until)}</div>
      </div>
      <div class="stat-pills">
        <span class="stat-pill pill-commit" data-filter-type="COMMIT">&#x2B21; ${commitCount} commits</span>
        <span class="stat-pill pill-merged" data-filter-type="MERGED_PR">&#x2714; ${mergedCount} merged PRs</span>
        <span class="stat-pill pill-issue" data-filter-type="ISSUE">&#x25CE; ${issueCount} issues</span>
        <span class="stat-pill pill-review" data-filter-type="PR_REVIEW">&#x1F441; ${reviewCount} reviews</span>
      </div>
      ${isWeighted ? `
      <div class="score-badge">
        <div class="score-value">${score}</div>
        <div class="score-label">Score</div>
      </div>` : ''}
      <span class="collapse-chevron">&#x25BC;</span>
    </div>
    <div class="card-body">
      ${activities.length === 0
        ? '<div style="padding:20px;text-align:center;color:var(--text-muted);font-size:13px">No matching activity in this period.</div>'
        : `<div class="activity-list">${activities.map(a => `
            <div class="activity-item" data-activity-type="${escapeHtml(a.type)}">
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

  const header = card.querySelector('.card-header');
  const body = card.querySelector('.card-body');

  header.addEventListener('click', (e) => {
    if (e.target.closest('.stat-pill')) return;
    card.classList.toggle('collapsed');
  });

  card.querySelectorAll('.stat-pill[data-filter-type]').forEach(pill => {
    pill.addEventListener('click', (e) => {
      e.stopPropagation();
      const type = pill.dataset.filterType;
      const wasActive = pill.classList.contains('active');

      card.querySelectorAll('.stat-pill').forEach(p => p.classList.remove('active'));

      if (wasActive) {
        body.querySelectorAll('.activity-item').forEach(item => item.style.display = '');
      } else {
        pill.classList.add('active');
        body.querySelectorAll('.activity-item').forEach(item => {
          item.style.display = item.dataset.activityType === type ? '' : 'none';
        });
      }

      if (card.classList.contains('collapsed')) {
        card.classList.remove('collapsed');
      }
    });
  });

  return card;
}

// ══════════════════════════════════════════════════════
// BUILD SUMMARY CARD
// ══════════════════════════════════════════════════════

function buildSummaryCard(reports, weights) {
  const isWeighted = reportType === 'weighted';
  let totalActivities = 0;
  let totalCommits = 0;
  let totalMerged = 0;
  let totalIssues = 0;
  let totalReviews = 0;

  const userResults = reports.map(r => {
    const commits = getCountByType(r, 'COMMIT');
    const merged = getCountByType(r, 'MERGED_PR');
    const issues = getCountByType(r, 'ISSUE');
    const reviews = getCountByType(r, 'PR_REVIEW');
    const actCount = (r.activities || []).length;
    totalActivities += actCount;
    totalCommits += commits;
    totalMerged += merged;
    totalIssues += issues;
    totalReviews += reviews;
    return {
      user: r.user,
      commits, merged, issues, reviews,
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
        <div class="stat-value">${totalMerged}</div>
        <div class="stat-label">Merged PRs</div>
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
              <th class="right">Merged PRs</th>
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
                <td class="right lb-breakdown">${r.merged} &times; ${weights.merged}</td>
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
