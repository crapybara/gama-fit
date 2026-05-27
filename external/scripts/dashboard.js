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