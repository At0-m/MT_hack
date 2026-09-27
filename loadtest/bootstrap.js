import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('bootstrap');
export function setup() { return setupFor('bootstrap'); }
export default function (data) { run('bootstrap', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('bootstrap', data); }
