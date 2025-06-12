import ws from "k6/ws";
import http from "k6/http";
import { check, fail, sleep } from "k6";
import { Trend } from "k6/metrics";
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

export const connectTime = new Trend("ws_opened_time", true);

export let options = {
  insecureSkipTLSVerify: true,
  scenarios: {
    playerBatch1: {
      executor: "per-vu-iterations",
      vus: 20,
      iterations: 1,
      maxDuration: "60s",
      startTime: "0s",
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
  const url = getUrlHttps("http", "gamelaunch");
  let start = Date.now();
  const res = http.post(url, payloads.gamelaunch(), { headers });
  let elapsed = Date.now() - start;
  gamelaunchResponseTime.add(elapsed);
  //console.log(`${__VU} Status: ${res.status}`);
  //console.log(`${__VU} Body: ${res.body}`);

  let data;
  try {
    data = JSON.parse(res.body);
  } catch (e) {
    fail(`${__VU} Invalid JSON response: ${res.body}`);
  }
  //console.log(`${__VU} - Parsed data:`, JSON.stringify(data));
  //console.log(`${__VU} - Data keys:`, Object.keys(data));

  check(res, {
    "status is 200": (r) => r.status === 200,
  });
  return data.url;
}

function runPlayerVU(responseDelay = 0) {
  let opened = false;
  let turnNumber = 0;
  let wsUrl = runGamelaunch();
  if (!wsUrl) return;
  var isWebsocketClosed = false;
  var currentState = "OFFLINE";

  var Board;

  //sleep(MAX_SLEEP);
  //console.log(`${__VU} - Game launch finished`);
  wsUrl = toWsUrl(wsUrl);
  //console.log(`${__VU} - Url transformed`);

  //sleep(MAX_SLEEP);
  let start = Date.now();
  let playerId;

  ws.connect(wsUrl, null, function (socket) {
    socket.on("open", () => {
      const elapsed = Date.now() - start;
      connectTime.add(elapsed);
      opened = true;
      check(true, { "WebSocket opened": (v) => v === true });
      //console.log(`${__VU} - WebSocket opened`);
      currentState = GameStates[1];
      //sleep(MAX_SLEEP);
    });

    socket.on("ping", () => {
      socket.ping();
    });
    socket.on("pong", () => {
      socket.ping();
    });

    socket.on("message", (msg) => {
      const data = JSON.parse(msg);
      //if (data.command != "pong") console.log(`${__VU} - received:`, data);
      if (data.command === "pong") {
        // sleep(1);
        socket.send(JSON.stringify({ command: "ping" }));
      }

      if (data.command === "connected" && !queueSent) {
        const elapsed = Date.now() - start;
        connectedMsgTime.add(elapsed); // record metric
        HandleConnection();
        //sleep(MAX_SLEEP);
        playerId = data.value.player_id;
        socket.setTimeout(() => {
          startQueueRequest = Date.now();
          socket.send(getMsgQueueRequest({ value: QUEUE_VALUE }));
          queueSent = true;
          socket.send(JSON.stringify({ command: "ping" }));
          //console.log(
          //  `${__VU} - Sent: ${getMsgQueueRequest({ value: QUEUE_VALUE })}`
          //);
        }, 20);
      }

      if (data.command === "queue_confirmation" && data.value) {
        const elapsed = Date.now() - startQueueRequest;
        queueConfirmationResponseTime.add(elapsed); // record metric
        HandleQueueConfirmation();
        currentState = GameStates[2];
        startPairedTimer = Date.now();
      } else if (data.command === "queue_confirmation" && !data.value) {
        // sleep(MAX_SLEEP);
        socket.send(getMsgQueueRequest({ value: QUEUE_VALUE }));
        //console.log(
        //  `${__VU} - Sent: ${getMsgQueueRequest({ value: QUEUE_VALUE })}`
        //);
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
        }, 20);
        currentState = GameStates[3];
      }

      /*    if (data.command === "game_info") {
        HandleGameInfo(gameInfo);
        //gameInfo = true;
      } */

      /*   if (data.command === "game_timer") {
        HandleGameTimer(gameTimerReceived);
      } */

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
        }, 20);
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
        }, 20);
      }

      if (data.command === "turn_switch") {
        let elapsed = Date.now() - startMovementTimer;
        movementResponseTime.add(elapsed, { Turn: turnNumber });
        HandleTurnSwitch();
        turnSwitchReceived = true;
        turnNumber++;

        if (turnNumber >= Turns.length) socket.close();
      }
    });

    socket.on("error", (msg) => {
      socket.close();
      console.log("[ERROR ON WEBSOCKET]: ", msg);
    });

    socket.on("close", () => {
      //console.log(`${__VU} - Closing socket connection`);
      isWebsocketClosed = true;
      CleanUpAndClose(socket, opened, isWebsocketClosed);
    });
    /*
    socket.setTimeout(() => {
      socket.close();
    }, 5000); */
  });
}

const CleanUpAndClose = (socket, opened, isWebsocketClosed) => {
  if (!pairedReceived) {
    check(false, { "Paired message not received": (v) => v === true });
  }
  /*   if (!gameInfo) {
    check(false, { "Game info message not received": (v) => v === true });
  } */
  if (!gameStartReceived) {
    check(false, { "Game start message not received": (v) => v === true });
  }
  if (!balanceUpdateReceived) {
    check(false, {
      "Balance Update message not received": (v) => v === true,
    });
  }
  if (!opponentReadyReceived) {
    check(false, {
      "Opponent ready message not received": (v) => v === true,
    });
  }
  if (!turnSwitchReceived) {
    check(false, {
      "Turn switch message not received": (v) => v === true,
    });
  }
  if (!gameStartReceived)
    check(false, { "Game Not Started": (v) => v === true });
  if (!opened) {
    check(false, { "WebSocket opened": (v) => v === true });
  }
  if (!isWebsocketClosed) {
    socket.send(getMsgLeaveQueue());
    socket.send(getMsgLeaveRoom());
    socket.send(getMsgConcedeGame());
  }
};

export function teardown(data) {
  //console.log(JSON.stringify(data));
}

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
