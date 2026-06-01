// Accepts the button element so we can animate it instantly
function playPowerSound(btnElement) {
    // 1. Play the audio
    let audio = new Audio('assets/power.mp3');
    audio.volume = 0.5;
    audio.play().catch(err => console.log("Audio play blocked by browser"));

    // 2. Instantly transition the button UI to the Tick Mark
    if (btnElement) {
        btnElement.classList.add('border-blue-500', 'bg-blue-500/20', 'shadow-[0_0_40px_rgba(59,130,246,0.3)]');
        
        const defaultUI = document.getElementById('creatine-default');
        const successUI = document.getElementById('creatine-success');
        
        if (defaultUI) defaultUI.classList.add('opacity-0', 'scale-150');
        if (successUI) {
            successUI.classList.remove('opacity-0', 'scale-50');
            successUI.classList.add('opacity-100', 'scale-100');
        }
    }

    // 3. Trigger the Full-Page CSS Hydro-Surge Glow
    document.body.classList.add('creatine-surge-active');
    setTimeout(() => {
        document.body.classList.remove('creatine-surge-active');
    }, 1000);
}

// Keep your Coin Flash listener below...