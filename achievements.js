// Achievement system
const ACHIEVEMENTS = {
    "first_visit": {
        name: "Nováček",
        description: "První návštěva na portálu",
        icon: "fa-star",
        color: "text-yellow-500"
    },
    "frequent_visitor": {
        name: "Pravidelný návštěvník",
        description: "10 návštěv za měsíc",
        icon: "fa-clock-rotate-left",
        color: "text-blue-500",
        threshold: 10,
        period: "monthly"
    },
    "power_user": {
        name: "Power User",
        description: "50 návštěv za měsíc",
        icon: "fa-rocket",
        color: "text-purple-500",
        threshold: 50,
        period: "monthly"
    },
    "super_fan": {
        name: "Super Fan",
        description: "100 návštěv za měsíc",
        icon: "fa-award",
        color: "text-gold",
        threshold: 100,
        period: "monthly"
    }
};

let achievementsEnabled = false;

// Hidden toggle for achievements
function toggleAchievements() {
    achievementsEnabled = !achievementsEnabled;
    localStorage.setItem('achievementsEnabled', achievementsEnabled);
    
    if (achievementsEnabled) {
        checkAchievements();
        showAchievements();
    } else {
        hideAchievements();
    }
}

// Check if user has earned achievements
async function checkAchievements() {
    try {
        const response = await fetch('/api/visitor-stats');
        const stats = await response.json();
        
        // Check for monthly achievements
        Object.values(ACHIEVEMENTS).forEach(achievement => {
            if (achievement.period === "monthly" && stats.monthly_visits >= achievement.threshold) {
                showAchievementToast(achievement);
            }
        });
        
        // First visit achievement
        if (stats.total_visits === 1) {
            showAchievementToast(ACHIEVEMENTS.first_visit);
        }
    } catch (error) {
        console.error('Error checking achievements:', error);
    }
}

// Show achievement toast
function showAchievementToast(achievement) {
    const toast = document.createElement('div');
    toast.className = `fixed bottom-4 right-4 bg-white rounded-lg shadow-lg p-4 w-64 flex items-center ${achievement.color}`;
    
    toast.innerHTML = `
        <div class="flex-1">
            <h3 class="font-bold text-lg">${achievement.name}</h3>
            <p class="text-gray-600">${achievement.description}</p>
        </div>
        <i class="fas ${achievement.icon} text-2xl ml-4"></i>
    `;
    
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.remove();
    }, 5000);
}

// Show achievements display
function showAchievements() {
    const achievementsDisplay = document.getElementById('achievementsDisplay');
    if (achievementsDisplay) {
        achievementsDisplay.style.display = 'block';
    }
}

// Hide achievements display
function hideAchievements() {
    const achievementsDisplay = document.getElementById('achievementsDisplay');
    if (achievementsDisplay) {
        achievementsDisplay.style.display = 'none';
    }
}

// Initialize achievements
function initializeAchievements() {
    // Check if achievements were enabled before
    achievementsEnabled = JSON.parse(localStorage.getItem('achievementsEnabled') || 'false');
    
    // Add hidden toggle button
    const hiddenToggle = document.createElement('button');
    hiddenToggle.className = 'hidden';
    hiddenToggle.style.cssText = `
        position: fixed;
        bottom: -100px;
        right: -100px;
        width: 50px;
        height: 50px;
        background: transparent;
        border: none;
        cursor: pointer;
        transition: transform 0.3s;
    `;
    
    hiddenToggle.onclick = () => {
        toggleAchievements();
        hiddenToggle.style.transform = 'translate(-50px, -50px)';
        setTimeout(() => {
            hiddenToggle.style.transform = '';
        }, 1000);
    };
    
    document.body.appendChild(hiddenToggle);
    
    // Check achievements when enabled
    if (achievementsEnabled) {
        checkAchievements();
        showAchievements();
    }
}

// Add celebration animation
function celebrate() {
    const confetti = document.createElement('div');
    confetti.className = 'absolute inset-0 pointer-events-none';
    confetti.innerHTML = `
        <div class="absolute inset-0 overflow-hidden">
            <div class="absolute inset-0 flex items-center justify-center">
                <div class="w-64 h-64 rounded-full bg-gradient-to-r from-purple-500 to-pink-500 opacity-25"></div>
            </div>
            <div class="absolute inset-0 flex items-center justify-center">
                <div class="w-48 h-48 rounded-full bg-gradient-to-r from-blue-500 to-cyan-500 opacity-25"></div>
            </div>
        </div>
    `;
    
    document.body.appendChild(confetti);
    
    setTimeout(() => {
        confetti.remove();
    }, 3000);
}

// Initialize when page loads
document.addEventListener('DOMContentLoaded', initializeAchievements);
