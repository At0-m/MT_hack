import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('day');
export function setup() { return setupFor('day'); }
export default function (data) { run('day', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('day', data); }
