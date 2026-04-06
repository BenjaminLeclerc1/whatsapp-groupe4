import http from "k6/http";
import { check, sleep } from "k6";

// Target: throughput-focused benchmark for message ingestion.
// IMPORTANT:
// - 500k msg/sec is generally not reachable from one local k6 process.
// - Use this script for controlled high-throughput tests and horizontal k6 scaling.

function mustEnv(name) {
  const v = __ENV[name];
  if (!v) throw new Error(`Missing env var: ${name}`);
  return v;
}

function parseUserIDs() {
  const raw = (__ENV.USER_IDS || "").trim();
  if (raw) return raw.split(",").map((s) => s.trim()).filter(Boolean);
  return [
    "11111111-1111-1111-1111-111111111111",
    "22222222-2222-2222-2222-222222222222",
    "33333333-3333-3333-3333-333333333333",
    "44444444-4444-4444-4444-444444444444",
    "55555555-5555-5555-5555-555555555555",
    "66666666-6666-6666-6666-666666666666",
    "77777777-7777-7777-7777-777777777777",
    "88888888-8888-8888-8888-888888888888",
    "99999999-9999-9999-9999-999999999999",
    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  ];
}

const TARGET_RPS = Number(__ENV.TARGET_RPS || 500000);
const PRE_ALLOCATED_VUS = Number(__ENV.PRE_ALLOCATED_VUS || 2000);
const MAX_VUS = Number(__ENV.MAX_VUS || 10000);
const DURATION = __ENV.DURATION || "30s";
const TIME_UNIT = __ENV.TIME_UNIT || "1s";
const BATCH_SIZE = Number(__ENV.BATCH_SIZE || 20); // requests per iteration

export const options = {
  discardResponseBodies: true,
  scenarios: {
    msg_throughput: {
      executor: "constant-arrival-rate",
      rate: TARGET_RPS, // iterations/timeUnit
      timeUnit: TIME_UNIT,
      duration: DURATION,
      preAllocatedVUs: PRE_ALLOCATED_VUS,
      maxVUs: MAX_VUS,
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<1200", "p(99)<2000"],
  },
};

export function setup() {
  const chatSvcUrl = mustEnv("CHAT_SVC_URL");
  const userIDs = parseUserIDs();
  const creatorID = userIDs[0];
  const participants = userIDs.slice(1);

  const createChatPayload = JSON.stringify({
    participants,
    type: "group",
    name: "k6-throughput-500k",
  });

  // setup() may run before services are fully ready; retry a few times.
  const retries = Number(__ENV.SETUP_RETRIES || 5);
  const timeout = __ENV.TIMEOUT || "5s";
  let res = null;
  for (let attempt = 1; attempt <= retries; attempt++) {
    res = http.post(`${chatSvcUrl}/api/v1/chats`, createChatPayload, {
      headers: {
        "Content-Type": "application/json",
        "X-User-ID": creatorID,
      },
      timeout,
      // Global discardResponseBodies=true is great for throughput, but setup
      // needs the returned JSON to extract chat id.
      responseType: "text",
    });

    if (res && !res.error && res.status === 201) {
      break;
    }

    if (attempt < retries) {
      sleep(1);
    }
  }

  if (!res) {
    throw new Error("Chat creation failed: no HTTP response from k6.");
  }
  if (res.error) {
    throw new Error(
      `Chat creation request error: ${res.error} (url=${chatSvcUrl}/api/v1/chats)`
    );
  }
  if (res.status !== 201) {
    throw new Error(
      `Chat creation failed: status=${res.status}, error=${res.error || "<none>"}, body=${res.body || "<empty>"}`
    );
  }

  let chat = null;
  try {
    chat = res.json();
  } catch (err) {
    throw new Error(`Invalid JSON in chat creation response: body=${res.body || "<empty>"}`);
  }
  if (!chat || !chat.id) {
    throw new Error(`Missing chat id in response: ${res.body || "<empty>"}`);
  }

  return { chatID: chat.id, userIDs };
}

export default function (data) {
  const msgSvcUrl = mustEnv("MSG_SVC_URL");
  const timeout = __ENV.TIMEOUT || "5s";
  const reqs = [];

  // Build N message requests and send them as one batch.
  for (let i = 0; i < BATCH_SIZE; i++) {
    const userID = data.userIDs[(__VU + __ITER + i) % data.userIDs.length];
    reqs.push([
      "POST",
      `${msgSvcUrl}/api/v1/messages`,
      JSON.stringify({
        chat_id: data.chatID,
        content: `k6 throughput vu=${__VU} iter=${__ITER} i=${i}`,
      }),
      {
        headers: {
          "Content-Type": "application/json",
          "X-User-ID": userID,
        },
        timeout,
      },
    ]);
  }

  const responses = http.batch(reqs);
  for (const r of responses) {
    check(r, {
      "status 201": (x) => x.status === 201,
    });
  }
}

