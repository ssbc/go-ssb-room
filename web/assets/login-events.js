const ssbUriLink = document.querySelector('#start-auth-uri');
const waitingElem = document.querySelector('#waiting');
const errorElem = document.querySelector('#failed');
const challengeElem = document.querySelector('#challenge');

const sc = challengeElem.dataset.sc;
const evtSource = new EventSource(`/withssb/events?sc=${sc}`);

ssbUriLink.addEventListener('click', (e) => {
  errorElem.classList.add('hidden');
  waitingElem.classList.remove('hidden');
});

evtSource.onerror = (e) => {
  waitingElem.classList.add('hidden');
  errorElem.classList.remove('hidden');
  console.error(e.data);
};

evtSource.addEventListener('failed', (e) => {
  waitingElem.classList.add('hidden');
  errorElem.classList.remove('hidden');
  console.error(e.data);
});

evtSource.addEventListener('success', (e) => {
  waitingElem.classList.add('hidden');
  evtSource.close();
  window.location = `/withssb/finalize?token=${e.data}`;
});
