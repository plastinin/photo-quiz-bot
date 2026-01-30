// DOM Elements
const setupScreen = document.getElementById('setupScreen');
const gameScreen = document.getElementById('gameScreen');
const gameOverScreen = document.getElementById('gameOverScreen');

const playersForm = document.getElementById('playersForm');
const addPlayerBtn = document.getElementById('addPlayerBtn');
const createSessionBtn = document.getElementById('createSessionBtn');

const currentPlayerBanner = document.getElementById('currentPlayerBanner');
const currentPlayerName = document.getElementById('currentPlayerName');

const moreBtn = document.getElementById('moreBtn');
const answerBtn = document.getElementById('answerBtn');
const newGameBtn = document.getElementById('newGameBtn');

const photo = document.getElementById('photo');
const photoLoader = document.getElementById('photoLoader');
const photoPrev = document.getElementById('photoPrev');
const photoNext = document.getElementById('photoNext');
const currentPhotoSpan = document.getElementById('currentPhoto');
const totalPhotosSpan = document.getElementById('totalPhotos');
const unlockedCountSpan = document.getElementById('unlockedCount');
const remainingSpan = document.getElementById('remaining');

const answerCard = document.getElementById('answerCard');
const answerText = document.getElementById('answerText');
const answerWaiting = document.getElementById('answerWaiting');

const scoreboardCard = document.getElementById('scoreboardCard');
const scoreboardList = document.getElementById('scoreboardList');
const finalScoreboard = document.getElementById('finalScoreboard');

const snackbar = document.getElementById('snackbar');

// State
let isLoading = false;
let playerCount = 1;
let answerShown = false; // Флаг: ответ уже показан
const MAX_PLAYERS = 10;

// Photo carousel state
let photoUrls = [];
let currentPhotoIndex = 0;
let totalPhotosCount = 1;
let unlockedPhotos = 1;

// Player inputs management
function addPlayerInput() {
    if (playerCount >= MAX_PLAYERS) {
        showSnackbar('Максимум 10 игроков');
        return;
    }
    
    playerCount++;
    
    const group = document.createElement('div');
    group.className = 'player-input-group';
    group.innerHTML = `
        <input type="text" class="input player-input" placeholder="Игрок ${playerCount}" maxlength="20">
        <button type="button" class="btn-icon btn-remove" title="Удалить">✕</button>
    `;
    
    playersForm.appendChild(group);
    group.querySelector('input').focus();
    
    group.querySelector('.btn-remove').addEventListener('click', function() {
        group.remove();
        playerCount--;
        updateRemoveButtons();
        updatePlaceholders();
    });
    
    updateRemoveButtons();
}

function updateRemoveButtons() {
    const removeButtons = playersForm.querySelectorAll('.btn-remove');
    removeButtons.forEach(btn => {
        if (playerCount <= 1) {
            btn.classList.add('hidden');
        } else {
            btn.classList.remove('hidden');
        }
    });
}

function updatePlaceholders() {
    const inputs = playersForm.querySelectorAll('.player-input');
    inputs.forEach((input, idx) => {
        input.placeholder = `Игрок ${idx + 1}`;
    });
}

// API calls
async function api(endpoint, method = 'GET', body = null) {
    try {
        const options = { method };
        if (body) {
            options.headers = { 'Content-Type': 'application/json' };
            options.body = JSON.stringify(body);
        }
        const response = await fetch(`/api/${endpoint}`, options);
        return await response.json();
    } catch (error) {
        console.error('API Error:', error);
        showSnackbar('Ошибка соединения с сервером');
        return null;
    }
}

// UI Functions
function showScreen(screen) {
    setupScreen.classList.add('hidden');
    gameScreen.classList.add('hidden');
    gameOverScreen.classList.add('hidden');
    screen.classList.remove('hidden');
}

function showSnackbar(message, duration = 3000) {
    snackbar.textContent = message;
    snackbar.classList.remove('hidden');
    setTimeout(() => {
        snackbar.classList.add('hidden');
    }, duration);
}

function setLoading(loading) {
    isLoading = loading;
    photoLoader.classList.toggle('hidden', !loading);
    moreBtn.disabled = loading;
    answerBtn.disabled = loading || answerShown;
}

// Photo carousel functions
function resetPhotoCarousel() {
    photoUrls = [];
    currentPhotoIndex = 0;
    unlockedPhotos = 0;
    updateCarouselNav();
}

function addPhotoToCarousel(url) {
    photoUrls.push(url);
    unlockedPhotos = photoUrls.length;
    currentPhotoIndex = unlockedPhotos - 1;
    updateCarouselNav();
}

function showPhotoAtIndex(index) {
    if (index < 0 || index >= photoUrls.length) return;
    
    currentPhotoIndex = index;
    
    setLoading(true);
    photo.onload = () => setLoading(false);
    photo.onerror = () => {
        setLoading(false);
        showSnackbar('Ошибка загрузки фото');
    };
    photo.src = photoUrls[index];
    
    updateCarouselNav();
}

