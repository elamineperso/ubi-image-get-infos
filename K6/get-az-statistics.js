import http from 'k6/http';

import { check } from 'k6';
import { Counter } from 'k6/metrics';

export const options = {
  vus: 10,
  duration: '1m',
};

const azResponses = new Counter('az_responses');

export default function () {
  const res = http.get('http://51.68.127.91:31570/api/az');

  check(res, {
    'status is 200': (r) => r.status === 200,
    'body not empty': (r) => r.body && r.body.length > 0,
    'is json': (r) =>
      r.headers['Content-Type'] &&
      r.headers['Content-Type'].includes('application/json'),
  });

  // Only parse if safe
  if (
    res.status === 200 &&
    res.body &&
    res.body.length > 0 &&
    res.headers['Content-Type'] &&
    res.headers['Content-Type'].includes('application/json')
  ) {
    const body = JSON.parse(res.body);

    if (body.az) {
      azResponses.add(1, { az: body.az });
    }
  }
}
