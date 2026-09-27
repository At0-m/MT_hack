import { test, expect, type Page } from '@playwright/test';
import type { Calculation } from '../../src/api/types';

async function login(page: Page) {
 const password = process.env.API_PASSWORD;
 if (!password) throw new Error('Set API_USER/API_PASSWORD from your private .env; no hardcoded credentials.');
 await page.goto('/');
 await page.getByLabel('\u041b\u043e\u0433\u0438\u043d', {exact:true}).fill(process.env.API_USER || 'devops');
 await page.getByLabel('\u041f\u0430\u0440\u043e\u043b\u044c', {exact:true}).fill(password);
 await page.getByRole('button', {name:'\u0412\u043e\u0439\u0442\u0438',exact:true}).click();
 await expect(page.getByRole('button', {name:'\u0412\u044b\u0431\u043e\u0440 \u0432\u0440\u0435\u043c\u0435\u043d\u043d\u043e\u0433\u043e \u043a\u0430\u0434\u0440\u0430'})).toBeVisible();
 await expect(page.locator('.status-message')).toHaveCount(0);
}

test('real auth survives reload; wheel is local; fleet, summary and CSV share server data', async ({page})=>{
 const exceptions: string[]=[];page.on('pageerror',e=>exceptions.push(e.message));
 await login(page);
 await page.reload();
 const wheelButton=page.getByRole('button',{name:'\u0412\u044b\u0431\u043e\u0440 \u0432\u0440\u0435\u043c\u0435\u043d\u043d\u043e\u0433\u043e \u043a\u0430\u0434\u0440\u0430'});
 await expect(wheelButton).toBeVisible();
 await expect(page.locator('.status-message')).toHaveCount(0);
 const queries: string[]=[];page.on('request',r=>{if(/\/(forecasts\/query|scenarios\/evaluate)$/.test(r.url()))queries.push(r.url());});
 await wheelButton.click();
 const slider=page.getByRole('slider');await slider.press('ArrowDown');await expect(slider).toHaveAttribute('aria-valuenow','1');
 await page.waitForTimeout(300);expect(queries).toHaveLength(0);
 await page.keyboard.press('Escape');
 await page.getByRole('button',{name:'\u0423\u043f\u0440\u0430\u0432\u043b\u0435\u043d\u0438\u0435 \u0432\u044b\u043f\u0443\u0441\u043a\u043e\u043c'}).click();
 const resultPromise=page.waitForResponse(r=>r.url().endsWith('/scenarios/evaluate')&&r.status()===200);
 await page.getByLabel('\u041a\u043e\u043b\u0438\u0447\u0435\u0441\u0442\u0432\u043e \u0442\u0440\u0430\u043c\u0432\u0430\u0435\u0432',{exact:true}).fill('12');
 const result=await (await resultPromise).json() as Calculation;
 expect(result.descriptor.kind).toBe('scenario');
 for(const frame of result.frames)for(const route of frame.routes)for(const reading of [route,...route.stop_readings])expect(reading.evaluated.boardings).toBe(reading.baseline.boardings);
 await expect(page.locator('.status-message')).toHaveCount(0);
 const summaryPromise=page.waitForResponse(r=>r.url().endsWith('/summaries/query')&&r.status()===200);
 await page.getByRole('button',{name:'\u041e\u0431\u0449\u0430\u044f \u0430\u043d\u0430\u043b\u0438\u0442\u0438\u043a\u0430'}).click();
 const summary=await (await summaryPromise).json();expect(summary.calculation_id).toBe(result.calculation_id);
 await expect(page.locator('.server-summary')).toContainText(summary.text);
 await page.getByRole('button',{name:'\u041e\u0442\u043a\u0440\u044b\u0442\u044c \u043f\u0430\u043d\u0435\u043b\u044c \u0438\u043d\u0441\u0442\u0440\u0443\u043c\u0435\u043d\u0442\u043e\u0432'}).click();
 const exportPromise=page.waitForResponse(r=>r.url().endsWith('/exports'));
 const downloadPromise=page.waitForEvent('download');
 await page.getByRole('button',{name:/CSV/}).click();
 const exported=await exportPromise;expect(exported.status()).toBe(200);expect(exported.headers()['x-calculation-id']).toBe(result.calculation_id);
 const download=await downloadPromise;expect(await download.failure()).toBeNull();
 await page.screenshot({path:'test-results/integrated-dashboard.png',fullPage:true});
 expect(exceptions).toEqual([]);
 await page.getByRole('button',{name:'\u0412\u042b\u0419\u0422\u0418',exact:true}).click();
 await expect(page.getByLabel('\u041b\u043e\u0433\u0438\u043d',{exact:true})).toBeVisible();
 await page.reload();await expect(page.getByLabel('\u041b\u043e\u0433\u0438\u043d',{exact:true})).toBeVisible();
});

test('month uses calendar boundaries and real daily stop arrays',async({page})=>{
 await login(page);
 await page.getByRole('button',{name:'\u0412\u044b\u0431\u043e\u0440 \u043f\u0435\u0440\u0438\u043e\u0434\u0430'}).click();
 const next=page.waitForResponse(r=>r.url().endsWith('/forecasts/query')&&r.request().postDataJSON()?.selection?.view_mode==='month');
 await page.getByLabel('\u0420\u0435\u0436\u0438\u043c \u043f\u0435\u0440\u0438\u043e\u0434\u0430').selectOption('month');
 const response=await next;expect(response.status()).toBe(200);const calculation=await response.json() as Calculation;
 expect(calculation.frames.length).toBeGreaterThanOrEqual(28);expect(calculation.frames.length).toBeLessThanOrEqual(31);
 const start=new Date(Date.parse(calculation.descriptor.selection.window.from)+3*3600000);expect(start.getUTCDate()).toBe(1);
 await expect(page.locator('.status-message')).toHaveCount(0);
 // Capability-aware: real route-only models must not fabricate stop data.
 for(const frame of calculation.frames)for(const route of frame.routes){if(calculation.descriptor.selection.spatial_detail==='route_stop')expect(route.stop_readings.length).toBeGreaterThan(0);else expect(route.stop_readings).toHaveLength(0);}
});
