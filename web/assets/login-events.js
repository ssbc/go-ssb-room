// get the challenge from out of the HTML
const sc = document.querySelector("#challenge").dataset.sc
const evtSource = new EventSource(`/withssb/events?sc=${sc}`);

const ping = document.querySelector('#ping');
const failed = document.querySelector('#failed');

evtSource.onerror = (e) => {
    failed.textContent = "Warning: The connection to the server was interupted."
}

// TODO: change to some css-style progress indicator
evtSource.addEventListener("ping", (e) => {
  ping.textContent = e.data;
})

evtSource.addEventListener("failed", (e) => {
  failed.textContent = e.data;
})

evtSource.addEventListener("success", (e) => {
  evtSource.close()
  window.location = `/withssb/finalize?token=${e.data}`
})
