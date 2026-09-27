import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('export');
export function setup() { return setupFor('export'); }
export default function (data) { run('export', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('export', data); }
