document.addEventListener("DOMContentLoaded", () => {
    
    // Smoothly animates the SVG rings
    window.triggerMacroRings = function() {
        const rings = document.querySelectorAll('.macro-ring');
        
        rings.forEach((ring, index) => {
            const targetPct = parseFloat(ring.getAttribute('data-target')) || 0;
            
            const radius = ring.r.baseVal.value; 
            const circumference = 2 * Math.PI * radius; 
            
            ring.style.strokeDasharray = `${circumference} ${circumference}`;
            
            const offset = circumference - (targetPct / 100) * circumference;
            ring.style.strokeDashoffset = circumference;
            
            setTimeout(() => {
                ring.style.transition = 'stroke-dashoffset 1.5s cubic-bezier(0.16, 1, 0.3, 1)';
                ring.style.strokeDashoffset = offset;
            }, 100 + (index * 100)); 
        });
    };

    // Trigger on first load
    setTimeout(window.triggerMacroRings, 100);

    // HTMX listener: Trigger again if user adds a meal or edits targets!
    document.body.addEventListener('htmx:afterSettle', function(event) {
        if (event.detail.target.id === 'daily-breakdown') {
            window.triggerMacroRings();
        }
    });
});