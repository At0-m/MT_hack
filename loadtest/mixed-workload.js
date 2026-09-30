import { optionsFor, setupFor, run, summaryFor, teardown as cleanup } from './lib/runner.js';
export const options = optionsFor('mixed');
export function setup() { return setupFor('mixed'); }
export default function (data) { run('mixed', data); }
export function teardown(data) { cleanup(data); }
export function handleSummary(data) { return summaryFor('mixed', data); }
