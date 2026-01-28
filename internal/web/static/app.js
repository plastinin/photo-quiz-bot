const startScreen = document.getElementById('startScreen');
const gameScreen = document.getElementById('gameScreen');
const gameOverScreen = document.getElementById('gameOverScreen');

const startBtn = document.getElementById('startBtn');
const moreBtn = document.getElementById('moreBtn');
const answerBtn = document.getElementById('answerBtn');
const nextBtn = document.getElementById('nextBtn');

const photo = document.getElementById('photo');
const photoLoader = document.getElementById('photoLoader');
const currentPhotoSpan = document.getElementById('currentPhoto');
const totalPhotosSpan = document.getElementById('totalPhotos');
const remainingSpan = document.getElementById('remaining');

const answerCard = document.getElementById('answerCard');
const answerText = document.getElementById('answerText');

const snackbar = document.getElementById('snackbar');

let isLoading = false;

// API calls
async function api(endpoint, method = 'GET') {
    try {
        const response = await fetch(`/api/${endpoint}`, { method });
        return await response.json();
    } catch (error) {
        console.error('API Error:', error);
        showSnackbar('Ошибка соединения с сервером');
        return null;
    }
}


function showScreen(screen) {
    startScreen.classList.add('hidden');
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
    startBtn.disabled = loading;
    moreBtn.disabled = loading;
    answerBtn.disabled = loading;
    nextBtn.disabled = loading;
}

function updatePhoto(data) {
    if (data.photoUrl) {
        setLoading(true);
        photo.onload = () => setLoading(false);
        photo.onerror = () => {
            setLoading(false);
            showSnackbar('Ошибка загрузки фото');
        };
        photo.src = data.photoUrl;
    }

    if (data.currentPhoto !== undefined) {
        currentPhotoSpan.textContent = data.currentPhoto;
    }
    if (data.totalPhotos !== undefined) {
        totalPhotosSpan.textContent = data.totalPhotos;
    }

    moreBtn.disabled = !data.hasMore;

    answerCard.classList.add('hidden');
}

async function updateStats() {
    const data = await api('stats');
    if (data) {
        remainingSpan.textContent = data.remaining;
    }
}

async function startGame() {
    setLoading(true);
    const data = await api('start', 'POST');
    setLoading(false);

    if (!data) return;

    if (data.gameOver) {
        showScreen(gameOverScreen);
        return;
    }

    if (data.success) {
        showScreen(gameScreen);
        updatePhoto(data);
        updateStats();
    } else {
        showSnackbar(data.message || 'Ошибка запуска игры');
    }
}

async function nextPhoto() {
    if (isLoading) return;

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
    if (isLoading) return;

    const data = await api('answer', 'POST');

    if (!data) return;

    if (data.success) {
        answerText.textContent = data.answer;
        answerCard.classList.remove('hidden');
    } else {
        showSnackbar(data.message || 'Ошибка');
    }
}

async function nextRound() {
    if (isLoading) return;

    setLoading(true);
    const data = await api('next-round', 'POST');
    setLoading(false);

    if (!data) return;

    if (data.gameOver) {
        showScreen(gameOverScreen);
        updateStats();
        return;
    }

    if (data.success) {
        updatePhoto(data);
        updateStats();
    } else {
        showSnackbar(data.message || 'Ошибка');
    }
}

startBtn.addEventListener('click', startGame);
moreBtn.addEventListener('click', nextPhoto);
answerBtn.addEventListener('click', showAnswer);
nextBtn.addEventListener('click', nextRound);

document.addEventListener('keydown', (e) => {
    if (gameScreen.classList.contains('hidden')) {
        if (e.code === 'Space' || e.code === 'Enter') {
            e.preventDefault();
            startGame();
        }
        return;
    }

    switch (e.code) {
        case 'Space':
            e.preventDefault();
            nextPhoto();
            break;
        case 'Enter':
            e.preventDefault();
            showAnswer();
            break;
        case 'ArrowRight':
            e.preventDefault();
            nextRound();
            break;
    }
});

// Initial stats load
updateStats();