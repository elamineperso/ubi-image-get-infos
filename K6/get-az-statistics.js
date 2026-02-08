import http from 'k6/http';
import { check, Counter } from 'k6/metrics';

export const options = {
  vus: 10,
  duration: '1m',
};

// Custom counter tagged by AZ
const azResponses = new Counter('az_responses');

export default function () {
  const res = http.get('https://my-api.example.com/api/az');

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  // Parse JSON response
  const body = res.json();

  if (body && body.az) {
    // Increment counter with AZ tag
    azResponses.add(1, { az: body.az });
  }
}
