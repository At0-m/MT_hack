import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('summary');
export function setup() { return setupFor('summary'); }
export default function (data) { run('summary', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('summary', data); }
