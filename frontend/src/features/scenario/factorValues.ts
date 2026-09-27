export const factorKeys = ['weather', 'event', 'season'] as const;
export type FactorKey = typeof factorKeys[number];
export type FactorDraft = Record<FactorKey, string>;

export function parseFactors(draft: FactorDraft): Record<FactorKey, number> | undefined {
  const values = {} as Record<FactorKey, number>;
  for (const key of factorKeys) {
    if (!draft[key].trim()) return undefined;
    const value = Number(draft[key]);
    if (!Number.isFinite(value) || value < 0.5 || value > 2 ||
        Math.abs(value * 1000 - Math.round(value * 1000)) >= 1e-8) return undefined;
    values[key] = value;
  }
  const product = values.weather * values.event * values.season;
  return product >= 0.25 && product <= 4 ? values : undefined;
}
