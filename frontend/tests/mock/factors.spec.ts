import { expect, test } from '@playwright/test';
import type { Calculation } from '../../src/api/types';

test('factors automatically refresh, reject invalid drafts and reset to baseline', async ({ page }) => {
  await page.goto('/');
  await page.getByLabel('Логин', { exact: true }).fill('demo');
  await page.getByLabel('Пароль', { exact: true }).fill('demo');
  await page.getByRole('button', { name: 'Войти', exact: true }).click();
  await page.getByRole('button', { name: 'Открыть панель инструментов', exact: true }).click();
  await expect(page.getByRole('button', { name: 'ПОПРАВКИ', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'ПОПРАВКИ', exact: true }).click();
  const requests: unknown[] = [];
  page.on('request', request => {
    if (request.url().endsWith('/scenarios/evaluate')) requests.push(request.postDataJSON());
  });
  const response = page.waitForResponse(r => r.url().endsWith('/scenarios/evaluate') && r.status() === 200);
  await page.getByLabel('Погода', { exact: true }).fill('1.2');
  const calculation = await (await response).json() as Calculation;
  expect(calculation.descriptor.kind).toBe('scenario');
  for (const frame of calculation.frames) {
    for (const route of frame.routes) {
      expect(route.evaluated.boardings).toBeCloseTo(route.baseline.boardings * 1.2, 6);
    }
  }
  await expect(page.locator('.status-message')).toHaveCount(0);
  await page.getByLabel('Погода', { exact: true }).fill('');
  await expect(page.getByRole('alert')).toBeVisible();
  const before = requests.length;
  await page.waitForTimeout(600);
  expect(requests.length).toBe(before);

  const latest = page.waitForResponse(r => r.url().endsWith('/scenarios/evaluate') && r.status() === 200);
  await page.getByLabel('Погода', { exact: true }).fill('1.3');
  await page.getByLabel('Погода', { exact: true }).fill('1.4');
  const changed = await (await latest).json() as Calculation;
  expect(changed.descriptor.kind).toBe('scenario');
  if (changed.descriptor.kind === 'scenario') expect(changed.descriptor.overrides.factors?.weather).toBe(1.4);
  await expect(page.locator('.status-message')).toHaveCount(0);
  await page.waitForTimeout(600);
  expect(requests.length).toBe(before + 1);

  const baseline = page.waitForResponse(r => r.url().endsWith('/forecasts/query') && r.status() === 200);
  await page.getByRole('button', { name: 'Исходный прогноз', exact: true }).click();
  const reset = await (await baseline).json() as Calculation;
  expect(reset.descriptor.kind).toBe('forecast');
  for (const frame of reset.frames) {
    for (const route of frame.routes) expect(route.evaluated.boardings).toBe(route.baseline.boardings);
  }
});
