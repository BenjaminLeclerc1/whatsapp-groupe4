import http from "k6/http";
import { check, sleep } from "k6";

// 10k "simultaneous connections" in k6 usually means 10k VUs running at once.
// This script is intentionally lightweight (health endpoint) to benchmark pure HTTP capacity.

const BASE_URL = __ENV.BASE_URL; // ex: https://my-app.azurecontainerapps.io
if (!BASE_URL) {
  throw new Error("Missing BASE_URL env var (ex: https://myapp.example.com)");
}

export const options = {
  // Keep memory/CPU lower by not storing bodies
  discardResponseBodies: true,

  // If you want to force more TCP connections, you can set:
  // -e NO_CONN_REUSE=true
  // Warning: this is much heavier and can hit OS/port limits quickly.
  noConnectionReuse: (__ENV.NO_CONN_REUSE || "").toLowerCase() === "true",
  noVUConnectionReuse: (__ENV.NO_VU_CONN_REUSE || "").toLowerCase() === "true",

  scenarios: {
    ramp_10k_vus: {
      executor: "ramping-vus",
      startVUs: Number(__ENV.START_VUS || 50),
      gracefulRampDown: "10s",
      stages: [
        { duration: __ENV.RAMP_1 || "30s", target: Number(__ENV.VUS_1 || 1000) },
        { duration: __ENV.RAMP_2 || "30s", target: Number(__ENV.VUS_2 || 5000) },
        { duration: __ENV.RAMP_3 || "30s", target: Number(__ENV.VUS_3 || 10000) },
        { duration: __ENV.HOLD || "30s", target: Number(__ENV.VUS_MAX || 10000) },
      ],
    },
  },

  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<500", "p(99)<1000"],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/health`, {
    timeout: __ENV.TIMEOUT || "5s",
    headers: {
      // helps with caching layers; keep stable and explicit
      Accept: "application/json",
    },
  });

  check(res, {
    "200": (r) => r.status === 200,
  });

  // Keep it tiny but non-zero to avoid a pure busy loop.
  sleep(Number(__ENV.SLEEP || 0.01));
}