function updateCarouselNav() {
    currentPhotoSpan.textContent = currentPhotoIndex + 1;
    unlockedCountSpan.textContent = unlockedPhotos;
    
    if (unlockedPhotos > 1) {
        photoPrev.classList.remove('hidden');
        photoNext.classList.remove('hidden');
    } else {
        photoPrev.classList.add('hidden');
        photoNext.classList.add('hidden');
    }
    
    photoPrev.disabled = currentPhotoIndex === 0;
    photoNext.disabled = currentPhotoIndex >= unlockedPhotos - 1;
    
    moreBtn.disabled = unlockedPhotos >= totalPhotosCount;
}

function prevPhoto() {
    if (currentPhotoIndex > 0) {
        showPhotoAtIndex(currentPhotoIndex - 1);
    }
}

function nextPhotoInCarousel() {
    if (currentPhotoIndex < unlockedPhotos - 1) {
        showPhotoAtIndex(currentPhotoIndex + 1);
    }
}

function updatePhoto(data) {
    if (data.photoUrl) {
        addPhotoToCarousel(data.photoUrl);
        
        setLoading(true);
        photo.onload = () => setLoading(false);
        photo.onerror = () => {
            setLoading(false);
            showSnackbar('Ошибка загрузки фото');
        };
        photo.src = data.photoUrl;
    }

    if (data.totalPhotos !== undefined) {
        totalPhotosCount = data.totalPhotos;
        totalPhotosSpan.textContent = data.totalPhotos;
    }

    updateCarouselNav();
    
    // Скрываем ответ и сбрасываем флаг
    answerCard.classList.add('hidden');
    answerShown = false;
    answerBtn.disabled = false;
}

function updateCurrentPlayer(player) {
    if (player) {
        currentPlayerName.textContent = player.name;
        currentPlayerBanner.classList.remove('hidden');
    }
}

function updateScoreboard(scoreboard) {
    if (!scoreboard || scoreboard.length === 0) {
        scoreboardCard.classList.add('hidden');
        return;
    }
    
    scoreboardCard.classList.remove('hidden');
    
    scoreboardList.innerHTML = scoreboard.map((player, idx) => {
        const position = idx + 1;
        const positionClass = position <= 3 ? `scoreboard__position--${position}` : '';
        const positionIcon = position === 1 ? '🥇' : position === 2 ? '🥈' : position === 3 ? '🥉' : position;
        const currentClass = player.isCurrentPlayer ? 'scoreboard__item--current' : '';
        const scoreDisplay = Number.isInteger(player.score) ? player.score : player.score.toFixed(1);
        
        return `
            <div class="scoreboard__item ${currentClass}">
                <div class="scoreboard__position ${positionClass}">${positionIcon}</div>
                <div class="scoreboard__name">${escapeHtml(player.name)}</div>
                <div class="scoreboard__score">${scoreDisplay} 🤑</div>
            </div>
        `;
    }).join('');
}

