import { describe, expect, it } from 'vitest';
import { parseFactors } from '../../src/features/scenario/factorValues';

describe('manual factor input', () => {
  it('accepts neutral factors and exact product boundaries', () => {
    expect(parseFactors({ weather: '1', event: '1', season: '1' })).toEqual({ weather: 1, event: 1, season: 1 });
    expect(parseFactors({ weather: '0.5', event: '0.5', season: '1' })).toBeDefined();
    expect(parseFactors({ weather: '2', event: '2', season: '1' })).toBeDefined();
  });
  it('rejects empty, nonfinite, overprecise and out-of-range input', () => {
    for (const weather of ['', ' ', 'NaN', 'Infinity', '0.49', '2.01', '1.0001']) {
      expect(parseFactors({ weather, event: '1', season: '1' })).toBeUndefined();
    }
    expect(parseFactors({ weather: '0.5', event: '0.5', season: '0.5' })).toBeUndefined();
    expect(parseFactors({ weather: '2', event: '2', season: '2' })).toBeUndefined();
  });
});
