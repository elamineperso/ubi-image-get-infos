import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

export const options = {
  vus: 10,
  duration: '1m',

  // This makes per-AZ results visible in summary
  thresholds: {
    'az_responses{az:AZ1}': ['count>=0'],
    'az_responses{az:AZ2}': ['count>=0'],
    'az_responses{az:AZ3}': ['count>=0'],
  },
};

const azResponses = new Counter('az_responses');

export default function () {
  const res = http.get('http://51.68.127.91:31570/api/az');

  const ok = check(res, {
    'status is 200': (r) => r.status === 200,
    'is json': (r) =>
      r.headers['Content-Type'] &&
      r.headers['Content-Type'].includes('application/json'),
  });

  if (ok) {
    const body = res.json(); // safer & cleaner than JSON.parse
    if (body.az) {
      azResponses.add(1, { az: body.az });
    }
  }
}

