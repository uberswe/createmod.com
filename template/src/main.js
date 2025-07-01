import '@tabler/core/dist/js/tabler.min.js';
import * as bootstrap from 'bootstrap';
window.bootstrap = bootstrap;

const lazyImages = document.querySelectorAll(".lazy");

const options = {
    root: null, // Use the viewport as the root
    rootMargin: "0px",
    threshold: 0.1 // Specify the threshold for intersection
};

const handleIntersection = (entries, observer) => {
    entries.forEach((entry) => {
        if (entry.isIntersecting) {
            entry.target.src = entry.target.getAttribute("data-src");
            observer.unobserve(entry.target);
        }
    });
};

const observer = new IntersectionObserver(handleIntersection, options);

lazyImages.forEach((image) => {
    observer.observe(image);
});