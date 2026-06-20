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
        if (!content) {
            preview.innerHTML = '<p class="text-zinc-600">Nothing to preview</p>';
        } else {
            // First parse videos manually so marked doesn't turn them into broken images
            let preProcessed = content.replace(/!\[([^\]]*)\]\(([^)]+\.(mp4|webm|ogg))\)/gi, '<video controls class="w-full rounded-xl my-4" src="$2" title="$1"></video>');
            
            // Render remaining markdown with marked.js
            if (typeof marked !== 'undefined') {
                preview.innerHTML = marked.parse(preProcessed);
            } else {
                preview.innerHTML = preProcessed.replace(/\n/g, '<br>');
            }
        }
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

let loggedDatesCache = [];
let currentCalendarMonth = new Date();

window.toggleCalendarPopup = async function() {
    const popup = document.getElementById('calendar-popup');
    if (popup.classList.contains('hidden')) {
        popup.classList.remove('hidden');
        await fetchLoggedDates();
        renderCalendar();
    } else {
        popup.classList.add('hidden');
    }
};

async function fetchLoggedDates() {
    try {
        const res = await fetch('/api/logs/dates');
        if (res.ok) {
            loggedDatesCache = await res.json() || [];
        }
    } catch(e) { console.error(e); }
}

window.changeCalendarMonth = function(offset) {
    currentCalendarMonth.setMonth(currentCalendarMonth.getMonth() + offset);
    renderCalendar();
};

window.setGymLogDate = function(dateStr) {
    currentLogDate = dateStr;
    const display = document.getElementById('log-date-display');
    if (display) display.innerText = currentLogDate;
    loadGymLog(currentLogDate);
    document.getElementById('calendar-popup').classList.add('hidden');
};

function renderCalendar() {
    const monthDisplay = document.getElementById('calendar-month-display');
    const daysContainer = document.getElementById('calendar-days');
    
    if (!monthDisplay || !daysContainer) return;
    
    const year = currentCalendarMonth.getFullYear();
    const month = currentCalendarMonth.getMonth();
    
    const monthNames = ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'];
    monthDisplay.innerText = monthNames[month] + ' ' + year;
    
    const firstDay = new Date(year, month, 1).getDay();
    const daysInMonth = new Date(year, month + 1, 0).getDate();
    
    let html = '';
    // Empty slots for days before the 1st
    for (let i = 0; i < firstDay; i++) {
        html += '<div></div>';
    }
    
    for (let day = 1; day <= daysInMonth; day++) {
        const dateStr = year + '-' + String(month + 1).padStart(2, '0') + '-' + String(day).padStart(2, '0');
        const hasLog = loggedDatesCache.includes(dateStr);
        const isCurrent = dateStr === currentLogDate;
        
        let classes = 'w-10 h-10 flex items-center justify-center rounded-full text-sm cursor-pointer transition-all mx-auto font-medium ';
        if (isCurrent) {
            classes += 'bg-app-pink text-white font-black shadow-[0_0_15px_rgba(255,0,160,0.5)]';
        } else if (hasLog) {
            classes += 'border-2 border-app-pink/50 text-app-pink hover:bg-app-pink/20 shadow-[0_0_10px_rgba(255,0,160,0.2)] font-bold';
        } else {
            classes += 'text-zinc-400 hover:bg-zinc-800 hover:text-white';
        }
        
        html += '<div class="' + classes + '" onclick="setGymLogDate(\'' + dateStr + '\')">' + day + '</div>';
    }
    
    daysContainer.innerHTML = html;
}

