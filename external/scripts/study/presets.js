import { qs, escapeName, readableName } from './utils.js';

let quotes = [];
const quoteDisplay = qs("#quote-display");
const videoGrid = qs("#video-grid");
const bgVideo = qs("#study-bg-video");

export async function loadUsername() {
  try {
    const res = await fetch("/api/user");
    if (!res.ok) return;
    const data = await res.json();
    qs("#study-username").textContent = data.username || "Study";
  } catch (_) {
    qs("#study-username").textContent = "Study";
  }
}

export async function loadQuotes() {
  try {
    const res = await fetch("assets/quotes/quotes.txt");
    const text = await res.text();
    quotes = text.split("\n").filter(q => q.trim().length > 0);
    showRandomQuote();
    setInterval(showRandomQuote, 30000); 
  } catch (_) {
    quotes = ["Focus on your goals."];
  }
}

function showRandomQuote() {
  if (quotes.length === 0) return;
  const idx = Math.floor(Math.random() * quotes.length);
  quoteDisplay.textContent = quotes[idx];
}

export async function loadVideos() {
  let videos = [];
  try {
    const res = await fetch("/api/resources/videos");
    const data = await res.json();
    videos = Array.isArray(data) ? data : [];
  } catch (_) {
    videos = [];
  }

  videoGrid.innerHTML = "";
  if (videos.length === 0) {
    const empty = document.createElement("p");
    empty.textContent = "No MP4 or WebM files found.";
    empty.style.color = "rgba(255,255,255,0.7)";
    videoGrid.appendChild(empty);
  } else {
    videos.forEach((name) => {
      const card = document.createElement("button");
      card.type = "button";
      card.className = "video-card";
      
      const label = document.createElement("span");
      label.textContent = readableName(name);

      card.append(label);
      card.addEventListener("click", () => {
        setBackground(name);
        qs("#preset-modal").classList.add("hidden");
      });
      videoGrid.appendChild(card);
    });
  }

  const saved = localStorage.getItem("study-bg-video");
  if (saved) {
    setBackground(saved);
  } else if (videos.length > 0) {
    setBackground(videos[0]);
  }
}

export function setBackground(name) {
  if (!name) return;
  const isUrl = name.startsWith("http");
  const url = isUrl ? name : `assets/videos/${escapeName(name)}`;
  if (bgVideo.src === url || bgVideo.src.endsWith(url)) return; 
  
  bgVideo.pause();
  bgVideo.src = url;
  bgVideo.loop = true; 
  bgVideo.playsInline = true;
  // Lowering playback quality hints if supported (though limited in standard <video>)
  bgVideo.load();
  bgVideo.play().catch(() => {});
  localStorage.setItem("study-bg-video", name);
}

export function initVideoLoop() {
  bgVideo.loop = true;
  
  // Use a throttled check instead of rapid timeupdate if we want extreme efficiency,
  // but standard loop attribute is actually very efficient. 
  // We'll keep a simple fallback and add Visibility API.
  
  bgVideo.addEventListener("ended", () => {
    bgVideo.currentTime = 0;
    bgVideo.play().catch(() => {});
  });

  // Page Visibility API: Stop video when tab is hidden to save RAM/CPU
  document.addEventListener("visibilitychange", () => {
    if (document.hidden) {
      bgVideo.pause();
    } else {
      bgVideo.play().catch(() => {});
    }
  });
}
