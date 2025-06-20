// Achievement system
const ACHIEVEMENTS = {
    "first_visit": {
        name: "Nováček",
        description: "První návštěva na portálu",
        icon: "fa-star",
        color: "text-yellow-500",
        theme: {
            backgroundColor: "bg-yellow-50",
            textColor: "text-yellow-700",
            borderColor: "border-yellow-200",
            hoverColor: "hover:bg-yellow-100"
        }
    },
    "frequent_visitor": {
        name: "Pravidelný návštěvník",
        description: "10 návštěv za měsíc",
        icon: "fa-clock-rotate-left",
        color: "text-blue-500",
        threshold: 10,
        period: "monthly",
        theme: {
            backgroundColor: "bg-blue-50",
            textColor: "text-blue-700",
            borderColor: "border-blue-200",
            hoverColor: "hover:bg-blue-100"
        }
    },
    "power_user": {
        name: "Power User",
        description: "50 návštěv za měsíc",
        icon: "fa-rocket",
        color: "text-purple-500",
        threshold: 50,
        period: "monthly",
        theme: {
            backgroundColor: "bg-purple-50",
            textColor: "text-purple-700",
            borderColor: "border-purple-200",
            hoverColor: "hover:bg-purple-100"
        }
    },
    "super_fan": {
        name: "Super Fan",
        description: "100 návštěv za měsíc",
        icon: "fa-award",
        color: "text-gold",
        threshold: 100,
        period: "monthly",
        theme: {
            backgroundColor: "bg-yellow-50",
            textColor: "text-yellow-700",
            borderColor: "border-yellow-200",
            hoverColor: "hover:bg-yellow-100"
        }
    }
};

// Track unlocked achievements
let unlockedAchievements = new Set();

// Store current theme
let currentTheme = {
    backgroundColor: "bg-white",
    textColor: "text-gray-800",
    borderColor: "border-gray-200",
    hoverColor: "hover:bg-gray-50"
};

// Apply theme to all cards
function applyTheme() {
    const cards = document.querySelectorAll('.card');
    cards.forEach(card => {
        card.className = card.className.split(' ').filter(cls => !cls.startsWith('bg-') && !cls.startsWith('text-') && !cls.startsWith('border-') && !cls.startsWith('hover:')).join(' ');
        card.className += ` ${currentTheme.backgroundColor} ${currentTheme.textColor} ${currentTheme.borderColor} ${currentTheme.hoverColor}`;
    });
}

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
        const visitorId = getCookie('visitor_id');
        
        // First visit achievement
        if (!localStorage.getItem('first_visit_' + visitorId)) {
            unlockAchievement('first_visit');
            localStorage.setItem('first_visit_' + visitorId, 'true');
        }
        
        // Get visitor's stats
        const visitor = stats.unique_visitors[visitorId];
        if (!visitor) return;
        
        // Monthly achievements
        const monthlyVisits = stats.monthly_visits;
        if (monthlyVisits >= ACHIEVEMENTS.frequent_visitor.threshold && 
            !localStorage.getItem('frequent_visitor_' + visitorId)) {
            unlockAchievement('frequent_visitor');
            localStorage.setItem('frequent_visitor_' + visitorId, 'true');
        }
        
        if (monthlyVisits >= ACHIEVEMENTS.power_user.threshold && 
            !localStorage.getItem('power_user_' + visitorId)) {
            unlockAchievement('power_user');
            localStorage.setItem('power_user_' + visitorId, 'true');
        }
        
        if (monthlyVisits >= ACHIEVEMENTS.super_fan.threshold && 
            !localStorage.getItem('super_fan_' + visitorId)) {
            unlockAchievement('super_fan');
            localStorage.setItem('super_fan_' + visitorId, 'true');
        }
    } catch (error) {
        console.error('Error checking achievements:', error);
    }
}

// Unlock achievement and show toast
function unlockAchievement(achievement) {
    unlockedAchievements.add(achievement);
    showAchievementToast(achievement);
    showAchievements();
}

// Show achievement toast
function showAchievementToast(achievement) {
    const toast = document.createElement('div');
    toast.className = 'fixed bottom-4 right-4 bg-white rounded-lg shadow-lg p-4 w-64 flex items-center text-green-500';
    toast.innerHTML = `
        <div class="flex-1">
            <h3 class="font-bold text-lg">Achievement Unlocked!</h3>
            <p class="text-gray-600">${ACHIEVEMENTS[achievement].name}</p>
        </div>
        <i class="fas fa-trophy text-2xl ml-4"></i>
    `;
    document.body.appendChild(toast);
    
    // Remove toast after 3 seconds
    setTimeout(() => {
        toast.remove();
    }, 3000);
}

