let currentSlide = 0;

function moveCarousel(direction) {
    const track = document.querySelector('.carousel-track');
    const slides = document.querySelectorAll('.carousel-slide');
    if (!track || slides.length === 0) return;

    const totalSlides = slides.length;
    currentSlide = (currentSlide + direction + totalSlides) % totalSlides;
    
    // Smooth translation
    track.style.transform = `translateX(-${currentSlide * 100}%)`;
}

// Auto-advance carousel every 5 seconds
setInterval(() => {
    const track = document.querySelector('.carousel-track');
    if (track) {
        moveCarousel(1);
    }
}, 5000);
