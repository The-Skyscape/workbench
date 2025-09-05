// Clock functionality for the navbar
function updateClock() {
    const clockElement = document.getElementById('navbar-clock');
    if (!clockElement) return;
    
    const now = new Date();
    const hours = now.getHours();
    const minutes = now.getMinutes();
    const seconds = now.getSeconds();
    const ampm = hours >= 12 ? 'PM' : 'AM';
    
    const h = hours % 12 || 12;
    const m = minutes < 10 ? '0' + minutes : minutes;
    const s = seconds < 10 ? '0' + seconds : seconds;
    const hStr = h < 10 ? '0' + h : h;
    
    const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    
    const day = days[now.getDay()];
    const month = months[now.getMonth()];
    const date = now.getDate();
    
    const timeStr = `${day}, ${month} ${date} • ${hStr}:${m}:${s} ${ampm}`;
    clockElement.textContent = timeStr;
}

// Start clock when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
        updateClock();
        setInterval(updateClock, 1000);
    });
} else {
    updateClock();
    setInterval(updateClock, 1000);
}