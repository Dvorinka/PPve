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
        
        // Check for monthly achievements
        Object.values(ACHIEVEMENTS).forEach(achievement => {
            if (achievement.period === "monthly" && stats.monthly_visits >= achievement.threshold) {
                unlockAchievement(achievement);
            }
        });
        
        // First visit achievement
        if (stats.total_visits === 1) {
            unlockAchievement(ACHIEVEMENTS.first_visit);
        }

        // Apply highest unlocked achievement theme
        const unlocked = Array.from(unlockedAchievements);
        if (unlocked.length > 0) {
            const highestAchievement = unlocked[unlocked.length - 1];
            currentTheme = ACHIEVEMENTS[highestAchievement].theme;
            applyTheme();
        }
    } catch (error) {
        console.error('Error checking achievements:', error);
    }
}

// Unlock achievement and show toast
function unlockAchievement(achievement) {
    const achievementId = Object.keys(ACHIEVEMENTS).find(key => 
        ACHIEVEMENTS[key].name === achievement.name
    );
    
    if (!unlockedAchievements.has(achievementId)) {
        unlockedAchievements.add(achievementId);
        showAchievementToast(achievement);
    }
}

// Show only unlocked achievements
function showAchievements() {
    const achievementsDisplay = document.getElementById('achievementsDisplay');
    if (achievementsDisplay) {
        // Clear existing achievements
        achievementsDisplay.innerHTML = '';
        
        // Show only unlocked achievements
        Array.from(unlockedAchievements).forEach(achievementId => {
            const achievement = ACHIEVEMENTS[achievementId];
            const achievementItem = document.createElement('div');
            achievementItem.className = 'achievement-item flex items-center p-3 rounded-lg mb-2';
            achievementItem.style.backgroundColor = achievement.theme.backgroundColor;
            achievementItem.style.color = achievement.theme.textColor;
            
            achievementItem.innerHTML = `
                <i class="fas ${achievement.icon} achievement-icon ${achievement.color}"></i>
                <div>
                    <h4 class="font-bold">${achievement.name}</h4>
                    <p class="text-sm">${achievement.description}</p>
                </div>
            `;
            
            achievementsDisplay.appendChild(achievementItem);
        });
        
        achievementsDisplay.style.display = 'block';
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
                toggleAchievements();
                
                // Reset cheat code
                cheatCodeIndex = 0;
                
                // Remove achievement toast after 3 seconds
                setTimeout(() => {
                    achievementToast.remove();
                }, 3000);
            }
        } else {
            // Reset cheat code if wrong key pressed
            cheatCodeIndex = 0;
        }
    });
    
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
