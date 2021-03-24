// get the challenge from out of the HTML
let sc = document.querySelector("#challenge").attributes.ch.value
var evtSource = new EventSource(`/sse/events?sc=${sc}`);

var ping = document.querySelector('#ping');
var failed = document.querySelector('#failed');

evtSource.onerror = (e) => {
    failed.textContent = "Warning: The connection to the server was interupted."
}

evtSource.addEventListener("ping", (e) => {
  ping.textContent = e.data;
})

evtSource.addEventListener("failed", (e) => {
  failed.textContent = e.data;
})

evtSource.addEventListener("success", (e) => {
  console.log('trigger redirect!')
  alert(e.data)
})
