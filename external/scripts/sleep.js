document.addEventListener("DOMContentLoaded", () => {
    
    window.triggerSleepRing = function() {
        const rings = document.querySelectorAll('.sleep-ring');
        
        rings.forEach(ring => {
            const targetScore = parseInt(ring.getAttribute('data-target')) || 0;
            
            // SMART CALCULATOR: Dynamically grab the radius from the HTML
            // so we can change the size of the ring without breaking the math!
            const radius = ring.r.baseVal.value; 
            const circumference = 2 * Math.PI * radius; 
            
            // Set the max dash array dynamically
            ring.style.strokeDasharray = `${circumference} ${circumference}`;
            
            // Calculate the stroke offset based on percentage
            const offset = circumference - (targetScore / 100) * circumference;
            
            // Start it empty, then animate it!
            ring.style.strokeDashoffset = circumference;
            
            setTimeout(() => {
                ring.style.strokeDashoffset = offset;
            }, 100);
        });
    };

    window.triggerSleepRing();

    document.body.addEventListener('htmx:afterSettle', function(event) {
        if (event.detail.target.id === 'sleep-summary') {
            window.triggerSleepRing();
        }
    });
});