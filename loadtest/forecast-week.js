import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('week');
export function setup() { return setupFor('week'); }
export default function (data) { run('week', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('week', data); }
