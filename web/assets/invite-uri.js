let hasFocus = true;
window.addEventListener('blur', () => {
  hasFocus = false;
});
window.addEventListener('focus', () => {
  hasFocus = true;
});

const waitingElem = document.getElementById('waiting');
const anchorElem = document.getElementById('claim-invite-uri');
anchorElem.onclick = function handleURI(ev) {
  ev.preventDefault();
  const ssbUri = anchorElem.href;
  const fallbackUrl = anchorElem.dataset.hrefFallback;
  waitingElem.classList.remove('hidden');
  setTimeout(function () {
    if (hasFocus) window.location.replace(fallbackUrl);
  }, 5000);
  window.location.replace(ssbUri);
};
