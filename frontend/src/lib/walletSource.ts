// walletSource.ts — mode-aware wallet data source.
//
// Reseller and Personal modes are the only callers today:
//   - Hub mode: Wails bindings forward to Hub /api/wallet/{info,transactions}
//   - Local mode: throws (Personal mode users hit Lurus Cloud and don't have
//     a dedicated wallet surface; their billing lives on identity.lurus.cn).
//
// The page uses Result-shape returns so a Wails round-trip failure that
// happens to leak past the binding still surfaces as an in-page error
// banner — never a silent success — per the lesson logged at
// [[feedback_wails_result_success]].

import {
  HubGetWalletInfo,
  HubListWalletTransactions,
} from '../../wailsjs/go/main/App'
import type { admin } from '../../wailsjs/go/models'

export interface WalletSource {
  readonly kind: 'local' | 'hub'
  getInfo(): Promise<admin.WalletInfo>
  listTransactions(page: number, pageSize: number): Promise<admin.WalletTransactionPage>
}

class HubWalletSource implements WalletSource {
  readonly kind = 'hub' as const

  async getInfo() {
    return HubGetWalletInfo()
  }

  async listTransactions(page: number, pageSize: number) {
    return HubListWalletTransactions({ page, page_size: pageSize } as admin.WalletQuery)
  }
}

class LocalWalletSource implements WalletSource {
  readonly kind = 'local' as const

  async getInfo(): Promise<admin.WalletInfo> {
    throw new Error('wallet.notSupportedInPersonal')
  }

  async listTransactions(): Promise<admin.WalletTransactionPage> {
    throw new Error('wallet.notSupportedInPersonal')
  }
}

export function makeWalletSource(mode: 'local' | 'hub'): WalletSource {
  return mode === 'hub' ? new HubWalletSource() : new LocalWalletSource()
}
