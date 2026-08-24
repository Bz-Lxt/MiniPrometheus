import { test, expect } from '@playwright/test'

const base = process.env.E2E_BASE || 'http://localhost:31871'

test.describe('MiniPrometheus 关键路径', () => {
  test('大屏渲染且不出现静默残缺', async ({ page }) => {
    await page.goto(base + '/')
    await expect(page.getByText('多维指标流')).toBeVisible()
    await expect(page.locator('canvas').first()).toBeVisible()
  })

  test('查询剖析器能画出执行树', async ({ page }) => {
    await page.goto(base + '/query')
    await page.getByRole('button', { name: /执行/ }).click()
    await expect(page.getByText('执行树')).toBeVisible()
    await expect(page.locator('canvas').nth(0)).toBeVisible()
  })

  test('集群页展示分片', async ({ page }) => {
    await page.goto(base + '/cluster')
    await expect(page.getByText('集群拓扑')).toBeVisible()
  })
})
