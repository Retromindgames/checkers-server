import ws from "k6/ws";
import http from "k6/http";
import { check, fail, sleep } from "k6";
import { Trend, Counter } from "k6/metrics";
import {
  getUrlHttp,
  toWsUrl,
  headers,
  options as customOptions,
  payloads,
  getUrlHttps,
} from "./test-config.js";
import {
  getMsgQueueRequest,
  getMsgLeaveQueue,
  getMsgReadyRoom,
  getMsgLeaveRoom,
  getMsgConcedeGame,
  getMsgMovePiece,
} from "./helpers.js";

import {
  HandleConnection,
  HandleQueueConfirmation,
  HandlePaired,
  HandleGameInfo,
  HandleGameTimer,
  HandleBalanceUpdate,
  HandleOpponentReady,
  HandleGameStart,
  HandleTurnSwitch,
  connectedMsgTime,
  pairedResponseTime,
  queueConfirmationResponseTime,
  movementResponseTime,
} from "./utils/commands.js";
import { MAX_SLEEP, QUEUE_VALUE } from "./utils/constants.js";
import { GameStates, Turns } from "./utils/gameState.js";

export const gamelaunchResponseTime = new Trend(
  "http_gamelaunch_response_time",
  true
);
export const wsErrors = new Counter("vu_ws_errors");
export const vuIte = new Counter("vu_iterations");
export const vuGamelaunch = new Counter("vu_gamelaunch");
export const vuGamelaunchOk = new Counter("vu_gamelaunch_ok");
export const vuWsConn = new Counter("vu_ws_conn");
export const vuWsConnOk = new Counter("vu_ws_conn_ok");
export const connectTime = new Trend("ws_opened_time", true);

/*
    Command to run in EC2 machine.
        nohup k6 run --out json=results-long.json --summary-export=summary-long.json tests-long-low.js > k6.log 2>&1 &

    to check progress:
        tail -f k6.log

    to stop:
        ps aux | grep k6
        kill <PID>

    to check results while running:
        tail -f results-long.json
*/

export let options = {
  insecureSkipTLSVerify: true,
  scenarios: {
    playerBatch1: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "5m", target: 400 }, 
        { duration: "10m", target: 1000 }, 
        { duration: "1m", target: 0 },   // ramp down to 0
      ],
      exec: "player1",
    }
  },
};

let queueSent = false;
let gameTimerReceived = false;
let gameStartReceived = false;
let balanceUpdateReceived = false;
let opponentReadyReceived = false;
let gameInfo = false;
let queueConfirmationReceived = false;
let pairedReceived = false;
var turnSwitchReceived = false;
let startQueueRequest = Date.now();
let startPairedTimer = Date.now();
let startMovementTimer = Date.now();

function runGamelaunch() {
  
  vuGamelaunch.add(1)

  const url = getUrlHttps("http", "gamelaunch");
  let start = Date.now();
  const res = http.post(url, payloads.gamelaunch(), { headers });
  let elapsed = Date.now() - start;
  gamelaunchResponseTime.add(elapsed);
  vuGamelaunchOk.add(1)

  let data;
  try {
    data = JSON.parse(res.body);
  } catch (e) {
    fail(`${__VU} Invalid JSON response: ${res.body}`);
  }
  check(res, {
    "status is 200": (r) => r.status === 200,
  });
  return data.url;
}

