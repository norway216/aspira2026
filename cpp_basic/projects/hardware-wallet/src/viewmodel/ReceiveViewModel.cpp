#include "ReceiveViewModel.h"
#include "service/WalletService.h"

ReceiveViewModel::ReceiveViewModel(WalletService* walletService, QObject* parent)
    : QObject(parent), m_walletService(walletService) {

    connect(m_walletService, &WalletService::walletLoaded, this,
        [this](Result<Wallet> result) {
            if (result.isOk()) {
                m_publicKey = result.value().publicKeyHex();
                emit publicKeyChanged();
                emit qrCodeDataChanged();
            }
        });
}

void ReceiveViewModel::refresh() {
    m_walletService->getWallet();
}
