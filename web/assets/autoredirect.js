let hasFocus = true;
window.addEventListener('blur', () => {
  hasFocus = false;
});
window.addEventListener('focus', () => {
  hasFocus = true;
});

const anchorElem = document.getElementById('join-room-uri');
anchorElem.onclick = function handleURI(ev) {
  const ssbUri = ev.target.dataset.href;
  const fallbackUrl = ev.target.dataset.hrefFallback;
  setTimeout(function () {
    if (hasFocus) window.location = fallbackUrl;
  }, 500);
  window.location = ssbUri;
};
