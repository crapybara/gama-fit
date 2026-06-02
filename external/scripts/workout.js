// Accepts the button element so we can animate it instantly
function playPowerSound(btnElement) {
    // 1. Play the audio
    let audio = new Audio('assets/power.mp3');
    audio.volume = 0.5;
    audio.play().catch(err => console.log("Audio play blocked by browser"));

    // 2. Instantly transition the button UI
    if (btnElement) {
        btnElement.classList.add('border-blue-500', 'bg-blue-500/20', 'shadow-[0_0_40px_rgba(59,130,246,0.3)]');
    }
}

// Keep your Coin Flash listener below...