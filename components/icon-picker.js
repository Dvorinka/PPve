class IconPicker {
    constructor() {
        this.modal = null;
        this.container = null;
        this.searchInput = null;
        this.iconList = null;
        this.isInitialized = false;
        this.isModalOpen = false;
        this.icons = [];
        this.currentSearch = '';
    }

    init() {
        if (this.isInitialized) return;

        // Create modal structure
        this.modal = document.createElement('div');
        this.modal.className = 'icon-picker-modal fixed inset-0 bg-black bg-opacity-50 z-50 hidden';
        this.modal.innerHTML = `
            <div class="fixed inset-0 flex items-center justify-center p-4">
                <div class="bg-white rounded-xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
                    <div class="flex flex-col h-full">
                        <div class="border-b border-gray-200">
                            <div class="flex justify-between items-center p-6">
                                <h3 class="text-xl font-semibold text-gray-900">Vyberte ikonu</h3>
                                <button class="icon-picker-close text-gray-400 hover:text-gray-500 p-2">
                                    <i class="fas fa-times"></i>
                                </button>
                            </div>
                        </div>
                        <div class="border-b border-gray-200">
                            <div class="p-6">
                                <input 
                                    type="text" 
                                    class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 icon-picker-search"
                                    placeholder="Hledat ikony..."
                                    autocomplete="off"
                                >
                            </div>
                        </div>
                        <div class="flex-1 overflow-y-auto">
                            <div class="p-6 grid grid-cols-6 gap-4 icon-picker-list">
                                <!-- Icons will be populated here -->
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;

        // Get references to elements
        this.container = this.modal.querySelector('.icon-picker-modal > div');
        this.searchInput = this.modal.querySelector('.icon-picker-search');
        this.iconList = this.modal.querySelector('.icon-picker-list');
        this.closeButton = this.modal.querySelector('.icon-picker-close');

        // Add to body
        document.body.appendChild(this.modal);

        // Initialize icons
        this.initializeIcons();

        // Setup event listeners
        this.setupEventListeners();

        this.isInitialized = true;
    }

    initializeIcons() {
        // Load icons from categories
        const categories = {
            'Doprava': ['car', 'car-side', 'truck', 'bus', 'bicycle', 'motorcycle', 'plane', 'plane-departure', 'ship', 'subway', 'train', 'train-subway', 'walking', 'gas-pump', 'map-marker-alt', 'route'],
            'Jídlo a nápoje': ['utensils', 'hamburger', 'pizza-slice', 'ice-cream', 'coffee', 'mug-hot', 'beer', 'wine-glass', 'wine-bottle', 'wine-glass-alt', 'wine-bottle-alt', 'apple-alt', 'bread-slice', 'cheese', 'drumstick-bite', 'egg', 'fish', 'hotdog', 'ice-cream', 'lemon', 'pepper-hot', 'shrimp', 'stroopwafel'],
            'Nástroje': ['tools', 'wrench', 'screwdriver', 'hammer', 'toolbox', 'ruler', 'ruler-combined', 'ruler-horizontal', 'ruler-vertical', 'screwdriver-wrench', 'screwdriver', 'hammer', 'paint-roller', 'paint-brush', 'pencil-ruler', 'ruler', 'screwdriver', 'toolbox', 'wrench'],
            'Kancelář': ['briefcase', 'folder', 'folder-open', 'file', 'file-alt', 'file-archive', 'file-audio', 'file-code', 'file-excel', 'file-image', 'file-pdf', 'file-word', 'file-powerpoint', 'file-signature', 'file-upload', 'file-download', 'file-export', 'file-import', 'file-invoice', 'file-invoice-dollar', 'file-medical', 'file-prescription']
        };

        // Convert to array format
        this.icons = Object.entries(categories).map(([category, icons]) => ({
            category,
            icons: icons.map(icon => ({
                name: icon,
                displayName: icon.replace(/-/g, ' '),
                className: `fa-${icon}`
            }))
        }));
    }

    renderIcons(searchTerm = '') {
        searchTerm = searchTerm.toLowerCase();
        this.currentSearch = searchTerm;

        // Clear existing content
        this.iconList.innerHTML = '';

        // Add icons
        this.icons.forEach(category => {
            const filteredIcons = category.icons.filter(icon => 
                icon.name.toLowerCase().includes(searchTerm) || 
                category.category.toLowerCase().includes(searchTerm)
            );

            if (filteredIcons.length > 0) {
                // Add category header
                const categoryHeader = document.createElement('div');
                categoryHeader.className = 'icon-category';
                categoryHeader.textContent = category.category;
                this.iconList.appendChild(categoryHeader);

                // Add icons
                filteredIcons.forEach(icon => {
                    const iconElement = document.createElement('div');
                    iconElement.className = 'icon-option';
                    iconElement.setAttribute('data-icon', icon.className);
                    iconElement.title = icon.displayName;
                    iconElement.innerHTML = `
                        <i class="fas ${icon.className} text-xl hover:text-blue-500 transition-colors duration-200"></i>
                        <span class="icon-name text-sm text-gray-600">${icon.displayName}</span>
                    `;
                    this.iconList.appendChild(iconElement);
                });
            }
        });

        // Add no results message if needed
        if (this.iconList.children.length === 0) {
            const noResults = document.createElement('div');
            noResults.className = 'col-span-full text-center py-8 text-gray-500';
            noResults.innerHTML = `
                <i class="fas fa-search mb-2 text-2xl"></i>
                <p>Žádné ikony nenalezeny</p>
            `;
            this.iconList.appendChild(noResults);
        }
    }

    setupEventListeners() {
        // Close button
        this.closeButton.addEventListener('click', () => this.close());

        // Search input
        this.searchInput.addEventListener('input', (e) => {
            const searchTerm = e.target.value.toLowerCase();
            this.renderIcons(searchTerm);
        });

        // Document click - close when clicking outside
        document.addEventListener('click', (e) => {
            if (this.isModalOpen && !this.container.contains(e.target)) {
                this.close();
            }
        });

        // Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.isModalOpen) {
                this.close();
            }
        });

        // Icon selection
        this.iconList.addEventListener('click', (e) => {
            const iconOption = e.target.closest('.icon-option');
            if (iconOption && iconOption.dataset.icon) {
                this.selectIcon(iconOption.dataset.icon);
            }
        });
    }

    open() {
        if (!this.isInitialized) this.init();
        if (this.isModalOpen) return;

        this.isModalOpen = true;
        this.modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';

        // Focus search input
        requestAnimationFrame(() => {
            this.searchInput.focus();
            this.searchInput.value = '';
            this.renderIcons('');
        });
    }

    close() {
        if (!this.isModalOpen) return;

        this.isModalOpen = false;
        this.modal.style.display = 'none';
        document.body.style.overflow = '';

        // Clear search
        this.searchInput.value = '';
        this.renderIcons('');
    }

    selectIcon(iconClass) {
        // Get target elements
        const selectedIcon = document.getElementById('selectedIcon');
        const customIconPreview = document.getElementById('customIconPreview');
        const iconPreview = document.getElementById('iconPreview');
        const appIcon = document.getElementById('appIcon');
        const customIconInput = document.getElementById('customIconInput');

        // Update selected icon
        if (selectedIcon) {
            selectedIcon.className = `fas ${iconClass} text-2xl text-gray-400`;
            selectedIcon.classList.remove('hidden');
        }

        // Hide custom icon preview
        if (customIconPreview) {
            customIconPreview.src = '';
            customIconPreview.classList.add('hidden');
        }

        // Update app icon
        if (appIcon) {
            appIcon.value = iconClass;
            appIcon.dispatchEvent(new Event('change'));
        }

        // Reset custom icon input
        if (customIconInput) {
            customIconInput.value = '';
            customIconInput.dispatchEvent(new Event('change'));
        }

        // Update preview
        if (iconPreview) {
            const colors = ['blue', 'green', 'red', 'yellow', 'indigo', 'purple', 'pink', 'gray'];
            const randomColor = colors[Math.floor(Math.random() * colors.length)];
            iconPreview.className = `mt-2 flex items-center justify-center w-16 h-16 bg-${randomColor}-100 rounded-md overflow-hidden`;
            iconPreview.innerHTML = `<i class="fas ${iconClass} text-2xl text-gray-400"></i>`;
        }

        // Close modal
        this.close();
    }

    static getInstance() {
        if (!window.iconPickerInstance) {
            window.iconPickerInstance = new IconPicker();
        }
        return window.iconPickerInstance;
    }
}