function runPlayerVU(responseDelay = 0) {
  vuIte.add(1) 
  let opened = false;
  let turnNumber = 0;
  let wsUrl = runGamelaunch();
  if (!wsUrl) return;
  var isWebsocketClosed = false;
  var currentState = "OFFLINE";

  var Board;

  wsUrl = toWsUrl(wsUrl);

  let start = Date.now();
  let playerId;
  vuWsConn.add(1)
  ws.connect(wsUrl, null, function (socket) {
    
    socket.on("open", () => {
      const elapsed = Date.now() - start;
      connectTime.add(elapsed);
      vuWsConnOk.add(1)
      opened = true;
      check(true, { "WebSocket opened": (v) => v === true });
      currentState = GameStates[1];
       (function pingLoop() {
        if (isWebsocketClosed) return;
        socket.send(JSON.stringify({ command: 'ping' }));
        setTimeout(pingLoop, 1000);
      })();
    });

    socket.on("ping", () => {
      socket.ping();
    });
    socket.on("pong", () => {
      socket.ping();
    });

    socket.on("message", (msg) => {
      const data = JSON.parse(msg);
      if (data.command === "pong") {
        socket.send(JSON.stringify({ command: "ping" }));
      }

      if (data.command === "connected" && !queueSent) {
        const elapsed = Date.now() - start;
        connectedMsgTime.add(elapsed); 
        HandleConnection();
        playerId = data.value.player_id;
        socket.setTimeout(() => {
          socket.send(getMsgQueueRequest({ value: QUEUE_VALUE }));
          queueSent = true;
          startQueueRequest = Date.now();
          socket.send(JSON.stringify({ command: "ping" }));
        }, 150);
      }

      if (data.command === "queue_confirmation" && data.value) {
        const elapsed = Date.now() - startQueueRequest;
        queueConfirmationResponseTime.add(elapsed); 
        HandleQueueConfirmation();
        currentState = GameStates[2];
        startPairedTimer = Date.now();
      } else if (data.command === "queue_confirmation" && !data.value) {
        socket.send(getMsgQueueRequest({ value: QUEUE_VALUE }));
      }

      if (data.command === "paired") {
        const elapsed = Date.now() - startPairedTimer;
        pairedResponseTime.add(elapsed);
        HandlePaired();
        pairedReceived = true;

        //sleep(MAX_SLEEP);
        socket.setTimeout(() => {
          socket.send(getMsgReadyRoom({ value: true })); // simulate readiness
          //console.log(`${__VU} - Sent: ${getMsgReadyRoom({ value: true })}`);
        }, 150);
        currentState = GameStates[3];
      }


      if (data.command === "game_start") {
        HandleGameStart(gameStartReceived, startMovementTimer);
        gameStartReceived = true;
        currentState = GameStates[4];
        Board = data.value.Board;
        socket.setTimeout(() => {
          startMovementTimer = Date.now();
          if (playerId === data.value.Board[Turns[turnNumber].from].player_id) {
            socket.send(
              getMsgMovePiece({
                player_id: data.value.Board[Turns[turnNumber].from].player_id,
                piece_id: data.value.Board[Turns[turnNumber].from].piece_id,
                from: Turns[turnNumber].from,
                to: Turns[turnNumber].to,
                is_capture: false,
                is_kinged: false,
              })
            );
          }
        }, 150);
      }

      if (data.command === "balance_update") {
        HandleBalanceUpdate();
        balanceUpdateReceived = true;
      }

      if (data.command === "opponent_ready" && data.value.is_ready) {
        HandleOpponentReady(opponentReadyReceived);
        opponentReadyReceived = true;
      }

      if (data.command === "room_failed_ready_check") {
        CleanUpAndClose(socket, opened, isWebsocketClosed);
        fail("Failed to be ready in room");
      }
      
      if (data.command === "move_piece") {
        socket.setTimeout(() => {
          if (playerId !== data.value.player_id && turnNumber < Turns.length) {
            startMovementTimer = Date.now();
            socket.send(
              getMsgMovePiece({
                player_id: Board[Turns[turnNumber].from].player_id,
                piece_id: Board[Turns[turnNumber].from].piece_id,
                from: Turns[turnNumber].from,
                to: Turns[turnNumber].to,
                is_capture: false,
                is_kinged: false,
              })
            );
          }
        }, 150);
      }

      if (data.command === "turn_switch") {
        let elapsed = Date.now() - startMovementTimer;
        movementResponseTime.add(elapsed, { Turn: turnNumber });
        HandleTurnSwitch();
        turnSwitchReceived = true;
        turnNumber++;
        let closingTimerSet = false;
        if (turnNumber >= Turns.length && !closingTimerSet) {
          closingTimerSet = true;
          socket.setTimeout(() => {
            socket.close();
          }, 60000);
        }
      }
    });

    socket.on("error", (msg) => {
      socket.close();
      wsErrors.add(1);
      console.log(`${__VU} - [ERROR ON WEBSOCKET]: `, JSON.stringify(msg, null, 2));
    });

    socket.on("close", () => {
      //console.log(`${__VU} - Closing socket connection`);
      isWebsocketClosed = true;
      CleanUpAndClose(socket, opened, isWebsocketClosed);
    });
    
    socket.setTimeout(() => {
      socket.close();
    }, 180000); // 3mins after connection, hard timeout to close conn
  });

  // fallback global timeout
  //setTimeout(() => {
  //  if (!isWebsocketClosed) {
  //    console.log(`${__VU} - Fallback timeout triggered`);
  //    // force cleanup logic if needed
  //  }
  //}, 200000); // global timeout as a safety net, must be bigger than previous one.
}

const CleanUpAndClose = (socket, opened, isWebsocketClosed) => {
  if (!pairedReceived) {
    check(false, { "Paired message received": (v) => v === true });
  }
  /*   if (!gameInfo) {
    check(false, { "Game info message not received": (v) => v === true });
  } */
  if (!gameStartReceived) {
    check(false, { "Game start message received": (v) => v === true });
  }
  if (!balanceUpdateReceived) {
    check(false, {
      "Balance Update message received": (v) => v === true,
    });
  }
  if (!opponentReadyReceived) {
    check(false, {
      "Opponent ready message received": (v) => v === true,
    });
  }
  if (!turnSwitchReceived) {
    check(false, {
      "Turn switch message received": (v) => v === true,
    });
  }
  if (!opened) {
    check(false, { "WebSocket opened": (v) => v === true });
  }
  if (!isWebsocketClosed) {
    socket.send(getMsgLeaveQueue());
    socket.send(getMsgLeaveRoom());
    socket.send(getMsgConcedeGame());
  }
};

export function player1() {
  Reset();
  runPlayerVU(1);
}

export const Reset = () => {
  queueSent = false;
  gameTimerReceived = false;
  gameStartReceived = false;
  balanceUpdateReceived = false;
  opponentReadyReceived = false;
  gameInfo = false;
  queueConfirmationReceived = false;
  pairedReceived = false;
  turnSwitchReceived = false;
  startQueueRequest = Date.now();
  startPairedTimer = Date.now();
  startMovementTimer = Date.now();
};
