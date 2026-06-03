import { qs } from './utils.js';

let timerId = null;
export let config = { pomo: 25, short: 5, long: 15 };
let timeLeft = 0;
export let isRunning = false;
let endTime = 0;
export let timerMode = "pomo"; 
let audioCtx = null;
let sessionStats = JSON.parse(localStorage.getItem("study-session-stats") || '{"rounds":0, "breaks":0, "longBreaks":0}');

const timerDisplay = qs("#timer-display");
const playIcon = qs("#play-icon");
const pauseIcon = qs("#pause-icon");
const skipBreakBtn = qs("#skip-break-btn");
const countRounds = qs("#count-rounds");
const countBreaks = qs("#count-breaks");
const countLongBreaks = qs("#count-long-breaks");
const todoList = qs("#todo-list");

export async function fetchPomoSettings() {
  try {
    const res = await fetch("/api/settings/pomo");
    if (!res.ok) return;
    const data = await res.json();
    config = {
      pomo: Number(data.pomo) || 25,
      short: Number(data.short) || 5,
      long: Number(data.long) || 15,
    };
  } catch (_) {
    config = { pomo: 25, short: 5, long: 15 };
  }
}

function updateTimerButtons() {
  playIcon.classList.toggle("hidden", isRunning);
  pauseIcon.classList.toggle("hidden", !isRunning);
  
  if (timerMode !== "pomo" && isRunning) {
    skipBreakBtn.classList.remove("hidden");
  } else {
    skipBreakBtn.classList.add("hidden");
  }
}

function updateTimerDisplay() {
  const mins = Math.floor(Math.max(0, timeLeft) / 60);
  const secs = Math.max(0, timeLeft) % 60;
  const value = `${String(mins).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
  timerDisplay.textContent = value;
  document.title = `${value} - Study`;
}

function runTimer() {
  clearInterval(timerId);
  isRunning = true;
  updateTimerButtons();

  timerId = setInterval(() => {
    timeLeft = Math.max(0, Math.round((endTime - Date.now()) / 1000));
    updateTimerDisplay();
    if (timeLeft <= 0) {
      clearInterval(timerId);
      isRunning = false;
      
      if (timerMode === "pomo") sessionStats.rounds++;
      else if (timerMode === "short") sessionStats.breaks++;
      else if (timerMode === "long") sessionStats.longBreaks++;
      saveStats();

      playFinishTone();
      resetTimer();
    }
  }, 500);
}

export function toggleTimer() {
  if (isRunning) {
    clearInterval(timerId);
    isRunning = false;
    updateTimerButtons();
    return;
  }

  endTime = Date.now() + timeLeft * 1000;
  runTimer();
}

export function resetTimer() {
  clearInterval(timerId);
  let duration = config.pomo;
  if (timerMode === "short") duration = config.short;
  else if (timerMode === "long") duration = config.long;
  
  timeLeft = duration * 60;
  endTime = 0;
  isRunning = false;
  updateTimerButtons();
  updateTimerDisplay();
}

function saveStats() {
  localStorage.setItem("study-session-stats", JSON.stringify(sessionStats));
  updateStatsDisplay();
}

export function updateStatsDisplay() {
  if (countRounds) countRounds.textContent = sessionStats.rounds;
  if (countBreaks) countBreaks.textContent = sessionStats.breaks;
  if (countLongBreaks) countLongBreaks.textContent = sessionStats.longBreaks;
}

export function setTimerMode(mode) {
  timerMode = mode;
  resetTimer();
}

export function skipBreak() {
  timeLeft = 0;
  endTime = Date.now();
}

export function stopTimer() {
  if (isRunning) {
    clearInterval(timerId);
    isRunning = false;
  }
}

function playFinishTone() {
  try {
    if (!audioCtx) {
      const AudioContext = window.AudioContext || window.webkitAudioContext;
      audioCtx = new AudioContext();
    }
    const oscillator = audioCtx.createOscillator();
    const gain = audioCtx.createGain();
    oscillator.type = "sine";
    oscillator.frequency.value = 880;
    gain.gain.setValueAtTime(0.0001, audioCtx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.24, audioCtx.currentTime + 0.03);
    gain.gain.exponentialRampToValueAtTime(0.0001, audioCtx.currentTime + 0.7);
    oscillator.connect(gain);
    gain.connect(audioCtx.destination);
    oscillator.start();
    oscillator.stop(audioCtx.currentTime + 0.75);
  } catch (_) {
  }
}

export function renderTodo(task) {
  const li = document.createElement("li");
  li.className = "todo-item";
  if (task.completed) li.classList.add("done");

  const content = document.createElement("div");
  content.className = "todo-content";

  const checkbox = document.createElement("input");
  checkbox.type = "checkbox";
  checkbox.className = "todo-checkbox";
  checkbox.checked = Boolean(task.completed);

  const span = document.createElement("span");
  span.className = "todo-text";
  span.textContent = task.title;

  const remove = document.createElement("button");
  remove.type = "button";
  remove.className = "todo-delete";
  remove.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6L6 18M6 6l12 12"/></svg>';

  checkbox.addEventListener("change", async () => {
    li.classList.toggle("done", checkbox.checked);
    await fetch(`/api/focus/${task.id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ completed: checkbox.checked })
    });
  });
  remove.addEventListener("click", async () => {
    li.remove();
    await fetch(`/api/focus/${task.id}`, { method: "DELETE" });
  });

  content.append(checkbox, span);
  li.append(content, remove);
  todoList.appendChild(li);
}

export async function saveTodos(title) {
  await fetch("/api/focus", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title: title })
  });
  loadTodos();
}

export async function loadTodos() {
  const res = await fetch("/api/focus");
  if (!res.ok) return;
  const tasks = await res.json();
  todoList.innerHTML = "";
  if (tasks) tasks.forEach(renderTodo);
}
