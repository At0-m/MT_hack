import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('geometry');
export function setup() { return setupFor('geometry'); }
export default function (data) { run('geometry', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('geometry', data); }
