import { qs, qsa } from './utils.js';
import { 
  toggleTimer, resetTimer, skipBreak, stopTimer,
  fetchPomoSettings, renderTodo, saveTodos, loadTodos,
  updateStatsDisplay, setTimerMode
} from './timer.js';
import { 
  toggleFullscreen, updateFullscreenIcons 
} from './fullscreen.js';
import { 
  loadUsername, loadQuotes, loadVideos, initVideoLoop
} from './presets.js';
import {
  loadMusicPresets, loadMusicPreset, toggleMusic,
  nextTrack, prevTrack, shufflePlaylist, handleSearch
} from './music.js';

function bindEvents() {
  const playPauseBtn = qs("#play-pause-btn");
  const stopBtn = qs("#stop-btn");
  const todoPopup = qs("#todo-popup");
  const todoInput = qs("#todo-input");
  const presetModal = qs("#preset-modal");
  const skipBreakBtn = qs("#skip-break-btn");
  const timerDisplay = qs("#timer-display");
  const timerSizeSlider = qs("#timer-size-slider");
  const musicSearchTrigger = qs("#music-search-trigger");
  const searchPopup = qs("#search-popup");
  const searchInput = qs("#search-input");
  const musicPresetSelect = qs("#music-preset-select");
  const audio = qs("#bg-audio");
  const volumeSlider = qs("#volume-slider");

  playPauseBtn.addEventListener("click", toggleTimer);
  stopBtn.addEventListener("click", resetTimer);

  qs("#todo-toggle-btn").addEventListener("click", () => todoPopup.classList.toggle("hidden"));
  qs("#todo-close-btn").addEventListener("click", () => todoPopup.classList.add("hidden"));
  todoInput.addEventListener("keydown", (event) => {
    if (event.key !== "Enter" || !todoInput.value.trim()) return;
    renderTodo({ text: todoInput.value.trim(), done: false });
    todoInput.value = "";
    saveTodos();
  });

  qs("#fullscreen-toggle").addEventListener("click", toggleFullscreen);
  document.addEventListener("fullscreenchange", updateFullscreenIcons);

  qs("#preset-open-btn").addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();
    presetModal.classList.remove("hidden");
  });
  qs("#preset-close-btn").addEventListener("click", () => {
    presetModal.classList.add("hidden");
    qs("#preset-close-btn").focus({ preventScroll: true });
  });
  presetModal.addEventListener("click", (e) => { 
    if(e.target === presetModal) {
      presetModal.classList.add("hidden");
    }
  });

  qsa(".tab-btn").forEach(btn => {
    btn.addEventListener("click", () => {
      qsa(".tab-btn").forEach(b => b.classList.remove("active"));
      btn.classList.add("active");
      qsa(".tab-content").forEach(c => c.classList.add("hidden"));
      qs("#" + btn.dataset.tab).classList.remove("hidden");
    });
  });

  qsa(".font-card").forEach(card => {
    card.addEventListener("click", () => {
      const font = card.dataset.font;
      timerDisplay.style.fontFamily = font + ", sans-serif";
      localStorage.setItem("study-timer-font", font);
    });
  });

  timerSizeSlider.addEventListener("input", () => {
    const size = timerSizeSlider.value;
    timerDisplay.style.fontSize = (size / 10) + "rem";
    localStorage.setItem("study-timer-size", size);
  });

  musicSearchTrigger.addEventListener("click", () => {
    searchPopup.classList.remove("hidden");
    searchInput.value = "";
    handleSearch();
    searchInput.focus();
  });
  qs("#search-close-btn").addEventListener("click", () => searchPopup.classList.add("hidden"));
  searchInput.addEventListener("input", handleSearch);

  qs("#music-toggle-btn").addEventListener("click", toggleMusic);
  qs("#music-next-btn").addEventListener("click", nextTrack);
  qs("#music-prev-btn").addEventListener("click", prevTrack);
  qs("#music-shuffle-btn").addEventListener("click", shufflePlaylist);

  qs("#resume-session-btn").addEventListener("click", () => setTimerMode("pomo"));
  qs("#take-break-btn").addEventListener("click", () => setTimerMode("short"));
  qs("#take-long-break-btn").addEventListener("click", () => setTimerMode("long"));

  skipBreakBtn.addEventListener("click", skipBreak);

  qs("#exit-study-orb").addEventListener("click", () => {
    stopTimer();
  });

  musicPresetSelect.addEventListener("change", () => loadMusicPreset(musicPresetSelect.value));
  audio.addEventListener("ended", nextTrack);

  const savedVolume = parseInt(localStorage.getItem("study-music-volume") || "55", 10);
  volumeSlider.value = String(Number.isFinite(savedVolume) ? savedVolume : 55);
  audio.volume = Number(volumeSlider.value) / 100;
  volumeSlider.addEventListener("input", () => {
    audio.volume = Number(volumeSlider.value) / 100;
    localStorage.setItem("study-music-volume", volumeSlider.value);
  });

  const savedFont = localStorage.getItem("study-timer-font");
  if(savedFont) timerDisplay.style.fontFamily = savedFont + ", sans-serif";
  const savedSize = localStorage.getItem("study-timer-size");
  if(savedSize) {
    timerSizeSlider.value = savedSize;
    timerDisplay.style.fontSize = (savedSize / 10) + "rem";
  }

  initVideoLoop();
}

async function init() {
  bindEvents();
  await Promise.all([fetchPomoSettings(), loadUsername(), loadVideos(), loadMusicPresets(), loadQuotes()]);
  loadTodos();
  updateStatsDisplay();
  resetTimer();
}

document.addEventListener("DOMContentLoaded", init);
