// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

const ssbUriLink = document.querySelector('#start-auth-uri');
const waitingElem = document.querySelector('#waiting');
const errorElem = document.querySelector('#failed');
const challengeElem = document.querySelector('#challenge');

const sc = challengeElem.dataset.sc;
const evtSource = new EventSource(`/withssb/events?sc=${sc}`);
let otherTab;

ssbUriLink.onclick = function handleURI(ev) {
  ev.preventDefault();
  const ssbUri = ssbUriLink.href;
  waitingElem.classList.remove('hidden');
  otherTab = window.open(ssbUri, '_blank');
};

evtSource.onerror = (e) => {
  waitingElem.classList.add('hidden');
  errorElem.classList.remove('hidden');
  console.error(e.data);
  if (otherTab) otherTab.close();
};

evtSource.addEventListener('failed', (e) => {
  waitingElem.classList.add('hidden');
  errorElem.classList.remove('hidden');
  console.error(e.data);
  if (otherTab) otherTab.close();
});

// prepare for the case that the success event happens while the browser is not on screen.
let hasFocus = true;
window.addEventListener('blur', () => {
  hasFocus = false;
});

evtSource.addEventListener('success', (e) => {
  waitingElem.classList.add('hidden');
  evtSource.close();
  if (otherTab) otherTab.close();
  const redirectTo = `/withssb/finalize?token=${e.data}`;
  if (hasFocus) {
    window.location.replace(redirectTo);
  } else {
    // wait for the browser to be back in focus and redirect then
    window.addEventListener('focus', () => {
      window.location.replace(redirectTo);
    });
  }
});
