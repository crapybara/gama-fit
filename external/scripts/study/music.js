import { qs, readableName } from './utils.js';

let playlist = []; // Array of {url, title}
let currentTrack = 0;
export let isMusicPlaying = false;
const trackCache = new Map(); // url -> boolean (valid)

const audio = qs("#bg-audio");
const musicSearchTrigger = qs("#music-search-trigger");
const musicPresetSelect = qs("#music-preset-select");
const musicPlayIcon = qs("#music-play-icon");
const musicPauseIcon = qs("#music-pause-icon");
const searchPopup = qs("#search-popup");
const searchInput = qs("#search-input");
const searchResults = qs("#search-results");

export async function loadMusicPresets(preferredPreset) {
  let presets = [];
  try {
    const res = await fetch("/api/resources/music-presets");
    const data = await res.json();
    presets = Array.isArray(data) ? data : [];
  } catch (_) {
    presets = [];
  }

  if (presets.length === 0) presets = ["music.txt"];
  musicPresetSelect.innerHTML = "";
  presets.forEach((preset) => {
    const option = document.createElement("option");
    option.value = preset;
    option.textContent = readableName(preset);
    musicPresetSelect.appendChild(option);
  });

  const saved = preferredPreset || localStorage.getItem("study-music-preset") || presets[0];
  musicPresetSelect.value = presets.includes(saved) ? saved : presets[0];
  await loadMusicPreset(musicPresetSelect.value);
}

export async function loadMusicPreset(preset) {
  localStorage.setItem("study-music-preset", preset);
  try {
    const res = await fetch(`/api/resources/music?preset=${encodeURIComponent(preset)}`);
    const lines = await res.json();
    playlist = parsePlaylist(lines);
  } catch (_) {
    playlist = [];
  }
  currentTrack = 0;
  setAudioTrack(currentTrack, isMusicPlaying);
}

function parsePlaylist(lines) {
  if (!Array.isArray(lines) || lines.length === 0) return [];
  
  let baseUrl = "";
  const parsed = [];
  
  for (let line of lines) {
    line = line.trim();
    if (!line) continue;
    
    if (line.startsWith("http") && !line.includes("|")) {
      baseUrl = line;
      if (!baseUrl.endsWith("/")) baseUrl += "/";
      if (line.endsWith(".mp3") || line.endsWith(".wav")) {
        parsed.push({ url: line, title: readableName(line) });
      }
      continue;
    }
    
    if (line.includes("|")) {
      const [urlPart, title] = line.split("|");
      const fullUrl = urlPart.startsWith("http") ? urlPart : baseUrl + urlPart;
      parsed.push({ url: fullUrl, title: title.trim() });
    } else {
      if (baseUrl) {
        parsed.push({ url: baseUrl + line, title: readableName(line) });
      }
    }
  }
  return parsed.filter(t => t.url && t.url.startsWith("http"));
}

async function validateTrack(url) {
  if (trackCache.has(url)) return trackCache.get(url);
  try {
    const res = await fetch(url, { method: "HEAD" });
    const isValid = res.ok;
    trackCache.set(url, isValid);
    return isValid;
  } catch (_) {
    trackCache.set(url, false);
    return false;
  }
}

async function setAudioTrack(index, autoplay) {
  if (playlist.length === 0) {
    audio.removeAttribute("src");
    musicSearchTrigger.textContent = "No tracks found";
    return;
  }

  let attempts = 0;
  while (attempts < playlist.length) {
    currentTrack = (index + attempts + playlist.length) % playlist.length;
    const track = playlist[currentTrack];
    
    musicSearchTrigger.textContent = "Validating...";
    const isValid = await validateTrack(track.url);
    
    if (isValid) {
      audio.src = track.url;
      musicSearchTrigger.textContent = track.title;
      if (autoplay) playMusic();
      return;
    }
    attempts++;
  }
  
  musicSearchTrigger.textContent = "No valid tracks";
}

export function updateMusicButtons() {
  musicPlayIcon.classList.toggle("hidden", isMusicPlaying);
  musicPauseIcon.classList.toggle("hidden", !isMusicPlaying);
}

export function playMusic() {
  if (!audio.src && playlist.length > 0) {
    setAudioTrack(currentTrack, true);
    return;
  }
  audio.play()
    .then(() => {
      isMusicPlaying = true;
      updateMusicButtons();
    })
    .catch(() => {
      isMusicPlaying = false;
      updateMusicButtons();
    });
}

export function pauseMusic() {
  audio.pause();
  isMusicPlaying = false;
  updateMusicButtons();
}

export function toggleMusic() {
  if (isMusicPlaying) pauseMusic();
  else playMusic();
}

export function nextTrack() {
  setAudioTrack(currentTrack + 1, isMusicPlaying);
}

export function prevTrack() {
  setAudioTrack(currentTrack - 1, isMusicPlaying);
}

export function shufflePlaylist() {
  if (playlist.length <= 1) return;
  for (let i = playlist.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [playlist[i], playlist[j]] = [playlist[j], playlist[i]];
  }
  currentTrack = 0;
  setAudioTrack(currentTrack, isMusicPlaying);
}

export function handleSearch() {
  const query = searchInput.value.toLowerCase();
  searchResults.innerHTML = "";
  const filtered = playlist.filter(track => track.title.toLowerCase().includes(query));
  
  filtered.slice(0, 500).forEach(track => {
    const btn = document.createElement("button");
    
    // Process title: remove leading numbers, then truncate to 20 words
    let cleanTitle = track.title.replace(/^\d+[\s.-]*/, "").trim();
    const words = cleanTitle.split(/\s+/);
    if (words.length > 20) {
      cleanTitle = words.slice(0, 20).join(" ") + "...";
    }
    
    btn.textContent = cleanTitle;
    btn.addEventListener("click", () => {
      const idx = playlist.indexOf(track);
      setAudioTrack(idx, true);
      searchPopup.classList.add("hidden");
    });
    searchResults.appendChild(btn);
  });
}
