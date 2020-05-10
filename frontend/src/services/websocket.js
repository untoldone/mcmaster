// https://developer.mozilla.org/en-US/docs/Web/API/WebSocket#Ready_state_constants
const WS_CONNECTING = 0;
const WS_OPEN = 1;
const WS_CLOSING = 2;
const WS_CLOSED = 3;

const DEFAULT_HOST = 'localhost';
const DEFAULT_PORT = 5000;
const DEFAULT_SSL = false;
const HEALTH_CHECK_INTERVAL = 1000;
const MAX_OUTSTANDING_MESSAGES = 5;
const DEFAULT_MESSAGE_TIMEOUT = 6000;

function randomId() {
  return Math.random().toString(36).substring(2);
}

export default class McMasterSocket {
  constructor(props) {
    this._host = props.host || DEFAULT_HOST;
    this._port = props.port || DEFAULT_PORT;
    this._secure = props.secure !== null ? props.secure : DEFAULT_SSL;

    this._isOpened = false;
    this._isAuthed = false;
    this._listeners = {};
    this._pendingMessageQueue = [];
  }

  open() {
    if (this._isOpened) {
      return;
    }

    this._scheduleSockSetup();

    this._healthInterval = setInterval(this._healthCheck.bind(this), HEALTH_CHECK_INTERVAL);

    this._isOpened = true;
  }

  close() {
    if (!this._isOpened) {
      return;
    }
  
    this._isOpened = false;
    this._isAuthed = false;

    this._sockTeardown();

    clearInterval(this._healthInterval);
    clearTimeout(this._scheduleSockSetupTimeout);
    this._nextScheduleSockSetup = null;

    this._rejectPendingMessages();
  }

  authenticate(token) {
    throw 'Implement';
    return this._sock.send('Authenticate', {'token': token})
            .then((result) => {
              this._isAuthed = result.body.result === 'authenticated';
              return this._isAuthed;
            });
  }

  send(commmand, data) {
    if (!this._isOpened) {
      throw 'Client must be opened';
    }

    let promise = this._queuePendingMessage(command, data);

    this._sendPendingMessages();

    return promise;
  }

  forceReconnectRetry() {
    if (!this._isOpened) {
      throw 'Client must be opened';
    }

    if (this._nextScheduleSockSetup == null) {
      return;
    }

    clearTimeout(this._scheduleSockSetupTimeout);
    this._nextScheduleSockSetup();
  }

  on(eventName, callback) {
    let eventListeners = this._eventListeners(eventName);
    eventListeners.push(callback);
  }

  off(eventName, callback) {
    let eventListeners = this._eventListeners(eventName);

    // delete all listeners if no parameters
    if(!eventName) {
      this._listeners = {};
      return;
    }

    // If no callback, delete list of listeners on that event
    // If callback, delete just that specific listener
    if(callback) {
      let index = eventListeners.indexOf(callback);
      if (index > -1) {
        eventListeners.splice(index, 1);
      }
    } else {
      delete eventListeners[eventName];
    }
  }

  _sockUri() {
    return `${this._secure ? 'wss' : 'ws'}://${this._host}:${this._port}/ws`;
  }

  _sockSetup() {
    if(this._sock) {
      throw '_sock must not exist';
    }

    try {
      this._sock = new WebSocket(this._sockUri());

      this._sock.onopen = this._onSockOpen.bind(this);
      this._sock.onclose = this._onSockClose.bind(this);
      this._sock.onmessage = this._onSockMessage.bind(this);
      this._sock.onerror = this._onSockError.bind(this);
    } catch (err) {
      if(this._sock) {
        this._sockTeardown();
        throw err;
      }
    }
  }

  _sockTeardown() {
    this._sock.onerror = null;
    this._sock.onmessage = null;
    this._sock.onclose = null;
    this._sock.onopen = null;

    this._sock.close();

    this._sock = null;
  }

  _scheduleSockSetup(connectionAttempt = 1) {
    if(this._sock && this._sock.readyState === WS_OPEN) {
      return;
    }

    try {
      this._sockSetup();
      this._nextScheduleSockSetup = null;
    } catch (err) {
      // Exponential backoff until max of 120 seconds
      var jitter = (Math.random() * 100) - 50;
      var milliToAttempt = Math.min(Math.pow(connectionAttempt, 2), 120) * 1000 + jitter * Math.min(1, connectionAttempt);
      this._nextScheduleSockSetup = this._scheduleSockSetup.bind(this, connectionAttempt + 1);
      this._scheduleSockSetupTimeout = setTimeout(this._nextScheduleSockSetup, milliToAttempt);

      this._emit('reconnect-retry-scheduled', { delay: milliToAttempt });
    }
  }

  _onSockOpen() {
    this._connectionAttempt = 0;
    this._emit('socket-opened');
    this._sendPendingMessages();
  }

  _onSockClose() {
    this._emit('socket-closed');

    if(this._isOpened) {
      // Closure was premature, clean up and reopen socket
      this._clearPendingMessagesSentAt();
      this._scheduleSockSetup();
    }
  }

  _onSockMessage(event) {
    let body = JSON.parse(event.data);
    let item = this._removeOutstanding(body.clientId);

    if (item && body.success) {
      item.resolve(body.data);
    } else if (item) {
      item.reject(body.data);
    } else {
      this._emit('inbound', body.data);
    }
  }

  _queuePendingMessage() {
    let clientId = randomId();
    let body = { clientId, command, data };

    return new Promise((resolve, reject) => {
      this._pendingMessageQueue.push({clientId, body, resolve, reject});
    });
  }

  _sendPendingMessages() {
    if(this._sock && this._sock.readyState === WS_OPEN) {
      let outstanding = 0;
      let pendingMessage;
      while((pendingMessage = this._pendingMessageQueue[outstanding]) && outstanding <= MAX_OUTSTANDING_MESSAGES) {
        pendingMessage.sentAt = new Date();
        this._sock.send(JSON.stringify(pendingMessage.body));
      }
    }
  }

  _clearPendingMessagesSentAt() {
    this._pendingMessageQueue.forEach((item) => delete item.sentAt);
  }

  _removePendingMessage(clientId) {
    if(clientId == null) {
      return;
    }

    let sentIndex = this._pendingMessageQueue.findIndex(i => i.clientId === clientId);
    if (sentIndex > -1) {
      return item = this._pendingMessageQueue.splice(sentIndex, 1);
    }
  }

  _rejectPendingMessages() {
    let rejectedBody = { message: 'Manually rejected' };
    this._pendingMessageQueue.forEach((item) => item.reject(rejectedBody));
    this._pendingMessageQueue = [];
  }

  _someUnhealthyPendingMessages() {
    let now = new Date();
    return this._pendingMessageQueue.some((item) => 
                                            item.sentAt &&
                                            now - item.sentAt > DEFAULT_MESSAGE_TIMEOUT)
  }

  _onSockError() {
    // No-op
  }

  // TODO: Base this on something else? Like a ping?
  _healthCheck() {
    if(this._someUnhealthyPendingMessages()) {
      this._sockTeardown();
      this._scheduleSockSetup();
    }
  }

  _emit(eventName, args) {
    let listeners = this._eventListeners(eventName);
    listeners.forEach((l) => {
      if(Array.isArray(args)) {
        l.apply(this, args);
      } else {
        l.call(this, args);
      }
    })
  }

  _eventListeners(eventName) {
    return this._listeners[eventName] = this._listeners[eventName] || [];
  }
}
