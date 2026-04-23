import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    baseline: {
      executor: "constant-arrival-rate",
      rate: 40,
      timeUnit: "1s",
      duration: "60s",
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
    flash_sale_10x: {
      executor: "constant-arrival-rate",
      startTime: "70s",
      rate: 400,
      timeUnit: "1s",
      duration: "120s",
      preAllocatedVUs: 200,
      maxVUs: 1000,
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<3000"],
    checks: ["rate>0.99"],
  },
};

const url = "http://localhost:8080/api/v1/chat";

export default function () {
  const payload = JSON.stringify({
    session_id: `s-${__VU}-${__ITER}`,
    user_id: `u-${__VU}`,
    message: "Please provide price and discount for EV-Pro",
  });

  const params = {
    headers: {
      "Content-Type": "application/json",
    },
  };

  const res = http.post(url, payload, params);
  check(res, {
    "status is 200": (r) => r.status === 200,
    "intent exists": (r) => r.json("intent") !== "",
  });
  sleep(0.1);
}
