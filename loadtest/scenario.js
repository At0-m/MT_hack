import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('scenario');
export function setup() { return setupFor('scenario'); }
export default function (data) { run('scenario', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('scenario', data); }
