import ws from 'k6/ws';
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';
import {
  getUrlHttps,
  toWsUrl,
  headers,
  options as customOptions,
  payloads
} from './test-config.js';
import { getMsgQueueRequest, getMsgLeaveQueue, getMsgReadyRoom, getMsgLeaveRoom } from './helpers.js';

export let gamelaunchResponseTime = new Trend('http_gamelaunch_response_time', true);
export let connectedMsgTime = new Trend('ws_connected_message_time', true);
export let queueConfirmationResponseTime = new Trend('ws_queue_confirmation_response_message_time', true);
export let pairedResponseTime = new Trend('ws_paired_response_message_time', true);
export let connectTime = new Trend('ws_connect_time', true);


export let options = {
  insecureSkipTLSVerify: true,
  scenarios: {
    player1: {
      executor: 'per-vu-iterations',
      vus: 2,
      iterations: 1,
      exec: 'player1'
    }
  },
};


function connectPlayer(playerId, responseDelay = 0) {
  const url = getUrlHttps('http', 'gamelaunch');
  let opened = false;
  let start = Date.now();
  const res = http.post(url, payloads.gamelaunch(), { headers });
  const elapsed = Date.now() - start; 
  gamelaunchResponseTime.add(elapsed);
  check(res, { 'gamelaunch status 200': (r) => r.status === 200 });
  const wsUrl = toWsUrl(JSON.parse(res.body).url);

  ws.connect(wsUrl, null, function (socket) {
    let queueSent = false;
    let pairedReceived = false;
    start = Date.now();
    let startQueueRequest;
    let startPairedTimer;

    setTimeout(() => {
      if (!opened) {
        check(false, { 'WebSocket opened': (v) => v === true });
      }
    }, 3000); // 3s timeout
    
    socket.on('open', () => {
      const elapsed = Date.now() - start; 
      connectTime.add(elapsed)
      opened = true;
      check(true, { 'WebSocket opened': (v) => v === true });
      console.log(`${playerId}: WebSocket opened`);
    });

    socket.on('message', (msg) => {
      const data = JSON.parse(msg);
      console.log(`${playerId} received:`, data);

      if (data.command === 'connected' && !queueSent) {
        const elapsed = Date.now() - start;
        connectedMsgTime.add(elapsed); // record metric        
        sleep(responseDelay); // Optional delay to control timing
        socket.send(getMsgQueueRequest({ value: 100 }));
        startQueueRequest = Date.now()
        queueSent = true;
        // TODO: fazer um check para termos recebido a connection.
        //check(res, {
        //  'ws connection status is 101': (r) => r && r.status === 101,
        //});
      }

      if (data.command === 'queue_confirmation' && data.value) {
        const elapsed = Date.now() - startQueueRequest;
        console.log(`Queue Confirmation message received after ${elapsed} ms`);
        queueConfirmationResponseTime.add(elapsed); // record metric
        startPairedTimer = Date.now();
        // TODO: fazer um check para termos recebido a queue confirmation.
        //check(res, {
        //  'ws connection status is 101': (r) => r && r.status === 101,
        //});
      }

      if (data.command === 'paired' && data.value) {
        const elapsed = Date.now() - startPairedTimer;

        console.log(`${playerId} received paired:`, data.value);
        pairedResponseTime.add(elapsed);
        sleep(responseDelay)
        socket.send(getMsgReadyRoom({ value: true })); // simulate readiness
        pairedReceived = true;
      }
    });

    socket.setTimeout(() => {
      if (!pairedReceived) {
        console.log(`${playerId}: did not receive paired in time.`);
      }
      socket.send(getMsgLeaveQueue());
      socket.send(getMsgLeaveRoom());
      socket.close();
    }, 10000);
  });
}

// Separate player exec functions
export function player1() {
  connectPlayer("Player1", 0.2);
}

export function player2() {
  connectPlayer("Player2", 0.2); // Optional slight delay
}
