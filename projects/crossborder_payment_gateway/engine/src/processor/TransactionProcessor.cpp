#include "processor/TransactionProcessor.h"
#include <sstream>

namespace payment_engine {

TransactionProcessor::TransactionProcessor(ValidationEngine* validator,
                                           ExchangeEngine* exchanger,
                                           AccountStore* accounts,
                                           TransactionStore* txn_store,
                                           HashChain* hash_chain)
    : validator_(validator)
    , exchanger_(exchanger)
    , accounts_(accounts)
    , txn_store_(txn_store)
    , hash_chain_(hash_chain)
    , tps_window_start_(Clock::now())
{
}

TransactionResult TransactionProcessor::Process(const TransactionRequest& req) {
    TransactionResult result;
    result.transaction_id = req.transaction_id;
    result.fee = req.fee;

    // 1. Validate
    auto validation_err = validator_->Validate(req);
    if (validation_err.has_value()) {
        result.status = TxnStatus::Failed;
        result.processed_at = CurrentTimestampISO();
        return result;
    }

    // 2. Look up exchange rate
    auto rate = exchanger_->GetRate(req.source_currency, req.target_currency);
    if (!rate.has_value()) {
        result.status = TxnStatus::Failed;
        result.processed_at = CurrentTimestampISO();
        return result;
    }
    result.exchange_rate = rate->rate;

    // 3. Calculate target amount: (source_amount - fee) * exchange_rate
    int64_t net_amount = req.source_amount - req.fee;
    if (net_amount < 0) net_amount = 0;
    result.target_amount = static_cast<int64_t>(net_amount * rate->rate);
    result.status = TxnStatus::Completed;
    result.processed_at = CurrentTimestampISO();

    // 4. Generate hash chain entry
    int64_t ts = CurrentTimestampNS();
    std::string hash = hash_chain_->Append(
        req.transaction_id, req.source_amount,
        req.target_currency, result.target_amount, ts);
    result.hash_chain_current = hash;

    // 5. Reserve funds from source account (deduct source_amount)
    {
        auto payer = accounts_->GetAccount(req.payer_account_id);
        if (payer) {
            int64_t cur_bal = payer->get().balance.load(std::memory_order_acquire);
            int64_t cur_res = payer->get().reserved_balance.load(std::memory_order_acquire);
            int64_t new_bal = cur_bal - req.source_amount - req.fee;
            int64_t new_res = cur_res + req.source_amount + req.fee; // reserved for settlement
            accounts_->UpdateBalance(req.payer_account_id, new_bal, new_res);
        }
    }

    // 6. Add funds to target account
    {
        auto payee = accounts_->GetAccount(req.payee_account_id);
        if (payee) {
            int64_t cur_bal = payee->get().balance.load(std::memory_order_acquire);
            accounts_->UpdateBalance(req.payee_account_id,
                                     cur_bal + result.target_amount,
                                     payee->get().reserved_balance.load(std::memory_order_acquire));
        }
    }

    // 7. Store transaction record
    TransactionStore::TxnRecord record;
    record.id = req.transaction_id;
    record.merchant_id = req.merchant_id;
    record.source_currency = req.source_currency;
    record.target_currency = req.target_currency;
    record.source_amount = req.source_amount;
    record.target_amount = result.target_amount;
    record.fee = req.fee;
    record.exchange_rate = result.exchange_rate;
    record.status = result.status;
    record.hash_chain_prev = hash_chain_->GetCurrentHash(); // Will get the proper prev
    record.hash_chain_curr = result.hash_chain_current;
    record.timestamp_ns = ts;
    txn_store_->Append(record);

    // 8. Update TPS
    txn_counter_.fetch_add(1, std::memory_order_release);
    tps_window_count_.fetch_add(1, std::memory_order_release);
    UpdateTPS();

    return result;
}

void TransactionProcessor::UpdateTPS() {
    auto now = Clock::now();
    auto elapsed = std::chrono::duration_cast<std::chrono::seconds>(
                       now - tps_window_start_).count();
    if (elapsed >= 1) {
        int64_t count = tps_window_count_.exchange(0, std::memory_order_acq_rel);
        tps_.store(count / std::max(elapsed, int64_t(1)), std::memory_order_release);
        tps_window_start_ = now;
    }
}

EngineResponse TransactionProcessor::HandleRequest(const EngineRequest& req) {
    EngineResponse resp;
    resp.id = req.id;
    resp.session_fd = req.session_fd;

    // Parse the transaction request from the JSON payload
    TransactionRequest txn_req = SimpleJson::ParseTransactionRequest(req.payload_json);

    // Override transaction_id if provided in outer envelope
    if (!req.id.empty() && txn_req.transaction_id.empty()) {
        txn_req.transaction_id = req.id;
    }

    // Process
    TransactionResult result = Process(txn_req);

    if (result.status == TxnStatus::Completed) {
        resp.status = "ok";
        resp.payload_json = SimpleJson::SerializeTransactionResult(result);
        resp.error = std::nullopt;
    } else {
        resp.status = "error";
        resp.payload_json = "{}";
        resp.error = EngineError{
            ErrorCode::InternalError,
            "Transaction processing failed",
            true
        };
    }

    return resp;
}

std::string TransactionProcessor::GetHealthInfo() const {
    SimpleJson doc;
    std::vector<std::string> fields;
    fields.push_back(SimpleJson::BuildStringField("status", "healthy"));
    fields.push_back(SimpleJson::BuildIntField("total_transactions",
                       txn_counter_.load(std::memory_order_acquire)));
    fields.push_back(SimpleJson::BuildIntField("tps",
                       tps_.load(std::memory_order_acquire)));
    return SimpleJson::ToObject(fields);
}

} // namespace payment_engine
