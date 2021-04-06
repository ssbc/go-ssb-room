let hasFocus = true;
window.addEventListener('blur', () => {
  hasFocus = false;
});
window.addEventListener('focus', () => {
  hasFocus = true;
});

const waitingElem = document.getElementById('waiting');
const anchorElem = document.getElementById('join-room-uri');
anchorElem.onclick = function handleURI(ev) {
  const ssbUri = ev.target.dataset.href;
  const fallbackUrl = ev.target.dataset.hrefFallback;
  waitingElem.classList.remove('hidden');
  setTimeout(function () {
    if (hasFocus) window.location = fallbackUrl;
  }, 5000);
  window.location = ssbUri;
};
