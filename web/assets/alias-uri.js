let hasFocus = true;
window.addEventListener('blur', () => {
  hasFocus = false;
});
window.addEventListener('focus', () => {
  hasFocus = true;
});

const waitingElem = document.getElementById('waiting');
const failureElem = document.getElementById('failure');
const anchorElem = document.getElementById('alias-uri');

// Autoredirect to the ssb uri ASAP
setTimeout(() => {
  const ssbUri = anchorElem.href;
  window.location.replace(ssbUri);
}, 100);

// Redirect to ssb uri or show failure state
anchorElem.onclick = function handleURI(ev) {
  ev.preventDefault();
  const ssbUri = anchorElem.href;
  waitingElem.classList.remove('hidden');
  setTimeout(function () {
    if (hasFocus) {
      waitingElem.classList.add('hidden');
      failureElem.classList.remove('hidden');
    }
  }, 5000);
  window.location.replace(ssbUri);
};
