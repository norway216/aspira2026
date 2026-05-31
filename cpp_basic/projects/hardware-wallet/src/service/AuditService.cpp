#include "AuditService.h"
#include "persistence/Database.h"
#include "persistence/AuditLogRepository.h"

#include <QDebug>

AuditService::AuditService(Database* db, QObject* parent)
    : QObject(parent), m_db(db) {
    m_auditRepo = new AuditLogRepository(db);
}

void AuditService::getEntries(int offset, int limit) {
    auto result = m_auditRepo->findAll(offset, limit);
    if (result.isFail()) {
        emit entriesLoaded(Result<std::vector<AuditLogEntry>>::fail(
            result.error().code, result.error().message));
        return;
    }
    emit entriesLoaded(Result<std::vector<AuditLogEntry>>::ok(std::move(result.value())));
}

void AuditService::verifyIntegrity() {
    auto result = m_auditRepo->verifyChain();
    if (result.isFail()) {
        emit integrityVerified(Result<bool>::fail(result.error().code, result.error().message));
        return;
    }

    bool intact = result.value();
    if (intact) {
        qDebug() << "Audit log integrity: VERIFIED";
    } else {
        qWarning() << "Audit log integrity: FAILED - chain is broken!";
    }

    emit integrityVerified(Result<bool>::ok(intact));
}

void AuditService::getEntryCount() {
    auto result = m_auditRepo->count();
    if (result.isFail()) {
        emit entryCountLoaded(Result<int>::fail(result.error().code, result.error().message));
        return;
    }
    emit entryCountLoaded(Result<int>::ok(result.value()));
}

void AuditService::logEvent(const QString& action, const QString& userId,
                            const QString& details) {
    m_auditRepo->append(action, userId, details);
}
