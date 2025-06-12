import { check, sleep } from "k6";
import { Trend } from "k6/metrics";

export let connectedMsgTime = new Trend("ws_connected_message_time", true);

export let pairedResponseTime = new Trend(
  "ws_paired_response_message_time",
  true
);

export let queueConfirmationResponseTime = new Trend(
  "ws_queue_confirmation_response_message_time",
  true
);
export let movementResponseTime = new Trend(
  "ws_movement_response_message_time",
  true
);

export const HandleConnection = () => {
  check(true, { "Connected message received": (v) => v === true });
};
export const HandleQueueConfirmation = () => {
  check(true, {
    "Queue confirmation message received": (v) => v === true,
  });
};
export const HandlePaired = () => {
  check(true, { "Paired message received": (v) => v === true });
};
export const HandleGameInfo = () => {
  check(true, { "Game info message received": (v) => v === true });
};
export const HandleGameTimer = () => {
  check(true, { "Game timer message received": (v) => v === true });
};
export const HandleGameStart = () => {
  check(true, { "Game start message received": (v) => v === true });
};
export const HandleBalanceUpdate = () => {
  check(true, { "Balance Update message received": (v) => v === true });
};

export const HandleOpponentReady = () => {
  check(true, { "Opponent ready message received": (v) => v === true });
};
export const HandleTurnSwitch = () => {
  check(true, { "Turn switch message received": (v) => v === true });
};
