// --- THEME & SETTINGS ---
window.setTheme = function(name) {
    localStorage.setItem('app-theme', name);
    document.documentElement.setAttribute('data-theme', name);
    
    // Notify server to persist
    const formData = new FormData();
    formData.append('theme', name);
    fetch('/api/settings/theme', { method: 'POST', body: formData });
};

// --- GYM LOGS ---
let currentLogDate = new Date().toISOString().split('T')[0];

window.changeLogDate = function(offset) {
    const d = new Date(currentLogDate);
    d.setDate(d.getDate() + offset);
    currentLogDate = d.toISOString().split('T')[0];
    
    const display = document.getElementById('log-date-display');
    if (display) display.innerText = currentLogDate;
    
    loadGymLog(currentLogDate);
};

async function loadGymLog(date) {
    const res = await fetch(`/api/logs?date=${date}`);
    const text = await res.text();
    const area = document.getElementById('log-content');
    if (area) area.value = text;
    
    // If in preview mode, refresh preview
    const preview = document.getElementById('log-preview-container');
    if (preview && !preview.classList.contains('hidden')) {
        toggleLogMode('preview');
    }
}

window.saveGymLog = async function() {
    const content = document.getElementById('log-content').value;
    const date = currentLogDate;
    
    const formData = new FormData();
    formData.append('content', content);
    formData.append('date', date);
    
    const btn = event?.target;
    if (btn) btn.disabled = true;
    
    try {
        const res = await fetch('/api/logs', { method: 'POST', body: formData });
        if (res.ok) {
            // Optional: visual feedback
            const originalText = btn ? btn.innerText : 'Save Log';
            if (btn) btn.innerText = 'SAVED!';
            setTimeout(() => { if (btn) { btn.innerText = originalText; btn.disabled = false; } }, 1500);
        }
    } catch(e) {
        console.error(e);
        if (btn) btn.disabled = false;
    }
};

window.toggleLogMode = function(mode) {
    const edit = document.getElementById('log-edit-container');
    const preview = document.getElementById('log-preview-container');
    const btnEdit = document.getElementById('btn-edit');
    const btnPreview = document.getElementById('btn-preview');
    
    if (mode === 'preview') {
        const content = document.getElementById('log-content').value;
        preview.innerHTML = content ? content.replace(/\n/g, '<br>') : '<p class="text-zinc-600">Nothing to preview</p>';
        edit.classList.add('hidden');
        preview.classList.remove('hidden');
        btnPreview.classList.add('bg-app-pink', 'text-white');
        btnPreview.classList.remove('bg-app-card', 'text-zinc-400');
        btnEdit.classList.remove('bg-app-pink', 'text-white');
        btnEdit.classList.add('bg-app-card', 'text-zinc-400');
    } else {
        edit.classList.remove('hidden');
        preview.classList.add('hidden');
        btnEdit.classList.add('bg-app-pink', 'text-white');
        btnEdit.classList.remove('bg-app-card', 'text-zinc-400');
        btnPreview.classList.remove('bg-app-pink', 'text-white');
        btnPreview.classList.add('bg-app-card', 'text-zinc-400');
    }
};

window.exportCurrentLog = function() {
    window.location.href = '/api/logs/export';
};

window.setPomo = function(p, s, l) {
    document.getElementsByName('pomo_duration')[0].value = p;
    document.getElementsByName('short_break')[0].value = s;
    document.getElementsByName('long_break')[0].value = l;
    document.getElementById('pomo-val').innerText = p;
    document.getElementById('short-val').innerText = s;
    document.getElementById('long-val').innerText = l;
};

// Initial load for today's log if on settings page
document.addEventListener('DOMContentLoaded', () => {
    const display = document.getElementById('log-date-display');
    if (display) {
        // Use the text content if already set by server, otherwise default to today
        if (display.innerText && display.innerText.match(/^\d{4}-\d{2}-\d{2}$/)) {
            currentLogDate = display.innerText;
        } else {
            display.innerText = currentLogDate;
        }
        loadGymLog(currentLogDate);
    }
});

// Make the animation globally available to HTMX & Go!
window.animateRing = function(elementId, percentage) {
    const circle = document.getElementById(elementId);
    if(!circle) return;
    const radius = circle.r.baseVal.value;
    const circumference = radius * 2 * Math.PI;
    circle.style.strokeDasharray = `${circumference} ${circumference}`;
    circle.style.strokeDashoffset = circumference;
    const offset = circumference - (percentage / 100) * circumference;
    setTimeout(() => { circle.style.strokeDashoffset = offset; }, 100);
};

// Keep the Coin Flash Animation
document.addEventListener("DOMContentLoaded", () => {
  document.body.addEventListener('htmx:oobAfterSwap', function(event) {
    if (event.detail.target.id === 'coin-widget') {
        const coinDisplay = document.getElementById('total-coins');
        if(coinDisplay) {
            coinDisplay.classList.add('coin-added');
            setTimeout(() => coinDisplay.classList.remove('coin-added'), 300);
        }
    }
  });
});