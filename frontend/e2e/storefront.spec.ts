import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'

test('catalog filters products and passes the accessibility audit', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /Poucas unidades/ })).toBeVisible()
  await expect(page.locator('.product')).toHaveCount(8)
  await expect(page.locator('.product-art img')).toHaveCount(8)

  const brokenImages = await page.locator('.product-art img').evaluateAll((images) =>
    images.filter((image) => !(image instanceof HTMLImageElement) || image.naturalWidth === 0).length,
  )
  expect(brokenImages).toBe(0)

  await page.getByRole('button', { name: 'Áudio' }).click()
  await expect(page.locator('.product')).toHaveCount(2)
  await page.getByRole('button', { name: 'Tudo' }).click()
  await expect(page.locator('.product')).toHaveCount(8)

  const accessibility = await new AxeBuilder({ page }).analyze()
  expect(accessibility.violations).toEqual([])
})

test('customer can reserve, pay, and ask about an order', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('link', { name: /Fone Nuvem/ }).click()
  await expect(page.getByRole('heading', { name: 'Fone Nuvem' })).toBeVisible()
  await page.getByRole('button', { name: /Reservar por 5 minutos/ }).click()

  await expect(page).toHaveURL(/\/orders\/[0-9a-f-]+$/i)
  await expect(page.getByRole('heading', { name: 'Seu pedido está reservado' })).toBeVisible()
  await page.getByRole('button', { name: /Simular pagamento aprovado/ }).click()
  await expect(page.getByRole('heading', { name: 'Pedido confirmado' })).toBeVisible()
  await expect(page.getByText('Pagamento confirmado')).toBeVisible()

  await page.getByLabel('Sua pergunta').fill('Meu pagamento foi confirmado?')
  await page.getByRole('button', { name: 'Perguntar' }).click()
  await expect(page.locator('.assistant-answer')).toContainText('STATUS PAID', { timeout: 15_000 })
})