function updateFinalScoreboard(scoreboard) {
    if (!scoreboard || scoreboard.length === 0) return;
    
    finalScoreboard.innerHTML = scoreboard.map((player, idx) => {
        const position = idx + 1;
        const positionIcon = position === 1 ? '🥇' : position === 2 ? '🥈' : position === 3 ? '🥉' : position;
        const winnerBadge = position === 1 ? '<span class="winner-badge">ПОБЕДИТЕЛЬ</span>' : '';
        const scoreDisplay = Number.isInteger(player.score) ? player.score : player.score.toFixed(1);
        
        return `
            <div class="scoreboard__item">
                <div class="scoreboard__position scoreboard__position--${position}">${positionIcon}</div>
                <div class="scoreboard__name">${escapeHtml(player.name)}${winnerBadge}</div>
                <div class="scoreboard__score">${scoreDisplay} 🤑</div>
            </div>
        `;
    }).join('');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

async function updateStats() {
    const data = await api('stats');
    if (data) {
        remainingSpan.textContent = data.remaining;
    }
}

// Session & Game actions
async function createSession() {
    const inputs = playersForm.querySelectorAll('.player-input');
    const players = [];
    
    inputs.forEach(input => {
        const name = input.value.trim();
        if (name) {
            players.push(name);
        }
    });
    
    if (players.length === 0) {
        showSnackbar('Введите хотя бы одного игрока');
        return;
    }
    
    const data = await api('session/create', 'POST', { players });
    
    if (!data || !data.success) {
        showSnackbar(data?.message || 'Ошибка создания сессии');
        return;
    }
    
    await startGame();
}

async function startGame() {
    setLoading(true);
    resetPhotoCarousel();
    answerShown = false;
    
    const data = await api('start', 'POST');
    setLoading(false);

    if (!data) return;

    if (data.gameOver) {
        updateFinalScoreboard(data.scoreboard);
        showScreen(gameOverScreen);
        return;
    }

    if (data.success) {
        showScreen(gameScreen);
        updatePhoto(data);
        updateCurrentPlayer(data.currentPlayer);
        updateScoreboard(data.scoreboard);
        updateStats();
    } else {
        showSnackbar(data.message || 'Ошибка запуска игры');
    }
}

async function unlockNextPhoto() {
    if (isLoading) return;
    if (unlockedPhotos >= totalPhotosCount) {
        showSnackbar('Все фото уже открыты');
        return;
    }

    setLoading(true);
    const data = await api('next-photo', 'POST');
    setLoading(false);

    if (!data) return;

    if (data.success) {
        updatePhoto(data);
    } else {
        showSnackbar(data.message || 'Больше нет фото');
    }
}

async function showAnswer() {
    if (isLoading || answerShown) return;

    const data = await api('answer', 'POST');

    if (!data) return;

    if (data.success) {
        answerText.textContent = data.answer;
        answerCard.classList.remove('hidden');
        answerWaiting.classList.remove('hidden');
        
        // Блокируем кнопку ответа
        answerShown = true;
        answerBtn.disabled = true;
    } else {
        showSnackbar(data.message || 'Ошибка');
    }
}

async function loadNextRound() {
    setLoading(true);
    resetPhotoCarousel();
    answerShown = false;
    
    const data = await api('next-round', 'POST');
    setLoading(false);

    if (!data) return;

    if (data.gameOver) {
        updateFinalScoreboard(data.scoreboard);
        showScreen(gameOverScreen);
        updateStats();
        return;
    }

    if (data.success) {
        updatePhoto(data);
        updateCurrentPlayer(data.currentPlayer);
        updateScoreboard(data.scoreboard);
        answerWaiting.classList.add('hidden');
        updateStats();
    } else {
        showSnackbar(data.message || 'Ошибка');
    }
}

function newGame() {
    playersForm.innerHTML = `
        <div class="player-input-group">
            <input type="text" class="input player-input" placeholder="Игрок 1" maxlength="20">
            <button type="button" class="btn-icon btn-remove hidden" title="Удалить">✕</button>
        </div>
    `;
    playerCount = 1;
    answerShown = false;
    updateRemoveButtons();
    resetPhotoCarousel();
    
    showScreen(setupScreen);
}

// Polling for updates (scoreboard + auto next round)
let pollInterval = null;
let lastRound = 0;

function startPolling() {
    if (pollInterval) return;
    
    pollInterval = setInterval(async () => {
        if (gameScreen.classList.contains('hidden')) {
            stopPolling();
            return;
        }
        
        const data = await api('scoreboard');
        if (data && data.success) {
            updateScoreboard(data.scoreboard);
            
            // Проверяем, изменился ли раунд (значит админ выставил оценку)
            if (data.currentRound && data.currentRound > lastRound && answerShown) {
                lastRound = data.currentRound;
                // Автоматически загружаем следующий раунд
                await loadNextRound();
            } else if (data.currentRound) {
                lastRound = data.currentRound;
            }
        }
    }, 1500);
}

function stopPolling() {
    if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
    }
}

// Start polling when game screen is shown
const observer = new MutationObserver(() => {
    if (!gameScreen.classList.contains('hidden')) {
        startPolling();
    } else {
        stopPolling();
    }
});

observer.observe(gameScreen, { attributes: true, attributeFilter: ['class'] });

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (!setupScreen.classList.contains('hidden')) {
        if (e.code === 'Enter' && e.target.classList.contains('player-input')) {
            e.preventDefault();
            createSession();
        }
        return;
    }
    
    if (!gameOverScreen.classList.contains('hidden')) {
        if (e.code === 'Enter') {
            e.preventDefault();
            newGame();
        }
        return;
    }

    if (gameScreen.classList.contains('hidden')) return;

    switch (e.code) {
        case 'Space':
            e.preventDefault();
            unlockNextPhoto();
            break;
        case 'Enter':
            e.preventDefault();
            if (!answerShown) {
                showAnswer();
            }
            break;
        case 'ArrowRight':
            e.preventDefault();
            nextPhotoInCarousel();
            break;
        case 'ArrowLeft':
            e.preventDefault();
            prevPhoto();
            break;
    }
});

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    addPlayerBtn.addEventListener('click', addPlayerInput);
    createSessionBtn.addEventListener('click', createSession);
    moreBtn.addEventListener('click', unlockNextPhoto);
    answerBtn.addEventListener('click', showAnswer);
    newGameBtn.addEventListener('click', newGame);
    
    photoPrev.addEventListener('click', prevPhoto);
    photoNext.addEventListener('click', nextPhotoInCarousel);
    
    updateRemoveButtons();
    updateStats();
});