// Show only unlocked achievements
function showAchievements() {
    const achievementsDisplay = document.getElementById('achievementsDisplay');
    const achievementItems = achievementsDisplay.querySelectorAll('.achievement-item');
    
    achievementItems.forEach(item => {
        const achievementId = item.querySelector('.achievement-icon').classList[1];
        const achievement = ACHIEVEMENTS[achievementId.replace('achievement-icon-', '')];
        if (achievement && unlockedAchievements.has(achievementId.replace('achievement-icon-', ''))) {
            item.style.display = 'block';
            // Apply achievement theme
            Object.entries(achievement.theme).forEach(([key, value]) => {
                item.classList.add(value);
            });
        } else {
            item.style.display = 'none';
        }
    });
    
    // Show achievements display if any achievements are unlocked
    if (unlockedAchievements.size > 0) {
        achievementsDisplay.classList.remove('hidden');
    } else {
        achievementsDisplay.classList.add('hidden');
    }
    
    // Apply highest unlocked achievement theme to the page
    applyHighestAchievementTheme();
}

// Apply highest unlocked achievement theme
function applyHighestAchievementTheme() {
    const unlocked = Array.from(unlockedAchievements);
    if (unlocked.length === 0) return;
    
    // Sort achievements by threshold to find the highest
    const sortedAchievements = Object.entries(ACHIEVEMENTS)
        .filter(([id]) => unlocked.includes(id))
        .sort(([, a], [, b]) => b.threshold - a.threshold);
    
    const highest = sortedAchievements[0][1];
    
    // Apply theme to body
    const body = document.body;
    // Remove existing theme classes
    ['bg-', 'text-', 'border-', 'hover:'].forEach(prefix => {
        const classes = Array.from(body.classList).filter(cls => !cls.startsWith(prefix));
        body.className = classes.join(' ');
    });
    
    // Add new theme classes
    Object.entries(highest.theme).forEach(([key, value]) => {
        body.classList.add(value);
    });
}

// Hide achievements display
function hideAchievements() {
    const achievementsDisplay = document.getElementById('achievementsDisplay');
    if (achievementsDisplay) {
        achievementsDisplay.style.display = 'none';
    }
}

// Doom-style cheat code toggle
const CHEAT_CODE = 'IDKFA';
let cheatCodeIndex = 0;
let cheatCodeTimer;

// Initialize achievements
function initializeAchievements() {
    // Check if achievements were enabled before
    achievementsEnabled = JSON.parse(localStorage.getItem('achievementsEnabled') || 'false');
    
    // Add cheat code input listener
    document.addEventListener('keydown', (e) => {
        // Reset timer if no key was pressed in 2 seconds
        if (cheatCodeTimer) {
            clearTimeout(cheatCodeTimer);
        }
        
        cheatCodeTimer = setTimeout(() => {
            cheatCodeIndex = 0;
        }, 2000);
        
        // Check if key matches current position in cheat code
        if (e.key.toUpperCase() === CHEAT_CODE[cheatCodeIndex]) {
            cheatCodeIndex++;
            
            // If complete code entered
            if (cheatCodeIndex === CHEAT_CODE.length) {
                // Show achievement unlocked animation
                const achievementToast = document.createElement('div');
                achievementToast.className = 'fixed bottom-4 right-4 bg-white rounded-lg shadow-lg p-4 w-64 flex items-center text-green-500';
                achievementToast.innerHTML = `
                    <div class="flex-1">
                        <h3 class="font-bold text-lg">Achievement Unlocked!</h3>
                        <p class="text-gray-600">"IDKFA" cheat code activated</p>
                    </div>
                    <i class="fas fa-trophy text-2xl ml-4"></i>
                `;
                document.body.appendChild(achievementToast);
                
                // Enable achievements
                achievementsEnabled = true;
                localStorage.setItem('achievementsEnabled', true);
                
                // Remove toast after 3 seconds
                setTimeout(() => {
                    achievementToast.remove();
                }, 3000);
            }
        } else {
            // Reset cheat code if wrong key pressed
            cheatCodeIndex = 0;
        }
    });
    
    // Initialize achievements if enabled
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
