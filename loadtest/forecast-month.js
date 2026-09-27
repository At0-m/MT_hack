import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('month');
export function setup() { return setupFor('month'); }
export default function (data) { run('month', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('month', data); }
