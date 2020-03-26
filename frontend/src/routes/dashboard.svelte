<script>
  import Authenticated from '../layouts/authenticated.svelte'

  let websocket;
  let command = "";
  let log = "";

  function sendCommand() {
    if (process.browser) {
      websocket.send(`{"command": "SendToTerminal", "params": { "text": "${command}"}}`);
      command = "";
    }
  }

  function onMessage(evt) {
    let inbound = JSON.parse(evt.data)

    if (inbound.action === "TerminalStdouted" || inbound.acount === "TerminalErrored") {
      log += inbound.body.text + "\n";
      let objDiv = document.getElementById("end-of-log");
      objDiv.scrollIntoView(false);
    }
  }

  function onCommandKeyDown(evt) {
    if(evt.keyCode === 13) {
      evt.preventDefault();
      sendCommand();
    }
  }

  if (process.browser) {
    websocket = new WebSocket("ws://localhost:5000/ws");
    websocket.onopen = function(evt) {
      let token = window.localStorage.getItem("token");
      websocket.send(`{"command": "Authenticate", "params": { "token": "${token}"}}`);
    };
    //websocket.onclose = function(evt) { onClose(evt) };
    websocket.onmessage = function(evt) { onMessage(evt) };
    //websocket.onerror = function(evt) { onError(evt) };
  }

</script>

<Authenticated>
  <div class="py-4">
    <!--<div class="border-4 border-dashed border-gray-200 rounded-lg h-96"></div>-->
    <pre id="server-log" class="border-4 border-gray-200 h-96 overflow-auto">
      {log}
      <span id="end-of-log"></span>
    </pre>
    <div class="flex flex-row">
      <input id="command" class="flex-grow form-input sm:text-sm sm:leading-5" placeholder="server command" bind:value="{command}" on:keydown={onCommandKeyDown} />
      <button type="button" class="flex-none inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-500 focus:outline-none focus:border-indigo-700 focus:shadow-outline-indigo active:bg-indigo-700 transition ease-in-out duration-150" on:click={sendCommand}>
        Send
        <svg class="ml-2 -mr-0.5 h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884zM18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" clip-rule="evenodd"/>
        </svg>
      </button>
    </div>
  </div>
</Authenticated>
