import { qs } from './utils.js';

export async function toggleFullscreen() {
  if (!document.fullscreenElement) {
    try {
      await document.documentElement.requestFullscreen();
      if (window.innerWidth < 1024 && screen.orientation && screen.orientation.lock) {
        try {
          await screen.orientation.lock("landscape");
        } catch (e) {
          console.warn("Orientation lock failed:", e);
        }
      }
    } catch (err) {
      console.error(`Error attempting to enable full-screen mode: ${err.message}`);
    }
  } else {
    if (document.exitFullscreen) {
      document.exitFullscreen();
      if (screen.orientation && screen.orientation.unlock) {
        screen.orientation.unlock();
      }
    }
  }
}

export function updateFullscreenIcons() {
  const isFS = !!document.fullscreenElement;
  qs("#fullscreen-icon").classList.toggle("hidden", isFS);
  qs("#exit-fullscreen-icon").classList.toggle("hidden", !isFS);
}
