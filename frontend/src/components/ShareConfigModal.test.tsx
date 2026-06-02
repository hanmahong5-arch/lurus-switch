import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// ---------------------------------------------------------------------------
// Standard mocks
// ---------------------------------------------------------------------------

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallbackOrOpts?: string | Record<string, unknown>) =>
      typeof fallbackOrOpts === 'string' ? fallbackOrOpts : key,
    i18n: { language: 'zh', changeLanguage: vi.fn() },
  }),
}))

// Mock qrcode.react so we don't need a canvas/SVG renderer in jsdom.
vi.mock('qrcode.react', () => ({
  QRCodeSVG: ({ value }: { value: string }) => (
    <svg data-testid="qr-svg" data-value={value} />
  ),
}))

const generateImportLinkMock = vi.fn()

vi.mock('../../wailsjs/go/main/App', () => ({
  GenerateImportLink: (...args: unknown[]) => generateImportLinkMock(...args),
}))

// Must import AFTER vi.mock() calls.
import { ShareConfigModal } from './ShareConfigModal'
import { useToastStore } from '../stores/toastStore'

const TEST_URL = 'switch://import?type=provider&data=eyJuYW1lIjoiVGVzdCJ9'

beforeEach(() => {
  vi.clearAllMocks()
  useToastStore.setState({ toasts: [] })
  generateImportLinkMock.mockResolvedValue(TEST_URL)
  // Mock navigator.clipboard
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
    configurable: true,
    writable: true,
  })
})

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

describe('ShareConfigModal', () => {
  it('calls GenerateImportLink with the supplied type and data', async () => {
    const data = { name: 'Test', baseUrl: 'https://example.com' }
    render(<ShareConfigModal type="provider" data={data} onClose={() => {}} />)

    await waitFor(() => {
      expect(generateImportLinkMock).toHaveBeenCalledWith('provider', JSON.stringify(data))
    })
  })

  it('renders the returned URL in the code element', async () => {
    render(
      <ShareConfigModal
        type="provider"
        data={{ name: 'Test', baseUrl: 'https://x.test' }}
        onClose={() => {}}
      />,
    )

    await waitFor(() => {
      expect(screen.getByTestId('share-url').textContent).toBe(TEST_URL)
    })
  })

  it('renders the QR code element with the URL value', async () => {
    render(
      <ShareConfigModal
        type="provider"
        data={{ name: 'Test', baseUrl: 'https://x.test' }}
        onClose={() => {}}
      />,
    )

    await waitFor(() => {
      const qr = screen.getByTestId('qr-svg')
      expect(qr).toBeTruthy()
      expect(qr.getAttribute('data-value')).toBe(TEST_URL)
    })
  })

  it('copy button calls navigator.clipboard.writeText with the URL', async () => {
    render(
      <ShareConfigModal
        type="provider"
        data={{ name: 'Test', baseUrl: 'https://x.test' }}
        onClose={() => {}}
      />,
    )

    await waitFor(() => screen.getByTestId('copy-btn'))
    fireEvent.click(screen.getByTestId('copy-btn'))

    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(TEST_URL)
    })
  })

  it('shows a success toast after copying', async () => {
    // Spy on addToast directly — the dedup window in toastStore can suppress
    // a second identical message within 3 s if a previous test already fired it.
    const addToastSpy = vi.spyOn(useToastStore.getState(), 'addToast')

    render(
      <ShareConfigModal
        type="provider"
        data={{ name: 'Test', baseUrl: 'https://x.test' }}
        onClose={() => {}}
      />,
    )

    await waitFor(() => screen.getByTestId('copy-btn'))
    fireEvent.click(screen.getByTestId('copy-btn'))

    await waitFor(() => {
      expect(addToastSpy).toHaveBeenCalledWith('success', expect.any(String))
    })

    addToastSpy.mockRestore()
  })

  it('shows an error message when GenerateImportLink rejects', async () => {
    generateImportLinkMock.mockRejectedValue(new Error('invalid type'))
    render(
      <ShareConfigModal
        type="badtype"
        data={{ name: 'Test' }}
        onClose={() => {}}
      />,
    )

    await waitFor(() => {
      expect(screen.getByText(/invalid type/i)).toBeTruthy()
    })

    // URL display and QR are absent on error.
    expect(screen.queryByTestId('share-url')).toBeNull()
    expect(screen.queryByTestId('qr-svg')).toBeNull()
  })

  it('calls onClose when the backdrop is clicked', async () => {
    const onClose = vi.fn()
    render(
      <ShareConfigModal
        type="provider"
        data={{ name: 'Test', baseUrl: 'https://x.test' }}
        onClose={onClose}
      />,
    )

    // Click the overlay backdrop (the fixed outer div).
    const backdrop = screen.getByRole('heading', { level: 2 }).closest('.fixed') as HTMLElement
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose when the Close button is clicked', async () => {
    const onClose = vi.fn()
    render(
      <ShareConfigModal
        type="provider"
        data={{ name: 'Test', baseUrl: 'https://x.test' }}
        onClose={onClose}
      />,
    )

    // Footer close button
    const closeBtns = screen.getAllByRole('button').filter(
      (b) => b.textContent?.includes('关闭') || b.getAttribute('aria-label') === '关闭',
    )
    fireEvent.click(closeBtns[0])
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
