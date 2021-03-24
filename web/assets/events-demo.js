let streamName = document.querySelector("#stream-name").attributes.stream.value

var evtSource = new EventSource(`/sse/events?stream=${streamName}`);

var eventList = document.querySelector('#event-list');

evtSource.addEventListener("testing", (e) => {
//   console.log(e)

  var newElement = document.createElement("li");
  newElement.textContent = `(${e.lastEventId}) message: ${e.data}`;
  eventList.prepend(newElement);
})