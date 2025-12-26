function isValidUrl(string) {
    let url;
    try {
        url = new URL(string);
    } catch (_) {
        return false;
    }
    return url.protocol === "http:" || url.protocol === "https:";
}

function shortenUrl() {
    const input = document.getElementById('urlInput');
    const url = input.value.trim();
    const errorMsg = document.getElementById('errorMsg');
    const btn = document.getElementById('shortenBtn');
    const resultSection = document.getElementById('resultSection');

    // Reset State
    errorMsg.classList.add('hidden');
    resultSection.classList.add('hidden');
    input.parentElement.classList.remove('border-red-500');

    // Validasi URL
    if (!isValidUrl(url)) {
        errorMsg.classList.remove('hidden');
        errorMsg.classList.add('flex');
        input.parentElement.classList.add('border-red-500');
        // Animasi shake untuk visual cue
        input.parentElement.animate([
            { transform: 'translateX(0)' },
            { transform: 'translateX(-5px)' },
            { transform: 'translateX(5px)' },
            { transform: 'translateX(0)' }
        ], { duration: 200 });
        return;
    }

    // Loading State
    const originalBtnContent = btn.innerHTML;
    btn.innerHTML = '<i class="fa-solid fa-circle-notch fa-spin"></i>';
    btn.disabled = true;
    btn.classList.add('opacity-80', 'cursor-not-allowed');

    // Simulasi API Call (Mode Demo - ganti dengan fetch API setelah backend siap)
    const startTime = performance.now();

    // TODO: Ganti setTimeout ini dengan fetch API setelah backend selesai
    // Contoh penggunaan:
    // fetch('/api/shorten', {
    //     method: 'POST',
    //     headers: { 'Content-Type': 'application/json' },
    //     body: JSON.stringify({ url: url })
    // })
    // .then(response => response.json())
    // .then(data => { ... })

    setTimeout(() => {
        const endTime = performance.now();
        const processTime = Math.round(endTime - startTime);

        // Kembalikan tombol ke keadaan semula
        btn.innerHTML = originalBtnContent;
        btn.disabled = false;
        btn.classList.remove('opacity-80', 'cursor-not-allowed');

        // Generate kode pendek secara random
        const randomId = Math.random().toString(36).substr(2, 6);
        const shortLink = `flash.link/${randomId}`;

        // Isi hasil ke tampilan
        document.getElementById('originalUrlText').textContent = url;
        document.getElementById('processTime').textContent = processTime;
        const shortAnchor = document.getElementById('shortUrlAnchor');
        shortAnchor.textContent = shortLink;
        shortAnchor.href = url;

        // Tampilkan hasil dengan animasi
        resultSection.classList.remove('hidden');

        // Auto scroll ke hasil di mobile
        if(window.innerWidth < 768) {
            resultSection.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }

        createParticles(btn);

    }, 800);
}

function copyLink() {
    const shortUrl = document.getElementById('shortUrlAnchor').textContent.trim();
    const btn = document.getElementById('copyBtn');
    const originalHTML = btn.innerHTML;

    navigator.clipboard.writeText(shortUrl).then(() => {
        btn.innerHTML = '<i class="fa-solid fa-check text-green-400"></i> <span class="text-green-400">Disalin!</span>';
        btn.classList.add('bg-green-400/10', 'border-green-400/20');

        setTimeout(() => {
            btn.innerHTML = originalHTML;
            btn.classList.remove('bg-green-400/10', 'border-green-400/20');
        }, 2000);
    });
}

function openLink() {
    const url = document.getElementById('originalUrlText').textContent;
    window.open(url, '_blank');
}

// Tekan Enter untuk submit
document.getElementById('urlInput').addEventListener('keypress', function (e) {
    if (e.key === 'Enter') {
        shortenUrl();
    }
});

// Fungsi efek partikel untuk kesan "Flash"
function createParticles(element) {
    const rect = element.getBoundingClientRect();
    for (let i = 0; i < 10; i++) {
        const particle = document.createElement('div');
        particle.classList.add('particle');
        document.body.appendChild(particle);

        const x = rect.left + rect.width / 2;
        const y = rect.top + rect.height / 2;

        const destinationX = x + (Math.random() - 0.5) * 100;
        const destinationY = y + (Math.random() - 0.5) * 100;

        particle.style.left = `${x}px`;
        particle.style.top = `${y}px`;

        const animation = particle.animate([
            { transform: `translate(0, 0) scale(1)`, opacity: 1 },
            { transform: `translate(${destinationX - x}px, ${destinationY - y}px) scale(0)`, opacity: 0 }
        ], {
            duration: 500 + Math.random() * 500,
            easing: 'cubic-bezier(0, .9, .57, 1)',
            delay: Math.random() * 200
        });

        animation.onfinish = () => particle.remove();
    }
}
