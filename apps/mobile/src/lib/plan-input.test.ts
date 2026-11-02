import test from 'node:test';
import assert from 'node:assert/strict';
import { planInput } from './plan-input.ts';
const valid = { title: ' Cà phê ', activity: 'coffee', date: '2030-01-01', time: '19:30', capacity: '', location: '', group_id: '' };
test('plan form converts explicit Vietnam time and trims input', () => {
  const data = planInput(valid, new Date('2029-01-01'));
  assert.equal(data.title, 'Cà phê'); assert.equal(data.starts_at, '2030-01-01T12:30:00.000Z'); assert.equal(data.capacity, 0);
});
test('invalid calendar, past time, fractional capacity and unknown activity are rejected', () => {
  for (const changes of [{ date: '2030-02-30' }, { time: '25:00' }, { capacity: '2.5' }, { capacity: '-1' }, { activity: 'fake' }, { date: '2020-01-01' }, { title: ' ' }]) assert.throws(() => planInput({ ...valid, ...changes }, new Date('2029-01-01')));
});
